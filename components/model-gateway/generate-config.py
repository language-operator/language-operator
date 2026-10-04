#!/usr/bin/env python3
"""
Generate LiteLLM configuration from LanguageModel CRD ConfigMap(s).

Supports two modes:
  - Multi-model (cluster proxy): reads all *.json files from /etc/langop/models/
  - Single-model (legacy):       reads /etc/langop/model.json

Two optional environment variables let the operator's owner shape the result:
  - LANGOP_GATEWAY_EXTRA_CONFIG: a YAML mapping deep-merged into the generated
    config (callbacks, keys, any LiteLLM setting)
  - LANGOP_GATEWAY_HMAC_SECRET: enables stateless per-agent API keys through
    custom_auth.py (see that module)
"""

import glob
import json
import os
import sys
import yaml
from pathlib import Path
from collections import Counter
from typing import Any, Dict, Iterable, List, Optional


def load_model_specs(models_dir: str = "/etc/langop/models",
                     legacy_path: str = "/etc/langop/model.json") -> List[Dict[str, Any]]:
    """Load one or more LanguageModel specs from ConfigMap files."""
    # Multi-model mode: directory with one file per model
    pattern = os.path.join(models_dir, "model__*.json")
    paths = sorted(glob.glob(pattern))
    if paths:
        specs = []
        for p in paths:
            try:
                with open(p) as f:
                    spec = json.load(f)
                # The LanguageModel's own name, from model__<name>.json: wildcard
                # models are addressed by it.
                spec[NAME_KEY] = os.path.basename(p)[len("model__"):-len(".json")]
                specs.append(spec)
                print(f"✓ Loaded model spec from {p}", file=sys.stderr)
            except (json.JSONDecodeError, OSError) as e:
                print(f"✗ Failed to load {p}: {e}", file=sys.stderr)
                sys.exit(1)
        return specs

    # Legacy single-model mode
    try:
        with open(legacy_path) as f:
            spec = json.load(f)
        print(f"✓ Loaded model spec from {legacy_path}", file=sys.stderr)
        return [spec]
    except FileNotFoundError:
        print(f"⚠ No model configs found in {models_dir}/ or at {legacy_path} — starting with empty model list", file=sys.stderr)
        return []
    except json.JSONDecodeError as e:
        print(f"✗ Invalid JSON in model config: {e}", file=sys.stderr)
        sys.exit(1)


def load_api_key(secret_ref: Optional[Dict[str, str]]) -> Optional[str]:
    """Load API key from mounted secret or environment variable."""
    if not secret_ref:
        return None

    # Secret is mounted as a file in /etc/secrets/<secret-name>/<key>
    secret_name = secret_ref.get("name")
    secret_key = secret_ref.get("key", "api-key")

    secret_path = f"/etc/secrets/{secret_name}/{secret_key}"

    # Try to load from mounted secret file
    if os.path.exists(secret_path):
        with open(secret_path, 'r') as f:
            key = f.read().strip()
        print(f"✓ Loaded API key from secret {secret_name}/{secret_key}", file=sys.stderr)
        return key

    # Fallback to environment variable
    env_var = secret_ref.get("name", "").upper().replace("-", "_")
    if env_var in os.environ:
        print(f"✓ Loaded API key from environment: {env_var}", file=sys.stderr)
        return os.environ[env_var]

    print(f"⚠ API key secret not found: {secret_name}/{secret_key}", file=sys.stderr)
    return None


# provider -> LiteLLM model prefix. openai stays unprefixed (LiteLLM's default);
# custom is the deprecated spelling of openai-compatible.
PROVIDER_PREFIX = {
    "openai": None,
    "anthropic": "anthropic",
    "gemini": "gemini",
    "azure": "azure",
    "bedrock": "bedrock",
    "vertex": "vertex_ai",
    "openai-compatible": "openai",
    "custom": "openai",
}

# Where load_model_specs records a model's resource name. Not a spec field.
NAME_KEY = "_name"

# A LanguageModel whose modelName is "*" stands for the provider's whole
# catalogue. It is registered as "<name>/*", and agents call "<name>/<model>";
# LiteLLM strips "<name>/" and sends "<model>" to the provider.
WILDCARD = "*"

# Backends reached through the OpenAI chat-completions API that may not serve
# the Responses API (Ollama, vLLM, LM Studio...).
OPENAI_COMPATIBLE = ("openai-compatible", "custom")


def map_provider_to_litellm(provider: Optional[str], model_name: str,
                            litellm_provider: Optional[str] = None) -> str:
    """The LiteLLM model string: "<prefix>/<modelName>".

    A modelName that already carries the prefix is left alone.
    """
    prefix = litellm_provider if not provider else PROVIDER_PREFIX.get(provider)
    if not prefix or model_name.startswith(prefix + "/"):
        return model_name
    return f"{prefix}/{model_name}"


