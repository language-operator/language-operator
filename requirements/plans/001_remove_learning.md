# Language Operator: Pivot to Pure Agentic Runtime Framework

## Strategic Vision

Transform language-operator from a "learning AI system" into **the best Kubernetes runtime for AI agents** - focusing on reliability, scale, multi-agent orchestration, and observability.

**Core Philosophy**: Let specialized tools (mem0, MCP servers, knowledge bases) handle learning/memory. Focus on being an excellent runtime that makes deploying and orchestrating agents as easy as deploying containers.

## Architecture Changes

### What's Being Removed

#### 1. **Learning System** (Complete Removal)
- **Controller**: [src/controllers/learning_controller.go](src/controllers/learning_controller.go) - Pattern detection, optimization triggering
- **CRD**: [src/api/v1alpha1/languageagentversion_types.go](src/api/v1alpha1/languageagentversion_types.go) - Version management for optimized code
- **Metrics**: [src/pkg/learning/metrics.go](src/pkg/learning/metrics.go) - Cost savings calculations
- **Events**: Learning-related events in [src/pkg/events/manager.go](src/pkg/events/manager.go)

#### 2. **Synthesis Engine** (Complete Removal)
- **Package**: [src/pkg/synthesis/](src/pkg/synthesis/) - Entire synthesis package
  - `synthesizer.go` - LLM-based code generation
  - `agent_synthesis.tmpl` - Ruby DSL generation template
  - `task_synthesis.tmpl` - Task optimization template
  - `persona_distillation.tmpl` - Persona condensation
- **Synthesis controller logic** in [src/controllers/languageagent_controller.go](src/controllers/languageagent_controller.go)

#### 3. **Persona System** (Simplification, Not Removal)
- **Keep CRD**: [src/api/v1alpha1/languagepersona_types.go](src/api/v1alpha1/languagepersona_types.go) - But simplify
- **Keep Controller**: [src/controllers/languagepersona_controller.go](src/controllers/languagepersona_controller.go) - But simplify
- **Remove**: Persona distillation/synthesis logic (no LLM-based condensation)
- **Remove**: Complex persona merging and compilation
- **Keep**: Persona as pure configuration data (mounted to agents)
- **Change**: Personas become config files mounted into containers, not synthesis inputs

#### 4. **Memory Stores** (Removal)
- `MemoryStoreSpec` from LanguageAgent CRD
- Redis/Postgres/S3 conversation persistence logic
- Multi-turn conversation state management

#### 5. **Self-Healing via Re-Synthesis**
- Crash-triggered re-synthesis logic
- `SynthesisInfo` status tracking
- Runtime error → code regeneration pipeline

### What's Being Simplified

#### LanguageAgent CRD - Dramatic Simplification

**Remove these fields**:
- `SynthesisModelRef` - no synthesis
- `MemoryStore` - delegated to MCP tools
- `RunsPendingLearning`, `LearningRequestPending` - no learning
- `SynthesisInfo` status - no synthesis
- `RuntimeErrors[]` with self-healing - simplified error tracking only
- All learning-related metrics (SymbolicTaskCount, NeuralTaskCount, LearningHealthScore, ProjectedMonthlyCostSavings)

**Keep these fields**:
- `Name`, `UUID` - Identity
- `Instructions` (string) - **KEPT** - Task definition passed as env var/mounted file
- `InstructionsFrom` - **NEW** - Reference to ConfigMap/Secret with instructions
- `PersonaRefs` - **KEPT** - Personas mounted as config files (not used for synthesis)
- `Mode` (autonomous, interactive, scheduled, event-driven)
- `Schedule` (for scheduled mode)
- `Image` - **NEW PRIMARY FIELD** - Container image with agent runtime
- `ImagePullPolicy`, `ImagePullSecrets`
- `Replicas`, `Resources`, `Affinity` - K8s deployment configuration
- `Tools[]` - MCP tool references
- `Models[]` - LLM configurations for agent to use
- `Safety` - Tool approval, rate limits, blocked tools
- `NetworkPolicy` - Agent-to-agent communication rules
- `Workspace` - PVC configuration
- `Env[]`, `EnvFrom[]` - Environment configuration
- `ServiceAccount` - RBAC configuration
- `Metrics` - Simplified execution metrics (ExecutionCount, SuccessRate, AverageExecutionTime, TotalToolCalls)

