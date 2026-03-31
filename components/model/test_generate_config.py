"""Tests for generate-config.py — focused on the timeout duration parser."""

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
