#!/usr/bin/env python3
"""
Generate LiteLLM configuration from LanguageModel CRD ConfigMap(s).

Supports two modes:
  - Multi-model (cluster proxy): reads all *.json files from /etc/langop/models/
  - Single-model (legacy):       reads /etc/langop/model.json
"""

import glob
import json
import os
import sys
import yaml
from pathlib import Path
from typing import Any, Dict, List, Optional


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
                    specs.append(json.load(f))
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


def map_provider_to_litellm(provider: str, model_name: str, endpoint: Optional[str] = None) -> str:
    """Map LanguageModel provider to LiteLLM model format."""
    provider_map = {
        "openai": model_name,
        "anthropic": model_name,
        "azure": f"azure/{model_name}",
        "bedrock": f"bedrock/{model_name}",
        "vertex": f"vertex_ai/{model_name}",
        "openai-compatible": f"openai/{model_name}",  # Use openai/ for generic OpenAI-compatible endpoints
        "custom": model_name,
    }

    return provider_map.get(provider, model_name)


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


def build_litellm_params(spec: Dict[str, Any], api_key: Optional[str]) -> Dict[str, Any]:
    """Build litellm_params from LanguageModel spec."""
    params: Dict[str, Any] = {}

    provider = spec.get("provider")
    model_name = spec.get("modelName")
    endpoint = spec.get("endpoint")

    # Set the model
    params["model"] = map_provider_to_litellm(provider, model_name, endpoint)

    # Set API base/endpoint
    if endpoint:
        # For openai-compatible providers, ensure endpoint ends with /v1
        if provider in ["openai-compatible", "custom"] and not endpoint.endswith("/v1"):
            params["api_base"] = f"{endpoint.rstrip('/')}/v1"
        else:
            params["api_base"] = endpoint

    # For openai-compatible providers, explicitly set custom_llm_provider to avoid strict validation
    if provider in ["openai-compatible", "custom"]:
        params["custom_llm_provider"] = "openai"

    # Set API key - use dummy for local/compatible endpoints without auth
    if api_key:
        params["api_key"] = api_key
    elif provider in ["openai-compatible", "custom"]:
        # Local LLM servers (LM Studio, Ollama, etc.) don't need auth but litellm requires the field
        params["api_key"] = "sk-local-dummy-key"

    # Add timeout
    if spec.get("timeout"):
        params["timeout"] = parse_duration_to_seconds(spec["timeout"])

    return params


def build_model_list(spec: Dict[str, Any], api_key: Optional[str]) -> List[Dict[str, Any]]:
    """Build the model_list section for LiteLLM config."""
    model_name = spec.get("modelName")
    litellm_params = build_litellm_params(spec, api_key)

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


def build_litellm_settings(spec: Dict[str, Any]) -> Dict[str, Any]:
    """Build litellm_settings for retries, fallbacks, etc."""
    settings: Dict[str, Any] = {}

    # Drop unknown/provider-specific params universally — agents (e.g. openclaw) may
    # send OpenAI-style fields (like `store`) that are not accepted by all providers.
    settings["drop_params"] = True

    # For openai-compatible providers, disable strict response validation
    provider = spec.get("provider")
    if provider in ["openai-compatible", "custom"]:
        settings["disable_strict_validation"] = True
        # Allow non-standard response fields
        settings["allowed_fails"] = 3
        settings["enable_json_schema_validation"] = False
        # Disable internal health checks for local models to prevent request buildup
        settings["health_check_interval"] = 0

    # Always return settings dict (even if mostly empty) for openai-compatible providers
    return settings


def generate_litellm_config(specs: List[Dict[str, Any]]) -> Dict[str, Any]:
    """Generate complete LiteLLM config from one or more LanguageModel specs."""
    config: Dict[str, Any] = {}

    # Build combined model list across all specs
    all_models: List[Dict[str, Any]] = []
    for spec in specs:
        api_key = load_api_key(spec.get("apiKeySecretRef"))
        all_models.extend(build_model_list(spec, api_key))
    config["model_list"] = all_models

    # litellm_settings: merge across all specs (first writer wins per key)
    merged_settings: Dict[str, Any] = {}
    for spec in specs:
        for k, v in build_litellm_settings(spec).items():
            merged_settings.setdefault(k, v)
    if merged_settings:
        config["litellm_settings"] = merged_settings

    config["general_settings"] = {"background_health_checks": False}

    return config


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