**New architecture**:
```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: data-analyst
spec:
  # Container image: Provides agent runtime/framework
  image: myregistry/agent-runtime:python-v1.2.0
  imagePullPolicy: Always

  # Instructions: Task definition (passed as env var or mounted file)
  instructions: |
    You are a data analyst. Analyze CSV files and generate insights.
    Focus on trends, anomalies, and actionable recommendations.

  # OR reference from ConfigMap/Secret
  # instructionsFrom:
  #   configMapRef:
  #     name: analyst-instructions
  #     key: task.txt

  # Personas: Behavioral configuration (mounted as config files)
  personaRefs:
    - name: analytical-persona
    - name: professional-tone

  # Execution mode
  mode: autonomous

  # Tools accessed via MCP (memory, knowledge, actions)
  tools:
    - name: mem0-memory
      enabled: true
    - name: vector-db-knowledge
      enabled: true
    - name: python-executor
      enabled: true

  # Models available to agent
  models:
    - name: claude-sonnet
      role: primary
    - name: gpt-4
      role: fallback

  # Standard K8s configuration
  replicas: 3
  resources:
    limits:
      memory: 2Gi
      cpu: 1000m

  # Multi-agent networking
  networkPolicy:
    allowAgentGroups:
      - analytics-team
```

### What's Being Enhanced

#### 1. **A2A Protocol Support** (Built-in by Design)

Every LanguageAgent already gets a ClusterIP Service on port 8080 and an external HTTPRoute. Phase 2 formalises this as A2A compliance:

- Agents must expose `GET /.well-known/agent.json` (A2A Agent Card)
- Agents must expose `POST /tasks` and `GET /tasks/{id}` for receiving work from other agents
- The operator adds a NetworkPolicy ingress rule allowing any agent pod to reach any other agent on port 8080
- Agent-to-agent orchestration is handled by the agents themselves (or external tools like Argo, Temporal) — not by the operator

### What Stays The Same

- **LanguageTool CRD**: MCP tool definitions (core to new architecture)
- **LanguageModel CRD**: Model configurations
- **LanguageCluster CRD**: Multi-cluster agent deployments
- **Webhook routing**: Agent-to-agent HTTP communication
- **NetworkPolicy generation**: Security and isolation
- **OpenTelemetry integration**: Metrics and tracing
- **Helm chart structure**: Deployment mechanism

## Implementation Phases

### Phase 1: Removal (Breaking Changes)

**Files to delete entirely**:
1. `src/controllers/learning_controller.go`
2. `src/controllers/languageagentversion_controller.go`
3. `src/api/v1alpha1/languageagentversion_types.go`
4. `src/pkg/synthesis/` (entire directory)
5. `src/pkg/learning/` (entire directory)

**Files to modify**:
1. [src/api/v1alpha1/languageagent_types.go](src/api/v1alpha1/languageagent_types.go)
   - Remove: SynthesisModelRef, MemoryStore, learning fields
   - Keep: PersonaRefs (but change behavior - mounted as config, not for synthesis)
   - Keep: Instructions (but change behavior - no synthesis, just pass to container)
   - Add: InstructionsFrom (ConfigMapRef/SecretRef)
   - Add: Image, ImagePullPolicy as primary fields
   - Simplify: Metrics struct (remove learning metrics)
   - Simplify: Status struct (remove synthesis/learning status)

2. [src/controllers/languageagent_controller.go](src/controllers/languageagent_controller.go)
   - Remove: Synthesis reconciliation logic
   - Remove: Learning trigger logic
   - Remove: Persona merging/distillation for synthesis
   - Add: Persona config mounting logic (mount personas as JSON/YAML files)
   - Add: Instructions mounting logic (env var or file mount)
   - Add: InstructionsFrom ConfigMap/Secret mounting
   - Keep: Pod/Deployment creation
   - Keep: Webhook routing
   - Keep: NetworkPolicy generation
   - Simplify: Status updates (no synthesis/learning tracking)

3. [src/api/v1alpha1/languagepersona_types.go](src/api/v1alpha1/languagepersona_types.go)
   - Keep: All persona fields (SystemPrompt, Instructions, Tone, KnowledgeSources, etc.)
   - Remove: Validation complexity (no synthesis validation needed)
   - Simplify: Status to basic Ready/NotReady (no synthesis health)

4. [src/controllers/languagepersona_controller.go](src/controllers/languagepersona_controller.go)
   - Remove: Persona distillation logic (LLM-based condensation)
   - Simplify: Just validate persona and mark Ready
   - Keep: Basic CRUD reconciliation