# Secret key -> LiteLLM param, for credentialsSecretRef. Env-style names are the
# ones each provider's own docs use; LiteLLM's own param names also work as-is.
CREDENTIAL_KEYS = {
    "api-key": "api_key",
    "API_KEY": "api_key",
    "AWS_ACCESS_KEY_ID": "aws_access_key_id",
    "AWS_SECRET_ACCESS_KEY": "aws_secret_access_key",
    "AWS_SESSION_TOKEN": "aws_session_token",
    "AWS_BEARER_TOKEN_BEDROCK": "api_key",  # LiteLLM sends a Bedrock model's api_key as the bearer token
    "AZURE_API_KEY": "api_key",
    "AZURE_AD_TOKEN": "azure_ad_token",
    "AZURE_TENANT_ID": "tenant_id",
    "AZURE_CLIENT_ID": "client_id",
    "AZURE_CLIENT_SECRET": "client_secret",
    "GEMINI_API_KEY": "api_key",
}
CREDENTIAL_PARAMS = {
    "api_key", "aws_access_key_id", "aws_secret_access_key", "aws_session_token",
    "azure_ad_token", "tenant_id", "client_id", "client_secret",
}
# Keys holding a service-account JSON. Passed to LiteLLM as the file path, so the
# JSON never lands in the generated config.
VERTEX_CREDENTIAL_KEYS = {
    "VERTEX_CREDENTIALS", "GOOGLE_APPLICATION_CREDENTIALS", "vertex_credentials",
    "service-account.json", "credentials.json",
}


def load_credentials(secret_ref: Optional[Dict[str, str]],
                     secrets_dir: str = "/etc/secrets") -> Dict[str, Any]:
    """LiteLLM params from a credentialsSecretRef Secret, mounted whole at
    <secrets_dir>/<name>/. Unknown keys are skipped by name; values are never logged.
    """
    if not secret_ref:
        return {}
    name = secret_ref.get("name")
    directory = Path(secrets_dir) / name
    if not directory.is_dir():
        print(f"⚠ Credentials secret not mounted: {name}", file=sys.stderr)
        return {}

    params: Dict[str, Any] = {}
    # Kubernetes mounts each key as a symlink into a hidden ..data directory.
    for path in sorted(p for p in directory.iterdir() if not p.name.startswith("..")):
        key = path.name
        if key in VERTEX_CREDENTIAL_KEYS:
            params["vertex_credentials"] = str(path)
        elif key in CREDENTIAL_KEYS or key in CREDENTIAL_PARAMS:
            params[CREDENTIAL_KEYS.get(key, key)] = path.read_text().strip()
        else:
            print(f"⚠ Ignoring unrecognised key {key!r} in credentials secret {name}", file=sys.stderr)
    print(f"✓ Loaded credentials from secret {name} ({', '.join(sorted(params))})", file=sys.stderr)
    return params


def parse_duration_to_seconds(timeout_str: str) -> float:
    """Parse a Go-style duration string to seconds.

    Supported suffixes (checked longest-first to avoid prefix collisions):
      ns, us, µs, ms, s, m, h
    Unknown suffixes return the default of 300.0 seconds.
    """
    if timeout_str.endswith("ms"):
        return float(timeout_str[:-2]) / 1_000
    elif timeout_str.endswith("ns"):
        return float(timeout_str[:-2]) / 1_000_000_000
    elif timeout_str.endswith("µs"):
        return float(timeout_str[:-2]) / 1_000_000
    elif timeout_str.endswith("us"):
        return float(timeout_str[:-2]) / 1_000_000
    elif timeout_str.endswith("h"):
        return float(timeout_str[:-1]) * 3600
    elif timeout_str.endswith("m"):
        return float(timeout_str[:-1]) * 60
    elif timeout_str.endswith("s"):
        return float(timeout_str[:-1])
    return 300.0  # default 5 minutes


