"""Tests for generate-config.py and custom_auth.py."""

import importlib.util
import sys
from pathlib import Path

import pytest

# Load generate-config.py as a module (hyphen in filename requires importlib)
_spec = importlib.util.spec_from_file_location(
    "generate_config",
    Path(__file__).parent / "generate-config.py",
)
generate_config = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(generate_config)

parse_duration_to_seconds = generate_config.parse_duration_to_seconds
build_litellm_params = generate_config.build_litellm_params
deep_merge = generate_config.deep_merge
load_extra_config = generate_config.load_extra_config
apply_operator_settings = generate_config.apply_operator_settings

sys.path.insert(0, str(Path(__file__).parent))
import custom_auth  # noqa: E402


class TestParseDurationToSeconds:
    def test_seconds(self):
        assert parse_duration_to_seconds("30s") == 30.0

    def test_minutes(self):
        assert parse_duration_to_seconds("5m") == 300.0

    def test_hours(self):
        assert parse_duration_to_seconds("2h") == 7200.0

    def test_milliseconds(self):
        assert parse_duration_to_seconds("500ms") == pytest.approx(0.5)

    def test_nanoseconds(self):
        assert parse_duration_to_seconds("1000000000ns") == pytest.approx(1.0)

    def test_microseconds_us(self):
        assert parse_duration_to_seconds("1000000us") == pytest.approx(1.0)

    def test_microseconds_unicode(self):
        assert parse_duration_to_seconds("1000000µs") == pytest.approx(1.0)

    def test_unknown_suffix_returns_default(self):
        assert parse_duration_to_seconds("10x") == 300.0

    def test_ms_does_not_match_s_branch(self):
        # 500ms should NOT be parsed as int("500m") seconds — that would ValueError
        result = parse_duration_to_seconds("500ms")
        assert result == pytest.approx(0.5)

    def test_ns_does_not_match_s_branch(self):
        result = parse_duration_to_seconds("100ns")
        assert result == pytest.approx(1e-7)

    def test_us_does_not_match_s_branch(self):
        result = parse_duration_to_seconds("50us")
        assert result == pytest.approx(5e-5)

    def test_fractional_seconds(self):
        assert parse_duration_to_seconds("1s") == 1.0

    def test_large_hours(self):
        assert parse_duration_to_seconds("24h") == pytest.approx(86400.0)


class TestBuildLitellmParamsTimeout:
    """Verify build_litellm_params passes timeout correctly for every suffix."""

    BASE_SPEC = {
        "provider": "openai",
        "modelName": "gpt-4",
    }

    def _params(self, timeout: str):
        spec = {**self.BASE_SPEC, "timeout": timeout}
        return build_litellm_params(spec, api_key=None)

    def test_timeout_s(self):
        assert self._params("30s")["timeout"] == 30.0

    def test_timeout_m(self):
        assert self._params("5m")["timeout"] == 300.0

    def test_timeout_h(self):
        assert self._params("1h")["timeout"] == 3600.0

    def test_timeout_ms(self):
        assert self._params("500ms")["timeout"] == pytest.approx(0.5)

    def test_timeout_ns(self):
        assert self._params("1000000000ns")["timeout"] == pytest.approx(1.0)

    def test_timeout_us(self):
        assert self._params("1000000us")["timeout"] == pytest.approx(1.0)

    def test_timeout_unicode_us(self):
        assert self._params("1000000µs")["timeout"] == pytest.approx(1.0)

    def test_no_timeout_omitted(self):
        spec = {**self.BASE_SPEC}
        params = build_litellm_params(spec, api_key=None)
        assert "timeout" not in params