5. [src/api/v1alpha1/zz_generated.deepcopy.go](src/api/v1alpha1/zz_generated.deepcopy.go)
   - Regenerate after CRD changes

6. [src/pkg/events/manager.go](src/pkg/events/manager.go)
   - Remove: Learning-related events (RecordLearningTask, RecordTaskSymbolicConversion, etc.)
   - Keep: Lifecycle events (Created, Ready, Failed)
   - Keep: Persona events (RecordPersonaCreated, RecordPersonaReady)

7. [chart/templates/](chart/templates/)
   - Remove: AgentVersion CRD, Learning RBAC
   - Keep: Persona CRD (simplified)
   - Update: Agent CRD with simplified spec
   - Keep: Tool, Model, Cluster CRDs

### Phase 2: Complete the Runtime (with A2A by Design)

Phase 2 finishes what Phase 1 deferred and establishes Google's A2A protocol as the standard for agent communication. One new CRD (`LanguageAgentTask`) is introduced to surface blocked task states — this is observability, not workflow management.

#### 2a. Write Specification Documents First ✅ COMPLETE

Contracts defined in `spec/`:

**`spec/agents.md`** — Contract for agent runtime containers:

| Path | Content | Source |
|------|---------|--------|
| `/etc/agent/instructions.txt` | Task instructions (plain text) | `spec.instructions` or `spec.instructionsFrom` |
| `/etc/agent/config.yaml` | Structured config: personas, tools, models, agent metadata | Assembled by operator from all referenced resources |

Environment variables injected by operator: `AGENT_NAME`, `AGENT_NAMESPACE`, `AGENT_UUID`, `AGENT_MODE`, `AGENT_CLUSTER_NAME`, `AGENT_CLUSTER_UUID`, `AGENT_OPERATOR_WEBHOOK_URL`, `AGENT_OPERATOR_WEBHOOK_TOKEN`.

**A2A requirements** — agent must expose on port 8080 (full A2A spec, no `/v1/` prefix):
- `GET /.well-known/agent.json` / `GET /agentCard` — Agent Card (public/extended)
- `POST /messages` / `POST /messages:stream` — send message (sync/SSE)
- `GET /tasks`, `GET /tasks/{id}`, `POST /tasks/{id}:cancel`, `GET /tasks/{id}:subscribe` — task management
- `POST /tasks/{id}/pushNotificationConfigs` (+ GET, DELETE) — push notification config
- `GET /health` — readiness probe

**Push notification contract** — agents must register the operator as a subscriber on startup and send state-change notifications on every transition, including `input-required` and `auth-required`. These are not errors — the operator will create a `LanguageAgentTask` CR and send `POST /messages` to unblock when resolution arrives.

**`spec/tools.md`** — Contract for LanguageTool implementations:
- Must expose MCP protocol at `spec.port` (default 8080)
- Must respond to `GET /health`
- Must list available tools at the MCP tools/list endpoint
- The operator injects the resolved endpoint URL into the agent's tools ConfigMap

#### 2b. Instructions Mounting (`/etc/agent/instructions.txt`)

**What**: Inject task instructions as plain text. Two sources:
1. `spec.instructions` (inline string) → create/maintain a ConfigMap, mount as `/etc/agent/instructions.txt`
2. `spec.instructionsFrom.configMapRef` / `spec.instructionsFrom.secretRef` → project the referenced key directly

**Files**:
- `src/controllers/languageagent_controller.go` — `buildVolumes()` at existing TODO (~line 724); add `reconcileInstructionsConfigMap()` called alongside existing `reconcileConfigMap()`

**Key types already in place**: `InstructionsSource`, `InstructionsFrom *InstructionsSource` in `src/api/v1alpha1/languageagent_types.go`

#### 2c. Config Assembly (`/etc/agent/config.yaml`)

**What**: Assemble a single YAML config file containing personas, tools, models, and agent metadata. The operator resolves all referenced resources and writes one ConfigMap that gets mounted at `/etc/agent/config.yaml`.

Structure:
```yaml
agent: {name, namespace, uuid, mode}
personas: [...] # full spec of each referenced LanguagePersona
tools: {name: {endpoint, protocol}} # resolved in-cluster MCP endpoints
models: {name: {role, provider, endpoint, model, secretRef}}
```

