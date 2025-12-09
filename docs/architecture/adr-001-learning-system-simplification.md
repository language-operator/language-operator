# ADR-001: Learning System Simplification

## Status

Accepted (December 2025)

## Context

The learning system initially implemented two concurrent learning paths that created complexity and operational issues:

1. **Threshold-Based Learning**: Triggered when `runsPendingLearning >= 10`, created `LanguageAgentVersion` resources
2. **Event-Based Learning**: Watched Kubernetes Events and Jobs, created individual ConfigMaps per task

### Problems with Dual-Path Architecture

**ConfigMap Explosion** (Issue #98):
- When threshold reached 10 runs, both systems triggered simultaneously
- Event-based path created 16+ ConfigMaps (s003-v1, s003-v2, s003-v3...)
- Threshold-based path created 1 `LanguageAgentVersion` resource
- Resulted in redundant LLM calls (16+ instead of 2-3 per cycle)

**Complexity**:
- ~1200 lines of event processing code
- ConfigMap size management (800KB limit, cleanup, rotation)
- Trace summarization to fit ConfigMap constraints
- Duplicate prevention tracking (annotations on Events/Jobs)
- Event and Job watches adding reconciliation overhead

**Hardcoded Task Identification**:
- `identifyTasksForOptimization()` returned hardcoded task list
- Did not query actual telemetry data
- Made optimization ineffective (found zero tasks)

## Decision

**Simplified to single threshold-based learning path using direct SigNoz queries.**

### Architecture Changes

**Before**:
```
When runsPendingLearning >= 10:
  1. Threshold path: identifyTasksForOptimization() [hardcoded tasks]
  2. Event path: watch Events → processTaskCompletedEvent() → create ConfigMaps
  3. Both paths trigger LLM synthesis
  4. Result: 16+ ConfigMaps + 1 LanguageAgentVersion
```

**After**:
```
When runsPendingLearning >= 10:
  1. Query SigNoz for execution traces (last 24 hours)
  2. Group traces by task name
  3. Calculate pattern confidence inline from traces
  4. Synthesize eligible tasks
  5. Create single LanguageAgentVersion resource
```

### Implementation (4 Phases)

**Phase 1: Revert Event-Based Data Collection** (Issue #99)
- Removed `processAgentExecutions()` call from Reconcile loop
- Prepared codebase for direct query approach

**Phase 2: Direct SigNoz Queries** (Issue #100)
- Rewrote `identifyTasksForOptimization()` to query SigNoz first
- Calculate pattern confidence from traces (not ConfigMap)
- Eliminated ConfigMap dependency

**Phase 3: Delete Event Processing Code** (Issue #101)
- Deleted 23 functions (~1190 lines, 38% reduction)
- Removed 3 structs: `TaskLearningStatus`, `AgentExecutionSummary`, `TaskExecutionStatus`
- Removed Event and Job watches from SetupWithManager
- Removed unused imports: `batchv1`, `builder`, `handler`, `predicate`, `reconcile`

**Phase 4: Cleanup and Documentation** (Issue #102)
- Removed 4 unused fields from `LearningReconciler` struct
- Removed `pkg/learning` import
- Created this ADR

## Consequences

### Positive

1. **Simpler Architecture**
   - Single learning path (threshold-based only)
   - 38% less code (3118 → 1928 lines in `learning_controller.go`)
   - No ConfigMap persistence layer
   - No event watching overhead

2. **Better Data Quality**
   - Real-time data from SigNoz (not cached in ConfigMaps)
   - Full trace data for synthesis (no summarization)
   - Query-based task identification (not hardcoded)

3. **Cost Reduction**
   - 2-3 LLM calls per learning cycle (down from 16+)
   - Only synthesize tasks with sufficient traces and confidence

4. **Operational Simplicity**
   - No ConfigMap size management
   - No trace summarization complexity
   - Easier to debug (single data source)

5. **Alignment**
   - Matches Ruby gem's proven direct query approach
   - Follows Kubernetes best practices (CRDs over ConfigMaps for versioning)

### Negative

1. **SigNoz Dependency**
   - Requires SigNoz to be available for learning
   - If SigNoz down, learning paused (not critical - agents still run)

2. **Query Performance**
   - Direct SigNoz queries add latency to optimization trigger
   - Mitigated by: only runs when threshold reached (not every reconcile)

### Neutral

1. **No Breaking Changes**
   - System unreleased, no backwards compatibility needed
   - Old ConfigMaps can be manually cleaned up if present

## References

- Issue #98: Learning System ConfigMap Explosion
- Issue #99: Revert event-based learning data collection
- Issue #100: Replace ConfigMap-based task identification with direct SigNoz queries
- Issue #101: Delete event watching, job tracking, and ConfigMap management code
- Issue #102: Clean up artifacts and document learning system simplification
- Ruby Gem Reference: `language-operator-gem` learning implementation
- DSL v1 Proposal: Organic functions architecture

## Decision Makers

- James Ryan (Go Engineer)
- Claude Sonnet 4.5 (AI Pair Programmer)

## Date

December 9, 2025