class TestOperatorSettings:
    def base(self):
        return {
            "model_list": [{"model_name": "gpt-4"}],
            "litellm_settings": {"drop_params": True},
            "general_settings": {"background_health_checks": False},
        }

    def test_unset_or_blank_extra_config_changes_nothing(self):
        assert apply_operator_settings(self.base(), env={}) == self.base()
        assert apply_operator_settings(self.base(), env={"LANGOP_GATEWAY_EXTRA_CONFIG": "  \n"}) == self.base()

    def test_extra_config_deep_merges_mappings_and_replaces_lists(self):
        extra = """
        litellm_settings:
          success_callback: ["generic"]
          failure_callback: ["generic"]
        general_settings:
          master_key: sk-placeholder
        """
        merged = apply_operator_settings(self.base(), env={"LANGOP_GATEWAY_EXTRA_CONFIG": extra})
        assert merged["litellm_settings"] == {
            "drop_params": True,
            "success_callback": ["generic"],
            "failure_callback": ["generic"],
        }
        assert merged["general_settings"] == {"background_health_checks": False, "master_key": "sk-placeholder"}
        assert merged["model_list"] == [{"model_name": "gpt-4"}]

        replaced = deep_merge({"a": [1, 2], "b": {"c": 1}}, {"a": [3], "b": {"d": 2}})
        assert replaced == {"a": [3], "b": {"c": 1, "d": 2}}

    def test_invalid_extra_config_exits(self):
        with pytest.raises(SystemExit):
            load_extra_config({"LANGOP_GATEWAY_EXTRA_CONFIG": "- not\n- a mapping"})
        with pytest.raises(SystemExit):
            load_extra_config({"LANGOP_GATEWAY_EXTRA_CONFIG": "key: [unclosed"})

    def test_hmac_secret_wires_custom_auth_without_overriding_an_explicit_one(self):
        wired = apply_operator_settings(self.base(), env={"LANGOP_GATEWAY_HMAC_SECRET": "s3cret"})
        assert wired["general_settings"]["custom_auth"] == "custom_auth.user_api_key_auth"

        explicit = apply_operator_settings(
            self.base(),
            env={
                "LANGOP_GATEWAY_HMAC_SECRET": "s3cret",
                "LANGOP_GATEWAY_EXTRA_CONFIG": "general_settings: {custom_auth: mine.auth}",
            },
        )
        assert explicit["general_settings"]["custom_auth"] == "mine.auth"


class TestCustomAuth:
    def test_keys_round_trip_and_reject_tampering(self):
        key = custom_auth.agent_key("s3cret", "9f8221e3-a8ce-47c0-9e31-4afeb62657df")
        assert key.startswith("sk-langop-9f8221e3-a8ce-47c0-9e31-4afeb62657df.")
        assert len(key.rsplit(".", 1)[1]) == 32
        assert custom_auth.verify("s3cret", key) == "9f8221e3-a8ce-47c0-9e31-4afeb62657df"
        assert custom_auth.verify("other", key) is None
        assert custom_auth.verify("s3cret", key[:-1] + "0") is None
        assert custom_auth.verify("s3cret", "sk-langop-no-signature") is None
        assert custom_auth.verify("s3cret", "sk-something-else") is None
        assert custom_auth.verify("s3cret", None) is None

    def test_signature_is_bound_to_the_agent_id(self):
        key = custom_auth.agent_key("s3cret", "agent-a")
        forged = key.replace("agent-a", "agent-b")
        assert custom_auth.verify("s3cret", forged) is None


class TestLitellmSettings:
    build_litellm_settings = staticmethod(generate_config.build_litellm_settings)

    def test_settings_depend_on_the_provider_set_not_the_order(self):
        # Process-wide settings: one openai-compatible model loosens validation
        # for the whole gateway, whichever spec happens to be read first.
        local_first = self.build_litellm_settings(["openai-compatible", "anthropic"])
        local_last = self.build_litellm_settings(["anthropic", "openai-compatible"])
        assert local_first == local_last
        assert local_first["disable_strict_validation"] is True
        assert local_first["drop_params"] is True

    def test_hosted_providers_only_get_drop_params(self):
        assert self.build_litellm_settings(["anthropic", "openai"]) == {"drop_params": True}

    def test_no_models_means_no_litellm_settings(self, monkeypatch):
        monkeypatch.delenv("LANGOP_GATEWAY_EXTRA_CONFIG", raising=False)
        monkeypatch.delenv("LANGOP_GATEWAY_HMAC_SECRET", raising=False)
        assert "litellm_settings" not in generate_config.generate_litellm_config([])


class TestSharedModelNames:
    def test_shared_model_name_is_reported_as_load_balancing(self, capsys):
        generate_config.warn_shared_model_names([
            {"provider": "openai-compatible", "modelName": "llama3.2"},
            {"provider": "openai-compatible", "modelName": "llama3.2"},
            {"provider": "anthropic", "modelName": "claude-sonnet-4-5"},
        ])
        err = capsys.readouterr().err
        assert "2 LanguageModels share modelName 'llama3.2'" in err
        assert "load-balanced" in err
        assert "claude-sonnet-4-5" not in err

    def test_unique_model_names_are_quiet(self, capsys):
        generate_config.warn_shared_model_names([{"modelName": "a"}, {"modelName": "b"}])
        assert capsys.readouterr().err == ""