**Files**:
- `src/controllers/languageagent_controller.go` — add `reconcileAgentConfigMap()` that builds and reconciles the config YAML; mount result at `/etc/agent/config.yaml`

**Helpers to reuse**:
- `GenerateConfigMapName(name, "persona")` in `src/controllers/utils.go`
- `fetchPersona()` already validates persona readiness — use same guard when loading persona specs

#### 2d. A2A NetworkPolicy

The existing `reconcileNetworkPolicy()` only allows trigger pods and the dashboard to reach agents on port 8080. Add a rule permitting **agent-to-agent** traffic so any agent pod in the same cluster can call another agent's A2A endpoint.

**File**: `src/controllers/languageagent_controller.go` — `reconcileNetworkPolicy()` (~line 1149)

**Change**: Add an ingress rule allowing pods with label `langop.io/kind=LanguageAgent` to reach port 8080.

#### 2e. Re-enable and Fix Tests

**Delete** (synthesis-only, no longer relevant):
- `src/controllers/languageagent_controller_test.go.disabled` — discard entirely

**Restore with edits**:
- `src/pkg/events/manager_test.go.disabled` → rename to `.go`, delete synthesis/self-healing test functions, keep registry and resource event tests

**New tests** (fresh `src/controllers/languageagent_controller_test.go`):
- `TestBuildVolumes_InlineInstructions`
- `TestBuildVolumes_InstructionsFromConfigMap`
- `TestBuildVolumes_PersonaRefs`
- `TestBuildVolumes_NoExtras`
- Use existing pattern: `testutil.SetupTestScheme(t)`, `fake.NewFakeClient`, `mockRegistryManager` (already in `languagetool_controller_test.go`)

#### 2f. Webhook Validation

Add to `validateSpec()` in `src/api/v1alpha1/languageagent_webhook.go`:
- Error if both `spec.instructions` and `spec.instructionsFrom` are set (ambiguous)
- Error if `spec.image` is empty (agents need a runtime image)

#### 2g. LanguageAgentTask CRD and Controller

**What**: Surface blocked task states (`input-required`, `auth-required`) as Kubernetes resources. The operator receives push notifications from agents and creates/updates `LanguageAgentTask` CRs. When a resolution is patched onto the CR, the controller sends `POST /messages` to the agent to unblock the task.

**New files**:
- `src/api/v1alpha1/languageagenttask_types.go` ✅ CREATED — CRD types with `state`, `inputRequired`, `authRequired`, `artifacts` in status
- `src/controllers/languageagenttask_controller.go` — watches `LanguageAgentTask`, sends resolution messages to agents
- `src/pkg/webhook/task_handler.go` — HTTP handler for operator push notification endpoint (`AGENT_OPERATOR_WEBHOOK_URL`)

**Key fields on `LanguageAgentTask`**:
- `spec.agentRef`, `spec.taskId`, `spec.contextId` — links back to the A2A task
- `status.state` — mirrors A2A states
- `status.inputRequired.prompt` / `status.authRequired.scheme` — what is needed
- Conditions: `Blocked`, `Resolved`, `Terminal`

#### Verification

```bash
cd src && make build && go test ./controllers ./api/...

# Verify pod spec has correct mounts:
kubectl get deployment <agent-name> -o jsonpath='{.spec.template.spec.volumes}' | jq .
# Expect: /etc/agent/instructions.txt and /etc/agent/config.yaml

# Verify A2A endpoint is reachable from another agent pod:
kubectl exec -it <other-agent-pod> -- curl http://<agent-name>.<ns>.svc.cluster.local:8080/.well-known/agent.json

# Verify blocked task surfaces as a Kubernetes resource:
kubectl get latask -A
# Expect: tasks in input-required or auth-required state appear here
```

### Phase 3: Documentation & Migration

**New documentation**:
1. `docs/migration-guide.md` - How to migrate from learning-based to container-based agents
2. `docs/architecture.md` - Updated architecture: runtime vs instructions separation
3. `docs/workflows.md` - Multi-agent workflow patterns
4. `docs/tools.md` - Using MCP tools for memory/knowledge
5. `docs/base-images.md` - Using and building agent runtime images
6. `examples/container-agent/` - Example agent deployments with instructions

**Migration examples**:
- Converting synthesized agent → container runtime + instructions
- Using mem0 tool for memory instead of MemoryStore
- Mounting instructions from ConfigMaps
- Setting up multi-agent workflows

