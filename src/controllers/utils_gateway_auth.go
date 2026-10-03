package controllers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

const (
	// GatewayAuthSecretName is the Secret, in each cluster namespace, holding the
	// HMAC secret the gateway verifies per-agent keys with. The cluster controller
	// creates it; a value already present (e.g. user-supplied) is kept.
	GatewayAuthSecretName = "gateway-auth"
	// GatewayAuthSecretKey is the key within GatewayAuthSecretName.
	GatewayAuthSecretKey = "hmac-secret"
	// gatewayHMACSecretEnv is how the gateway receives the HMAC secret;
	// generate-config.py turns on custom_auth when it is set.
	gatewayHMACSecretEnv = "LANGOP_GATEWAY_HMAC_SECRET"

	// ModelAPIKeyEnv is the per-agent gateway key, injected next to MODEL_ENDPOINT.
	ModelAPIKeyEnv = "MODEL_API_KEY"
	// agentKeyPrefix and agentKeySignatureLength must match
	// components/model-gateway/custom_auth.py.
	agentKeyPrefix          = "sk-langop-"
	agentKeySignatureLength = 32

	// gatewayKeyRetryInterval is how soon an agent is reconciled again when its
	// cluster's gateway auth Secret does not exist yet.
	gatewayKeyRetryInterval = 10 * time.Second
)

// gatewayAgentKey is the key an agent presents to the gateway:
// "sk-langop-<agent-id>.<first 32 hex chars of HMAC-SHA256(secret, agent-id)>".
// The gateway recomputes the signature, so it needs no key store, and the agent
// id becomes the request's LiteLLM user_id.
func gatewayAgentKey(secret, agentID string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(agentID))
	return fmt.Sprintf("%s%s.%s", agentKeyPrefix, agentID, hex.EncodeToString(mac.Sum(nil))[:agentKeySignatureLength])
}

// gatewayAgentKeySecretName is the per-agent Secret holding MODEL_API_KEY, so the
// key never appears in the agent's WorkflowTemplate.
func gatewayAgentKeySecretName(agentName string) string {
	return agentName + "-gateway-key"
}

// generateGatewayHMACSecret returns 32 random bytes, hex encoded.
func generateGatewayHMACSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