class TestCustomAuthHook:
    """The hook as LiteLLM calls it. Needs LiteLLM (and FastAPI) installed."""

    @pytest.fixture(autouse=True)
    def _litellm(self, monkeypatch):
        pytest.importorskip("litellm")
        monkeypatch.setenv("LANGOP_GATEWAY_HMAC_SECRET", "s3cret")
        monkeypatch.delenv("LITELLM_MASTER_KEY", raising=False)

    def call(self, key):
        import asyncio
        return asyncio.run(custom_auth.user_api_key_auth(None, key))

    def test_bad_key_is_a_401_not_a_server_error(self):
        # A plain exception becomes a 500 in LiteLLM >= 1.103 (see custom_auth.py).
        from fastapi import HTTPException
        with pytest.raises(HTTPException) as exc:
            self.call("Bearer sk-langop-agent.bad")
        assert exc.value.status_code == 401

    def test_agent_key_authenticates_as_that_agent(self):
        auth = self.call("Bearer " + custom_auth.agent_key("s3cret", "agent-a"))
        assert auth.user_id == "agent-a"
        assert auth.metadata == {"langop_agent_id": "agent-a"}


class TestProviderMapping:
    map_provider = staticmethod(generate_config.map_provider_to_litellm)

    @pytest.mark.parametrize("provider,expected", [
        ("openai", "m"),
        ("anthropic", "anthropic/m"),
        ("gemini", "gemini/m"),
        ("azure", "azure/m"),
        ("bedrock", "bedrock/m"),
        ("vertex", "vertex_ai/m"),
        ("openai-compatible", "openai/m"),
        ("custom", "openai/m"),
    ])
    def test_provider_prefix(self, provider, expected):
        assert self.map_provider(provider, "m") == expected

    def test_litellm_provider_is_the_prefix(self):
        assert self.map_provider(None, "deepseek-chat", "deepseek") == "deepseek/deepseek-chat"
        assert self.map_provider("", "qwen-max", "dashscope") == "dashscope/qwen-max"

    def test_an_already_prefixed_model_name_is_not_prefixed_twice(self):
        assert self.map_provider("anthropic", "anthropic/claude-sonnet-4-5") == "anthropic/claude-sonnet-4-5"
        assert self.map_provider("bedrock", "bedrock/converse/x") == "bedrock/converse/x"


class TestCredentials:
    def secret(self, tmp_path, name, **files):
        d = tmp_path / name
        d.mkdir()
        for key, value in files.items():
            (d / key).write_text(value + "\n")
        # Kubernetes' own bookkeeping entries in a Secret volume
        (d / "..data").mkdir()
        return {"name": name}

    def load(self, tmp_path, ref):
        return generate_config.load_credentials(ref, secrets_dir=str(tmp_path))

    def test_aws_keys_map_to_litellm_params(self, tmp_path):
        ref = self.secret(tmp_path, "aws", AWS_ACCESS_KEY_ID="AKIA", AWS_SECRET_ACCESS_KEY="s3", AWS_SESSION_TOKEN="tok")
        assert self.load(tmp_path, ref) == {
            "aws_access_key_id": "AKIA", "aws_secret_access_key": "s3", "aws_session_token": "tok"}

    def test_bedrock_bearer_token_is_the_api_key(self, tmp_path):
        ref = self.secret(tmp_path, "bedrock", AWS_BEARER_TOKEN_BEDROCK="bearer")
        assert self.load(tmp_path, ref) == {"api_key": "bearer"}

    def test_azure_ad_app(self, tmp_path):
        ref = self.secret(tmp_path, "azure", AZURE_TENANT_ID="t", AZURE_CLIENT_ID="c", AZURE_CLIENT_SECRET="cs")
        assert self.load(tmp_path, ref) == {"tenant_id": "t", "client_id": "c", "client_secret": "cs"}

    def test_vertex_service_account_is_passed_as_a_path(self, tmp_path):
        ref = self.secret(tmp_path, "gcp", **{"service-account.json": '{"type": "service_account"}'})
        creds = self.load(tmp_path, ref)
        assert creds == {"vertex_credentials": str(tmp_path / "gcp" / "service-account.json")}

    def test_litellm_param_names_pass_through(self, tmp_path):
        ref = self.secret(tmp_path, "raw", aws_access_key_id="AKIA", api_key="k")
        assert self.load(tmp_path, ref) == {"aws_access_key_id": "AKIA", "api_key": "k"}

    def test_unknown_keys_are_skipped_by_name_without_their_value(self, tmp_path, capsys):
        ref = self.secret(tmp_path, "mixed", GEMINI_API_KEY="g", SOMETHING_ELSE="do-not-print-me")
        assert self.load(tmp_path, ref) == {"api_key": "g"}
        err = capsys.readouterr().err
        assert "SOMETHING_ELSE" in err
        assert "do-not-print-me" not in err and "g\n" not in err

    def test_missing_secret_mount_yields_nothing(self, tmp_path):
        assert self.load(tmp_path, {"name": "absent"}) == {}