## Critical Files Reference

### Files to Delete (Phase 1)
- [src/controllers/learning_controller.go](src/controllers/learning_controller.go)
- [src/controllers/languageagentversion_controller.go](src/controllers/languageagentversion_controller.go)
- [src/api/v1alpha1/languageagentversion_types.go](src/api/v1alpha1/languageagentversion_types.go)
- [src/pkg/synthesis/](src/pkg/synthesis/) (entire directory)
- [src/pkg/learning/](src/pkg/learning/) (entire directory)

### Files to Modify (Phase 1)
- [src/api/v1alpha1/languageagent_types.go](src/api/v1alpha1/languageagent_types.go) - Simplify to container-based model
- [src/controllers/languageagent_controller.go](src/controllers/languageagent_controller.go) - Remove synthesis/learning, add config mounting
- [src/api/v1alpha1/languagepersona_types.go](src/api/v1alpha1/languagepersona_types.go) - Simplify (remove synthesis validation)
- [src/controllers/languagepersona_controller.go](src/controllers/languagepersona_controller.go) - Simplify (remove distillation)
- [src/pkg/events/manager.go](src/pkg/events/manager.go) - Remove learning events
- [chart/templates/](chart/templates/) - Remove version CRD, simplify persona CRD

### Files to Create (Phase 2)
- ✅ `spec/agents.md` — agent runtime container contract (A2A endpoints, mount paths, push notification protocol)
- ✅ `spec/tools.md` — LanguageTool implementation contract (MCP, health, schema)
- ✅ `src/api/v1alpha1/languageagenttask_types.go` — LanguageAgentTask CRD types
- `src/controllers/languageagenttask_controller.go` — watches LanguageAgentTask, sends resolution messages to agents
- `src/pkg/webhook/task_handler.go` — operator push notification HTTP handler
- `src/controllers/languageagent_controller_test.go` — new test file for volume/mount logic

### Files to Modify (Phase 2)
- [src/controllers/languageagent_controller.go](src/controllers/languageagent_controller.go) — `buildVolumes()` instructions + config mounting; `reconcileNetworkPolicy()` A2A ingress rule; add `reconcileInstructionsConfigMap()` and `reconcileAgentConfigMap()`; inject `AGENT_OPERATOR_WEBHOOK_URL` / `AGENT_OPERATOR_WEBHOOK_TOKEN` env vars
- [src/api/v1alpha1/languageagent_webhook.go](src/api/v1alpha1/languageagent_webhook.go) — add image required + instructions mutual-exclusion validation
- [src/pkg/events/manager_test.go.disabled](src/pkg/events/manager_test.go.disabled) — restore as `.go`, remove synthesis tests
- [src/cmd/main.go](src/cmd/main.go) — register `LanguageAgentTask` controller and push notification webhook server

## Verification Steps

### Phase 1 Verification (Removal)
1. **Build succeeds**: `cd src && make build`
2. **Tests pass**: `cd src && make test`
3. **CRDs generate**: `cd src && make manifests`
4. **No synthesis references**: `grep -r "synthesis" src/ | wc -l` should be minimal (only comments)
5. **No learning references**: `grep -r "learning" src/ | wc -l` should be minimal
6. **Persona CRD exists**: `kubectl get crd languagepersonas.langop.io` succeeds
7. **Personas mount correctly**: Check generated pod spec includes persona volume mounts

### Phase 2 Verification (Runtime)
1. **Tests pass**:
   ```bash
   cd src && go test ./controllers ./api/...
   ```

2. **Instructions mounted correctly**:
   ```bash
   kubectl apply -f examples/container-agent/simple-agent.yaml
   kubectl get deployment simple-agent -o jsonpath='{.spec.template.spec.volumes}' | jq .
   # Expect: volume named "instructions" mounting to /etc/agent/instructions
   ```

3. **Personas mounted correctly**:
   ```bash
   kubectl get deployment simple-agent -o jsonpath='{.spec.template.spec.containers[0].volumeMounts}' | jq .
   # Expect: one mount per personaRef at /etc/agent/personas/{name}.json
   ```

4. **Webhook rejects invalid spec**:
   ```bash
   # Should fail: no image
   kubectl apply -f - <<EOF
   apiVersion: langop.io/v1alpha1
   kind: LanguageAgent
   metadata: {name: bad-agent, namespace: default}
   spec: {instructions: "do something"}
   EOF
   # Expect: admission webhook error about missing image
   ```

