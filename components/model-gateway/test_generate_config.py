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
