# LanguageModel Examples

Example `LanguageModel` resources for the cluster's model gateway. Each one is a complete, valid spec: replace `my-cluster` with your LanguageCluster's namespace, create the Secret named in its comment, and apply it.

```bash
kubectl apply -f anthropic-model.yaml
```

These files are checked in CI: a unit test runs every `LanguageModel` here through the admission webhook's validation, and an integration test applies them all to a real API server.

| File | Provider | Needs | Secret |
|------|----------|-------|--------|
| `anthropic-model.yaml` | Anthropic | — | `api-key` |
| `openai-model.yaml` | OpenAI | — | `api-key` |
| `azure-openai.yaml` | Azure OpenAI | `endpoint`, `apiVersion`; `modelName` is the deployment name | `api-key`, or an Azure AD app via `credentialsSecretRef` |
| `aws-bedrock.yaml` | AWS Bedrock | `region` | `AWS_ACCESS_KEY_ID` + `AWS_SECRET_ACCESS_KEY`, or `AWS_BEARER_TOKEN_BEDROCK`, via `credentialsSecretRef` |
| `google-vertex.yaml` | Google Vertex AI | `project`, `location` | `service-account.json` via `credentialsSecretRef` |
| `deepseek.yaml` | Any other LiteLLM provider, by `litellmProvider` | — | `api-key` |
| `ollama-local.yaml` | Ollama (OpenAI-compatible) | `endpoint` | none |
| `lm-studio.yaml` | LM Studio (OpenAI-compatible) | `endpoint` | none |
| `multi-endpoint-loadbalanced.yaml` | Two models sharing a `modelName` | — | none |
| `corporate-proxy.yaml` | Any, through an HTTP(S) proxy | proxy variables on the gateway | as for the model |

The full field reference, with every provider's options, is [docs/api/languagemodel.md](../../../docs/api/languagemodel.md).

## Credentials

Credentials never go in the LanguageModel itself, and never reach agent pods:

- **`apiKeySecretRef`** names one key of a Secret (default key `api-key`).
- **`credentialsSecretRef`** names a whole Secret, for providers that need several values. Its keys are recognised by name (`AWS_ACCESS_KEY_ID`, `AWS_BEARER_TOKEN_BEDROCK`, `AZURE_CLIENT_SECRET`, `service-account.json`, …) and applied to that model only.

## Networking

A LanguageModel has no network settings. The model gateway makes every provider call, and the operator does **not** restrict the gateway's outbound traffic: its NetworkPolicy applies to agent pods only.

- If your cluster enforces a **default-deny** egress policy, allow the gateway pods (label `app.kubernetes.io/component: gateway`) to reach your providers, usually on TCP 443.
- If providers are only reachable through a **corporate proxy**, set `HTTPS_PROXY`/`HTTP_PROXY`/`NO_PROXY` on the gateway through the LanguageCluster, as in `corporate-proxy.yaml`. The gateway honours the standard proxy variables.
- Self-hosted servers (Ollama, LM Studio, vLLM) must be reachable from the gateway pod at the `endpoint` you give.