## Migration Path for Existing Users

### Before (Synthesis-based)
```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: data-analyst
spec:
  instructions: |
    You are a data analyst. Analyze CSV files and generate insights.
  personaRefs:
    - name: analytical-persona  # Used for synthesis
  tools:
    - name: python-executor
  synthesisModelRef:
    name: claude-sonnet  # LLM used to generate Ruby code
```

### After (Container-based)
```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: data-analyst
spec:
  # Container image: Provides agent runtime
  image: langop/agent-runtime:python-v1.0.0

  # Instructions: Task definition (config, not code)
  instructions: |
    You are a data analyst. Analyze CSV files and generate insights.

  # Personas: Mounted as config files (NOT used for synthesis)
  personaRefs:
    - name: analytical-persona  # Mounted to /app/personas/analytical-persona.json

  mode: autonomous
  tools:
    - name: mem0-memory  # For persistent memory
    - name: python-executor
  models:
    - name: claude-sonnet
      role: primary
```

**Key Architecture Change**:
- **Container Image**: Agent runtime/framework (how to execute, how to use tools)
- **Instructions**: Task definition (what to do) - passed as env var or mounted file
- **Personas**: Behavioral configuration (tone, knowledge sources, constraints) - mounted as JSON files
- **Separation of Concerns**: Runtime is reusable, instructions and personas are configuration

**User responsibilities**:
1. Choose or build agent runtime image (or use provided base images)
2. Define task instructions in YAML or ConfigMap
3. Create personas with behavioral configuration (optional)
4. Configure tools and models
5. Deploy via kubectl/Helm

**Agent Runtime Responsibilities** (in container code):
1. Read `/app/instructions.txt` for task definition
2. Read `/app/personas/*.json` for behavioral configuration
3. Apply persona settings (system prompt, tone, knowledge sources, constraints)
4. Connect to tools via endpoints in environment variables
5. Use models configured in `/app/models/config.json`
6. Execute task according to mode (autonomous/interactive/scheduled)

**Operator responsibilities**:
1. Deploy agent as Deployment/CronJob
2. **Mount instructions** as `/app/instructions.txt` or inject as `AGENT_INSTRUCTIONS` env var
3. **Mount personas** as `/app/personas/*.json` (one file per persona)
4. Inject tool endpoints and credentials as environment variables
5. Inject model configurations as environment variables
6. Provide webhook routing for multi-agent communication
7. Collect telemetry and traces
8. Manage lifecycle and scaling

**Example mounted filesystem in agent container**:
```
/app/
  instructions.txt           # Task definition
  personas/
    analytical-persona.json  # First persona config
    professional-tone.json   # Second persona config
  tools/
    config.json             # Tool endpoints and schemas
  models/
    config.json             # Model configurations
```

## Success Metrics

1. **Simplicity**: CRD size reduced by >50% (measured in field count)
2. **Performance**: Controller reconciliation <100ms (no LLM calls)
3. **Reliability**: Agent uptime >99.9% (no synthesis failures)
4. **Adoption**: Users can deploy first agent in <5 minutes
5. **Observability**: 100% of agent executions traced end-to-end

## Risks & Mitigations

**Risk**: Breaking change for existing users
**Mitigation**:
- Provide migration guide with conversion examples
- Keep v1alpha1 for old CRDs, introduce v1alpha2 for new architecture
- Automated conversion webhook (old spec → new spec)

**Risk**: Loss of "magic" synthesis feature
**Mitigation**:
- Provide starter templates for common agent patterns
- Example Dockerfiles for various languages
- Pre-built base images with MCP client libraries

**Risk**: Increased complexity for users (building containers)
**Mitigation**:
- Provide official base images for Python, Node.js, Go
- Users only provide instructions (configuration), not code (in most cases)
- Base images handle: MCP tool integration, model API clients, telemetry
- Comprehensive documentation and examples
- GitHub Actions templates for custom runtime builds (advanced use case)

## Timeline

- **Phase 1 (Removal)**: ✅ COMPLETE
- **Phase 2 (Runtime completion)**: 1-2 days — instructions/persona mounting, tests, webhook
- **Phase 3 (Docs & examples)**: 1 week — migration guide, base images, example YAMLs

## Phase 1 Completion Status

**Status**: ✅ **COMPLETED** (December 2024)

### What Was Accomplished