def build_litellm_params(spec: Dict[str, Any], api_key: Optional[str],
                         credentials: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
    """Build litellm_params from LanguageModel spec.

    Precedence, lowest first: credentials from credentialsSecretRef, the
    apiKeySecretRef key, the typed fields (region, project, ...), then spec.params.
    """
    params: Dict[str, Any] = {}

    provider = spec.get("provider")
    model_name = spec.get("modelName")
    endpoint = spec.get("endpoint")

    # Set the model
    params["model"] = map_provider_to_litellm(provider, model_name, spec.get("litellmProvider"))
    if model_name == WILDCARD and params["model"] == WILDCARD:
        # openai is unprefixed for named models, but a wildcard needs the prefix.
        params["model"] = "openai/*"

    # Set API base/endpoint
    if endpoint:
        # For openai-compatible providers, ensure endpoint ends with /v1
        if provider in OPENAI_COMPATIBLE and not endpoint.endswith("/v1"):
            params["api_base"] = f"{endpoint.rstrip('/')}/v1"
        else:
            params["api_base"] = endpoint

    if provider in OPENAI_COMPATIBLE:
        # Explicitly OpenAI-style, to avoid strict validation.
        params["custom_llm_provider"] = "openai"
        # Serve /v1/responses by translating to chat completions: these backends
        # often have no Responses API, and LiteLLM would otherwise forward the
        # request to their (missing) /v1/responses and get a 404.
        params["use_chat_completions_api"] = True

    params.update(credentials or {})

    # Set API key - use dummy for local/compatible endpoints without auth
    if api_key:
        params["api_key"] = api_key
    elif provider in OPENAI_COMPATIBLE and "api_key" not in params:
        # Local LLM servers (LM Studio, Ollama, etc.) don't need auth but litellm requires the field
        params["api_key"] = "sk-local-dummy-key"

    for field, param in (("region", "aws_region_name"), ("project", "vertex_project"),
                         ("location", "vertex_location"), ("apiVersion", "api_version")):
        if spec.get(field):
            params[param] = spec[field]

    # Add timeout
    if spec.get("timeout"):
        params["timeout"] = parse_duration_to_seconds(spec["timeout"])

    params.update(spec.get("params") or {})

    return params


def build_model_list(spec: Dict[str, Any], api_key: Optional[str],
                     credentials: Optional[Dict[str, Any]] = None) -> List[Dict[str, Any]]:
    """Build the model_list section for LiteLLM config."""
    model_name = spec.get("modelName")
    if model_name == WILDCARD:
        model_name = f"{spec.get(NAME_KEY) or 'wildcard'}/{WILDCARD}"
    litellm_params = build_litellm_params(spec, api_key, credentials)

    model_entry: Dict[str, Any] = {
        "model_name": model_name,
        "litellm_params": litellm_params,
    }

    # Add rate limits
    rate_limits = spec.get("rateLimits", {})
    if rate_limits:
        if rate_limits.get("requestsPerMinute"):
            model_entry["rpm"] = rate_limits["requestsPerMinute"]
        if rate_limits.get("tokensPerMinute"):
            model_entry["tpm"] = rate_limits["tokensPerMinute"]

    return [model_entry]


def build_litellm_settings(providers: Iterable[str]) -> Dict[str, Any]:
    """Build litellm_settings from the set of providers in the model list.

    These settings are process-wide in LiteLLM, not per model, so they depend
    only on which providers are present: one openai-compatible model loosens
    validation for the whole gateway.
    """
    settings: Dict[str, Any] = {}

    # Drop unknown/provider-specific params universally — agents (e.g. openclaw) may
    # send OpenAI-style fields (like `store`) that are not accepted by all providers.
    settings["drop_params"] = True

    # For openai-compatible providers, disable strict response validation
    if any(p in OPENAI_COMPATIBLE for p in providers):
        # Send /v1/messages for OpenAI-provider models to chat completions. By
        # default LiteLLM routes it through the Responses API, which these
        # backends lack (and which, bridged, drops the reply text). Process-wide,
        # so it also covers a real openai model, which serves chat completions fully.
        settings["use_chat_completions_url_for_anthropic_messages"] = True
        settings["disable_strict_validation"] = True
        # Allow non-standard response fields
        settings["allowed_fails"] = 3
        settings["enable_json_schema_validation"] = False
        # Disable internal health checks for local models to prevent request buildup
        settings["health_check_interval"] = 0

    return settings


def warn_shared_model_names(specs: List[Dict[str, Any]]) -> None:
    """Say so when LanguageModels share a modelName.

    Agents call the gateway by modelName, so LiteLLM serves every entry with
    the same name as one model group and load-balances requests across them.
    That is the way to spread a model over several endpoints, but it also
    happens by accident, so make it visible in the gateway log.
    """
    # Wildcards are addressed by their resource name, so they never collide.
    counts = Counter(spec.get("modelName") for spec in specs if spec.get("modelName") != WILDCARD)
    for name, count in sorted(counts.items(), key=lambda kv: str(kv[0])):
        if name and count > 1:
            print(f"⚠ {count} LanguageModels share modelName {name!r}: requests for it are load-balanced across them",
                  file=sys.stderr)


def build_model_group_aliases(specs: List[Dict[str, Any]]) -> Dict[str, str]:
    """Map each LanguageModel alias onto its modelName, for router_settings.model_group_alias.

    LiteLLM resolves a real model name before an alias, so an alias equal to a
    modelName would never be used; and two models cannot answer to one alias.
    Both are dropped with a warning instead of passed on. The first model, in
    the specs' (sorted file) order, keeps a contested alias.
    """
    model_names = {spec.get("modelName") for spec in specs}
    aliases: Dict[str, str] = {}
    for spec in specs:
        model_name = spec.get("modelName")
        if model_name == WILDCARD:
            continue
        for alias in spec.get("aliases") or []:
            if alias in model_names:
                print(f"⚠ Alias {alias!r} of {model_name!r} is the modelName of a LanguageModel, which takes precedence: ignoring the alias",
                      file=sys.stderr)
            elif alias in aliases:
                if aliases[alias] != model_name:
                    print(f"⚠ Alias {alias!r} is claimed by both {aliases[alias]!r} and {model_name!r}: keeping {aliases[alias]!r}",
                          file=sys.stderr)
            else:
                aliases[alias] = model_name
    return aliases


def deep_merge(base: Dict[str, Any], extra: Dict[str, Any]) -> Dict[str, Any]:
    """Merge ``extra`` into ``base``: mappings merge recursively, anything else is replaced."""
    merged = dict(base)
    for key, value in extra.items():
        if isinstance(value, dict) and isinstance(merged.get(key), dict):
            merged[key] = deep_merge(merged[key], value)
        else:
            merged[key] = value
    return merged


def load_extra_config(env: Optional[Dict[str, str]] = None) -> Dict[str, Any]:
    """Operator-supplied LiteLLM settings from LANGOP_GATEWAY_EXTRA_CONFIG (a YAML mapping).

    Lets whoever runs the operator enable callbacks, keys or any other LiteLLM
    setting without a new image. Empty or unset means no change; anything that
    is not a mapping is a configuration error.
    """
    env = os.environ if env is None else env
    raw = env.get("LANGOP_GATEWAY_EXTRA_CONFIG", "")
    if not raw.strip():
        return {}
    try:
        extra = yaml.safe_load(raw)
    except yaml.YAMLError as e:
        print(f"✗ LANGOP_GATEWAY_EXTRA_CONFIG is not valid YAML: {e}", file=sys.stderr)
        sys.exit(1)
    if extra is None:
        return {}
    if not isinstance(extra, dict):
        print("✗ LANGOP_GATEWAY_EXTRA_CONFIG must be a YAML mapping of LiteLLM settings", file=sys.stderr)
        sys.exit(1)
    return extra


def apply_operator_settings(config: Dict[str, Any], env: Optional[Dict[str, str]] = None) -> Dict[str, Any]:
    """Apply the env-driven extras: the extra-config merge and the HMAC agent-key auth."""
    env = os.environ if env is None else env

    extra = load_extra_config(env)
    if extra:
        config = deep_merge(config, extra)
        print(f"✓ Merged LANGOP_GATEWAY_EXTRA_CONFIG ({', '.join(sorted(extra))})", file=sys.stderr)

    if env.get("LANGOP_GATEWAY_HMAC_SECRET"):
        general = dict(config.get("general_settings") or {})
        general.setdefault("custom_auth", "custom_auth.user_api_key_auth")
        config["general_settings"] = general
        print("✓ Per-agent HMAC keys enabled (custom_auth)", file=sys.stderr)

    return config


def generate_litellm_config(specs: List[Dict[str, Any]]) -> Dict[str, Any]:
    """Generate complete LiteLLM config from one or more LanguageModel specs."""
    config: Dict[str, Any] = {}

    # Build combined model list across all specs
    all_models: List[Dict[str, Any]] = []
    for spec in specs:
        api_key = load_api_key(spec.get("apiKeySecretRef"))
        credentials = load_credentials(spec.get("credentialsSecretRef"))
        all_models.extend(build_model_list(spec, api_key, credentials))
    config["model_list"] = all_models
    warn_shared_model_names(specs)

    aliases = build_model_group_aliases(specs)
    if aliases:
        config["router_settings"] = {"model_group_alias": aliases}

    if specs:
        config["litellm_settings"] = build_litellm_settings(spec.get("provider") or spec.get("litellmProvider") for spec in specs)

    general_settings: Dict[str, Any] = {"background_health_checks": False}
    config["general_settings"] = general_settings

    return apply_operator_settings(config)


def main():
    """Main entry point."""
    print("🔧 Generating LiteLLM config from LanguageModel spec(s)...", file=sys.stderr)

    specs = load_model_specs()
    litellm_config = generate_litellm_config(specs)

    output = yaml.dump(litellm_config, default_flow_style=False, sort_keys=False)
    print(output)

    if specs:
        print(f"✅ LiteLLM config generated successfully ({len(specs)} model(s))", file=sys.stderr)
        for spec in specs:
            print(f"   {spec.get('provider')}/{spec.get('modelName')}", file=sys.stderr)
    else:
        print("✅ LiteLLM config generated with empty model list (no models registered yet)", file=sys.stderr)


if __name__ == "__main__":
    main()