class TestBuildParams:
    build = staticmethod(generate_config.build_litellm_params)

    def test_bedrock_with_keys_and_region(self):
        params = self.build({"provider": "bedrock", "modelName": "anthropic.claude-sonnet-4-5", "region": "us-east-1"},
                            None, {"aws_access_key_id": "AKIA", "aws_secret_access_key": "s3"})
        assert params == {"model": "bedrock/anthropic.claude-sonnet-4-5", "aws_access_key_id": "AKIA",
                          "aws_secret_access_key": "s3", "aws_region_name": "us-east-1"}

    def test_vertex_and_azure_typed_fields(self):
        vertex = self.build({"provider": "vertex", "modelName": "gemini-2.5-pro", "project": "p", "location": "us-central1"}, None)
        assert vertex["vertex_project"] == "p" and vertex["vertex_location"] == "us-central1"
        azure = self.build({"provider": "azure", "modelName": "gpt4o-deploy", "endpoint": "https://x.openai.azure.com",
                            "apiVersion": "2025-01-01-preview"}, "k")
        assert azure == {"model": "azure/gpt4o-deploy", "api_base": "https://x.openai.azure.com",
                         "api_key": "k", "api_version": "2025-01-01-preview"}

    def test_params_are_typed_and_win_over_derived_values(self):
        params = self.build({"provider": "bedrock", "modelName": "m", "region": "us-east-1",
                             "params": {"aws_region_name": "eu-west-1", "extra_headers": {"X-Team": "a"}}}, None)
        assert params["aws_region_name"] == "eu-west-1"
        assert params["extra_headers"] == {"X-Team": "a"}

    def test_api_key_secret_ref_wins_over_a_credentials_api_key(self):
        params = self.build({"provider": "gemini", "modelName": "m"}, "from-apiKeySecretRef", {"api_key": "from-credentials"})
        assert params["api_key"] == "from-apiKeySecretRef"

    def test_litellm_provider_with_endpoint(self):
        params = self.build({"litellmProvider": "deepseek", "modelName": "deepseek-chat", "endpoint": "https://api.deepseek.com"}, "k")
        assert params == {"model": "deepseek/deepseek-chat", "api_base": "https://api.deepseek.com", "api_key": "k"}

    def test_openai_compatible_bridges_responses_and_custom_is_the_same(self):
        spec = {"modelName": "llama3.2", "endpoint": "http://ollama:11434"}
        compat = self.build(dict(spec, provider="openai-compatible"), None)
        assert compat["use_chat_completions_api"] is True
        assert compat == self.build(dict(spec, provider="custom"), None)
        settings = generate_config.build_litellm_settings(["openai-compatible"])
        assert settings["use_chat_completions_url_for_anthropic_messages"] is True
        assert "use_chat_completions_url_for_anthropic_messages" not in generate_config.build_litellm_settings(["anthropic"])

    @pytest.mark.parametrize("spec,api_key,expected", [
        ({"provider": "openai", "modelName": "gpt-4o"}, "k", {"model": "gpt-4o", "api_key": "k"}),
        ({"provider": "azure", "modelName": "d", "endpoint": "https://a"}, "k", {"model": "azure/d", "api_base": "https://a", "api_key": "k"}),
        ({"provider": "openai", "modelName": "gpt-4o", "timeout": "30s"}, "k", {"model": "gpt-4o", "api_key": "k", "timeout": 30.0}),
    ])
    def test_existing_single_key_models_are_unchanged(self, spec, api_key, expected):
        assert self.build(spec, api_key) == expected