#### Files Deleted (~14,000+ lines removed)
- ✅ `src/controllers/learning_controller.go` (2,177 lines)
- ✅ `src/controllers/languageagentversion_controller.go` (303 lines)
- ✅ `src/api/v1alpha1/languageagentversion_types.go` (147 lines)
- ✅ `src/pkg/synthesis/` - Entire directory (~7,000+ lines)
  - `synthesizer.go`, `agent_synthesis.tmpl`, `task_synthesis.tmpl`, `persona_distillation.tmpl`
  - `configmap.go`, `quota_manager.go`, `rate_limiter.go`, `schema.go`, `validator.go`
  - All tests
- ✅ `src/pkg/learning/` - Entire directory (~700+ lines)
  - `metrics.go`, `metrics_test.go`
- ✅ `chart/crds/langop.io_languageagentversions.yaml` - Helm CRD removed

#### Files Modified Successfully
- ✅ `src/api/v1alpha1/languageagent_types.go`
  - Removed: `AgentVersionRef`, `MemoryStore`, `SynthesisInfo`, `RuntimeErrors`, all learning metrics
  - Added: `InstructionsFrom` field for ConfigMap/Secret mounting
  - Kept: `Image`, `Instructions`, `PersonaRefs` (behavior changed to mounting)

- ✅ `src/controllers/languageagent_controller.go`
  - Removed: `reconcileCodeConfigMap`, `distillPersona`, `getSynthesisModel`, `createSynthesizer` (~500 lines)
  - Removed: `performSelfHealingSynthesis`, `shouldAttemptSelfHealing`, `detectPodFailures`, `buildErrorContext` (~400 lines)
  - Removed: `createInitialAgentVersion`, `resolveCodeConfigMapName`, `checkAndIncrementRunsCounter` (~250 lines)
  - Removed: Self-healing detection, rate limiter, quota manager fields from reconciler struct
  - Total: ~1,150+ lines removed

- ✅ `src/api/v1alpha1/languagepersona_types.go`
  - Removed: `PersonaValidation`, `PersonaMetrics`, `ToolFrequency`, `TopicFrequency` types
  - Simplified: Status to basic `Ready`/`NotReady` with conditions only
  - Removed: `UsageCount`, `ActiveAgents`, `ValidationResult`, `Metrics` from status

- ✅ `src/controllers/languagepersona_controller.go`
  - Already simple (no distillation logic existed)
  - Just creates ConfigMap with persona JSON

- ✅ `src/pkg/events/manager.go`
  - Removed: All synthesis event methods (`RecordSynthesisStarted`, `RecordSynthesisSucceeded`, `RecordSynthesisFailed`)
  - Removed: All self-healing events (`RecordSelfHealingTriggered`, etc.)
  - Removed: Learning/optimization events (`RecordOptimizationTriggered`)
  - Removed: Rate limiting/quota events (`RecordRateLimitExceeded`, `RecordQuotaExceeded`)
  - Kept: Core lifecycle and persona events

- ✅ `src/cmd/main.go`
  - Removed: Synthesis schema validation
  - Removed: Rate limiter and quota manager initialization
  - Removed: Learning controller setup
  - Removed: LanguageAgentVersion controller setup

- ✅ CRDs Regenerated
  - `make generate` - Updated deepcopy methods
  - `make manifests` - Updated CRD YAML files
  - Copied updated CRDs to `chart/crds/`

#### Build & Test Status
- ✅ **Build**: `make build` passes successfully
- ✅ **Core Tests**: Controllers, API, Events packages all pass
- ✅ **Validation**: No compilation errors

### Items Left as TODOs (Next Steps for Phase 2)

#### 1. **Implement Instructions and Persona Mounting** (Critical)
**Location**: `src/controllers/languageagent_controller.go:746-747`

Current state (placeholder):
```go
// TODO: Add instructions mounting from InstructionsFrom ConfigMap/Secret
// TODO: Add persona mounting from PersonaRefs
```

**What needs to be done**:
1. Implement logic to mount `spec.instructions` as `/app/instructions.txt` or `AGENT_INSTRUCTIONS` env var
2. Implement logic to mount instructions from `spec.instructionsFrom.configMapRef` or `secretRef`
3. Implement logic to mount each persona from `spec.personaRefs` as `/app/personas/<persona-name>.json`
4. Update pod template generation in `buildPodTemplate()` function
5. Add volume and volumeMount entries for instructions and personas

