"""
Stateless per-agent API keys for the LiteLLM proxy, enabled by
LANGOP_GATEWAY_HMAC_SECRET.

A key is ``sk-langop-<agent-id>.<signature>`` where the signature is the
first 32 hex characters of HMAC-SHA256(secret, agent-id). The gateway can
verify it without a database or a key list, and the agent id lands on every
request as the LiteLLM user id so spend callbacks can attribute usage to the
agent. The proxy's own master key (LITELLM_MASTER_KEY) keeps working so
clients that do not carry an agent key are not locked out.

generate-config.py wires this module in as ``general_settings.custom_auth``
whenever the secret is set; nothing changes when it is not.
"""

import hmac
import hashlib
import os
from typing import Optional

KEY_PREFIX = "sk-langop-"
SIGNATURE_LENGTH = 32


def sign(secret: str, agent_id: str) -> str:
    """The signature part of an agent key."""
    digest = hmac.new(secret.encode(), agent_id.encode(), hashlib.sha256).hexdigest()
    return digest[:SIGNATURE_LENGTH]


def agent_key(secret: str, agent_id: str) -> str:
    """A complete agent key for ``agent_id``."""
    return f"{KEY_PREFIX}{agent_id}.{sign(secret, agent_id)}"


def verify(secret: str, api_key: Optional[str]) -> Optional[str]:
    """The agent id carried by a valid key, or None."""
    if not api_key or not api_key.startswith(KEY_PREFIX):
        return None
    body = api_key[len(KEY_PREFIX):]
    agent_id, sep, signature = body.rpartition(".")
    if not sep or not agent_id or not signature:
        return None
    if hmac.compare_digest(signature, sign(secret, agent_id)):
        return agent_id
    return None


def _master_key_matches(api_key: Optional[str]) -> bool:
    master = os.environ.get("LITELLM_MASTER_KEY")
    return bool(master) and bool(api_key) and hmac.compare_digest(api_key, master)


async def user_api_key_auth(request, api_key: str):
    """LiteLLM custom auth hook (``general_settings.custom_auth``)."""
    from litellm.proxy._types import LitellmUserRoles, UserAPIKeyAuth

    secret = os.environ.get("LANGOP_GATEWAY_HMAC_SECRET", "")
    key = (api_key or "").removeprefix("Bearer ").strip()

    if _master_key_matches(key):
        return UserAPIKeyAuth(api_key=key, user_role=LitellmUserRoles.PROXY_ADMIN)

    agent_id = verify(secret, key) if secret else None
    if agent_id is None:
        raise Exception("invalid API key")

    return UserAPIKeyAuth(
        api_key=key,
        user_id=agent_id,
        metadata={"langop_agent_id": agent_id},
    )