**Files to modify**:
- `src/controllers/languageagent_controller.go` - Add mounting logic in `buildPodTemplate()`

#### 2. **Clean Up Remaining Synthesis References**

**Found 8 synthesis references in source code**:
1. `src/pkg/telemetry/adapter.go` - Comments mentioning "synthesis.version" attribute (OK to leave or update)
2. `src/api/v1alpha1/languageagent_webhook.go` - Synthesis cost estimation logic (should be removed)
   - Lines with: "agent synthesis would exceed cost quota", "Base synthesis cost", etc.
3. `src/controllers/languageagent_controller.go` - Comments about synthesis (should clean up)

**Action items**:
- Remove synthesis cost estimation from webhook validation
- Update comments to remove synthesis references

#### 3. **Re-enable and Fix Tests**

**Disabled test files** (5 files with `.disabled` or `.disabled2` extensions):
1. `src/controllers/languageagent_controller_test.go.disabled` - Main agent controller tests
   - Contains tests for synthesis, AgentVersion creation, hash management
   - Needs major rewrite for container-based model
2. `src/controllers/learning_controller_test.go.disabled` - Learning controller tests
   - Can be deleted (learning removed)
3. `src/controllers/learning_controller_test.go.disabled2` - Duplicate
   - Can be deleted
4. `src/controllers/learning_controller_test.go.backup` - Backup
   - Can be deleted
5. `src/pkg/events/manager_test.go.disabled` - Events manager tests
   - Needs update to remove synthesis/learning event tests
   - Should re-enable core event tests

**Action items**:
- Delete learning controller test files
- Rewrite agent controller tests for container-based model
- Re-enable events manager tests (remove synthesis/learning tests)
- Add new tests for instructions/persona mounting

#### 4. **Update Webhook Validation**

Current webhook (`src/api/v1alpha1/languageagent_webhook.go`) still has:
- Synthesis cost estimation logic
- References to synthesis quota

**What needs to be done**:
- Remove synthesis cost estimation
- Add validation for `Image` field (required, valid format)
- Add validation for `InstructionsFrom` (mutual exclusivity with `Instructions`)
- Update validation logic for container-based model

#### 5. **Documentation Updates** (Phase 4 but worth noting)

**Need to create**:
- `docs/migration-guide.md` - How to migrate from synthesis-based to container-based agents
- `docs/architecture.md` - Updated architecture documentation
- `examples/container-agent/` - Example agent YAML files
- `examples/base-images/` - Example Dockerfiles for Python, Node.js, Go runtimes

### Verification Checklist

- ✅ Build succeeds: `cd src && make build`
- ✅ Core tests pass: `go test ./controllers ./api/... ./pkg/events`
- ✅ CRDs generate: `cd src && make manifests`
- ⚠️ Minimal synthesis references: 8 remaining (mostly comments, some in webhook)
- ⚠️ Minimal learning references: 12 remaining (mostly comments)
- ❓ Persona CRD exists: Not yet deployed to cluster
- ❌ Personas mount correctly: **Not implemented yet** (TODO above)
- ❌ Instructions mount correctly: **Not implemented yet** (TODO above)

### Summary

**Phase 1 is functionally complete** with all major synthesis and learning systems removed. The operator builds successfully and core tests pass. The main work remaining for Phase 2 is:

1. **Critical**: Implement instructions and persona mounting logic
2. **Cleanup**: Remove remaining synthesis references in webhook and comments
3. **Testing**: Re-enable and update test suites
4. **Documentation**: Migration guide and examples

Total lines removed: **~14,000+ lines of code**

The codebase is now significantly simpler and focused on being a pure Kubernetes runtime for AI agents.

---

## Conclusion

This pivot transforms language-operator from a complex "AI system that learns" into a **focused, reliable, Kubernetes-native agentic runtime**. By removing synthesis and learning, we:

1. ✅ Eliminate complex dependencies (LLM synthesis, pattern detection)
2. ✅ Reduce reconciliation latency (no synthesis delays)
3. ✅ Improve reliability (no synthesis failures)
4. ✅ Align with container-native workflows (standard K8s patterns)
5. ✅ Delegate memory/knowledge to specialized tools (mem0, MCP servers)
6. ✅ Focus on core value: **orchestrating agents at scale on Kubernetes**

The result is a simpler, faster, more reliable operator that does one thing exceptionally well: **running AI agents in production Kubernetes clusters**.
