# Implementation Plan: Fix Learning System ConfigMap Explosion (Issue #98)

## Problem Summary

The learning system creates 16+ ConfigMaps and excessive LLM calls when `runsPendingLearning` reaches threshold (10 runs). Root cause: **two separate, conflicting learning systems run simultaneously**.

## Root Cause Analysis

### Two Learning Paths

**Path 1: Threshold-Based Learning** ([learning_controller.go:223-244](../src/controllers/learning_controller.go#L223-L244))
- Trigger: `runsPendingLearning >= 10`
- Action: Calls `triggerOptimization()` → creates `LanguageAgentVersion` resource
- Result: Single version resource with merged optimized code

**Path 2: Event-Based Learning** ([learning_controller.go:640-690](../src/controllers/learning_controller.go#L640-L690))
- Trigger: Trace analysis finds patterns
- Action: Creates `LearningEvent` for each task → calls `processLearningTrigger()` → creates individual ConfigMap per task
- Result: Multiple ConfigMaps (s003-v1, s003-v2, s003-v3...)

### The Explosion

When threshold hits 10:
1. Path 1 triggers `triggerOptimization()`
2. Path 1 calls `identifyTasksForOptimization()` → returns 2 hardcoded tasks
3. **For each task**: calls `synthesizer.SynthesizeTask()` (expensive LLM call)
4. Path 2 ALSO triggers for same tasks (from trace analysis)
5. Each Path 2 trigger creates separate ConfigMap via `processLearningTrigger()` → `CreateVersionedConfigMap()`
6. Result: 16+ ConfigMaps instead of 1 version resource

### Specific Issues

1. **Hardcoded Task List** ([learning_controller.go:3767-3816](../src/controllers/learning_controller.go#L3767-L3816))
   - `identifyTasksForOptimization()` returns hardcoded "read_existing_story" and "append_to_story"
   - Creates multiple LLM calls for every learning cycle
   - Should query actual telemetry data, not return hardcoded tasks

2. **No Content Deduplication** ([learning_controller.go:1096](../src/controllers/learning_controller.go#L1096))
   - Always increments version: `newVersion := taskStatus.CurrentVersion + 1`
   - No check if generated code is identical to existing ConfigMap

3. **Dual System Conflict**
   - Threshold-based creates `LanguageAgentVersion` (correct)
   - Event-based creates individual ConfigMaps (legacy, incorrect)
   - Both run simultaneously

## Solution: Remove Event-Based Path Entirely

### Design Decision: Single Learning Path via LanguageAgentVersion

**Keep**: Threshold-based learning (Path 1) - creates `LanguageAgentVersion` resources
**Delete**: Event-based learning (Path 2) - all functions, all code, no backwards compatibility

The `LanguageAgentVersion` resource is the correct pattern:
- Atomic versioning (all optimized tasks in single resource)
- Proper Kubernetes resource lifecycle
- Already implemented in Path 1
- Aligns with DSL v1 proposal architecture

**Aggressive Cleanup**: Since unreleased, delete all event-based code completely.

### Implementation Steps

#### Step 1: Delete Event-Based Learning Functions Entirely
**File**: [src/controllers/learning_controller.go](../src/controllers/learning_controller.go)

**Delete These Functions** (17 functions, ~3000 lines):

**Event Generation & Processing** (Core of event-based path):
1. `checkLearningTriggers()` (line 596) - Generates learning events from trace analysis
2. `checkErrorTriggers()` (line 698) - Generates error-based learning events
3. `processLearningTrigger()` (line 1030) - Creates individual ConfigMaps per task
4. `generateLearnedCode()` (line 1209) - Generates code for single task
5. `recordLearningEvent()` (line 2362) - Records events for event-based system

**Error Analysis** (Only used by event-based path):
6. `getTaskFailures()` (line 765) - Extracts failures from events
7. `getAgentEvents()` (line 806) - Queries Kubernetes events
8. `isAgentRelatedEvent()` (line 838) - Filters events
9. `parseTaskFailureFromEvent()` (line 858) - Parses event messages
10. `extractTaskNameFromEvent()` (line 902) - Extracts task names from events
11. `updateConsecutiveFailures()` (line 936) - Tracks failure counts
12. `shouldTriggerErrorResynthesis()` (line 979) - Determines if error trigger needed
13. `calculateRecentErrorRate()` (line 1004) - Calculates error rates
14. `buildErrorContext()` (line 1317) - Builds error context for synthesis
15. `analyzeErrorPatterns()` (line 1382) - Analyzes error patterns
16. `generatePatternBasedCode()` (line 1429) - Pattern-based code generation

**Deployment Updates** (Event-based path updated deployments directly):
17. `updateDeployment()` (line 1472) - Updates deployment for single task
18. `findAgentDeployment()` (line 1552) - Finds deployment to update
19. `updateAlternativeWorkload()` (line 1589) - Updates CronJob/Job/DaemonSet
20. `updateCronJobConfigMap()` (line 1648) - Updates CronJob ConfigMap ref
21. `updateJobConfigMap()` (line 1696) - Updates Job ConfigMap ref
22. `updateDaemonSetConfigMap()` (line 1744) - Updates DaemonSet ConfigMap ref
23. `extractCronJobConfigMapReference()` (line 1794) - Extracts ConfigMap name
24. `patchCronJobConfigMap()` (line 1815) - Patches CronJob ConfigMap
25. `extractJobConfigMapReference()` (line 1847) - Extracts Job ConfigMap
26. `patchJobConfigMap()` (line 1868) - Patches Job ConfigMap
27. `extractDaemonSetConfigMapReference()` (line 1900) - Extracts DaemonSet ConfigMap
28. `patchDaemonSetConfigMap()` (line 1921) - Patches DaemonSet ConfigMap
29. `extractConfigMapReference()` (line 1953) - Extracts Deployment ConfigMap
30. `patchDeploymentConfigMap()` (line 1975) - Patches Deployment ConfigMap
31. `waitForDeploymentRollout()` (line 2020) - Waits for rollout
32. `rollbackDeployment()` (line 2067) - Rolls back deployment
33. `verifyDeploymentHealth()` (line 2080) - Verifies deployment health

**Delete These Types**:
- `LearningEvent` struct
- `TaskFailure` struct
- `PatternAnalysis` struct (if only used by deleted functions)

**Total Deletion**: ~33 functions, ~3500 lines of code

#### Step 2: Fix identifyTasksForOptimization() Hardcoded Tasks
**File**: [src/controllers/learning_controller.go](../src/controllers/learning_controller.go)
**Lines**: 3767-3816

**Current (Hardcoded)**:
```go
tasks = append(tasks, synthesis.TaskSynthesisRequest{
    TaskName:     "read_existing_story",  // HARDCODED
    Instructions: "Read the existing story file and return its current content",
    // ...
})
```

**Replace With (Query-Based)**:
```go
func (r *LearningReconciler) identifyTasksForOptimization(ctx context.Context, agent *langopv1alpha1.LanguageAgent) ([]synthesis.TaskSynthesisRequest, error) {
    // Query learning status for tasks with sufficient traces
    learningStatus, err := r.getLearningStatus(ctx, agent)
    if err != nil {
        return nil, fmt.Errorf("failed to get learning status: %w", err)
    }

    var tasks []synthesis.TaskSynthesisRequest

    // Identify tasks that meet optimization criteria
    for taskName, status := range learningStatus {
        // Skip if not enough traces
        if status.TraceCount < r.LearningThreshold {
            continue
        }

        // Skip if already optimized recently
        if time.Since(status.LastLearningAttempt) < r.LearningInterval {
            continue
        }

        // Skip if pattern confidence too low
        if status.PatternConfidence < r.PatternConfidenceMin {
            continue
        }

        // Extract task schema from current agent code
        taskSchema, err := r.extractTaskSchema(ctx, agent, taskName)
        if err != nil {
            r.Log.Error(err, "Failed to extract task schema", "task", taskName)
            continue
        }

        // Get trace data for this task
        traces, err := r.getTaskTraces(ctx, agent, taskName)
        if err != nil {
            r.Log.Error(err, "Failed to get traces for task", "task", taskName)
            continue
        }

        // Build synthesis request from actual data
        tasks = append(tasks, synthesis.TaskSynthesisRequest{
            TaskName:           taskName,
            Instructions:       taskSchema.Instructions,
            Inputs:             taskSchema.Inputs,
            Outputs:            taskSchema.Outputs,
            TaskCode:           taskSchema.CurrentCode,
            Traces:             formatTracesForSynthesis(traces),
            TraceCount:         int(status.TraceCount),
            CommonPattern:      status.CommonPattern,
            ConsistencyScore:   int(status.PatternConfidence * 100),
            UniquePatternCount: status.UniquePatternCount,
            ToolsList:          extractToolsFromTraces(traces),
            AgentName:          agent.Name,
            Namespace:          agent.Namespace,
        })
    }

    return tasks, nil
}
```

**New Helper Functions Required**:
- `extractTaskSchema()` - Parse agent code to get task definition
- `getTaskTraces()` - Query telemetry backend for task execution traces
- `formatTracesForSynthesis()` - Convert traces to synthesis-friendly format
- `extractToolsFromTraces()` - Extract tool names from trace data

#### Step 3: Add Content Deduplication (Future Optimization)
**File**: [src/pkg/synthesis/configmap.go](../src/pkg/synthesis/configmap.go)

**Add before creating new version**:
```go
// Check if code content has actually changed
existingCM, err := r.getLatestConfigMap(ctx, agent)
if err == nil {
    existingCode := existingCM.Data["agent.rb"]
    if existingCode == configMapOptions.Code {
        // Code unchanged, skip version creation
        return existingCM, nil
    }
}
```

**Note**: This is a performance optimization for Step 4, not critical for fixing the explosion.

#### Step 4: Update Reconciliation Logic
**File**: [src/controllers/learning_controller.go](../src/controllers/learning_controller.go)
**Lines**: 140-250 (Reconcile function)

**Keep**: Threshold-based trigger (lines 223-244)
**Remove**: Event-based trigger processing

**Current Flow**:
```
Reconcile()
  → Check threshold (runsPendingLearning >= 10)
  → triggerOptimization()
  → identifyTasksForOptimization() [HARDCODED]
  → SynthesizeTask() for each
  → createAgentVersion() [CORRECT]

  [SEPARATELY]
  → checkLearningTriggers() [GENERATES EVENTS]
  → processLearningTrigger() for each event [CREATES CONFIGMAPS - WRONG]
```

**New Flow**:
```
Reconcile()
  → Check threshold (runsPendingLearning >= 10)
  → triggerOptimization()
  → identifyTasksForOptimization() [QUERY-BASED, NOT HARDCODED]
  → SynthesizeTask() for each identified task
  → createAgentVersion() [SINGLE VERSION RESOURCE]
  → Reset counter

  [NO MORE EVENT-BASED CONFIGMAP CREATION]
```

## Implementation Order

1. ✅ **Step 1**: Delete event-based learning path entirely (stops explosion, major cleanup)
2. ✅ **Step 2**: Fix hardcoded task identification (enables real optimization)
3. ✅ **Step 3**: Simplify reconciliation logic (remove dead code after Step 1)
4. ⏭️ **Step 4**: Add content deduplication (future optimization, not critical)

## Testing Strategy

### Unit Tests
```go
func TestIdentifyTasksForOptimization_NoHardcodedTasks(t *testing.T) {
    // Verify function queries learning status, not hardcoded values
    // Assert: tasks returned match learning status data
}

func TestTriggerOptimization_SingleVersionCreated(t *testing.T) {
    // Simulate 10 runs
    // Trigger optimization
    // Assert: exactly 1 LanguageAgentVersion created
    // Assert: 0 individual ConfigMaps created
}

func TestNoEventBasedConfigMapCreation(t *testing.T) {
    // Verify processLearningTrigger is NOT called
    // Assert: only LanguageAgentVersion resources created
}
```

### Integration Tests
1. Deploy agent s003 (simple story agent)
2. Run 10 times to trigger learning
3. Verify:
   - Exactly 1 `LanguageAgentVersion` resource created
   - 0 individual ConfigMaps created (s003-v1, s003-v2, etc.)
   - Counter reset to 0 after optimization
   - No "OptimizationTriggered after 0 runs" events

### Manual Testing
```bash
# 1. Deploy test agent
kubectl apply -f examples/simple-agent/agent.yaml

# 2. Monitor learning
watch -n 1 'kubectl get languageagentversions,configmaps -l langop.io/agent=s003'

# 3. Trigger 10 runs
for i in {1..10}; do
  kubectl create job test-run-$i --from=cronjob/s003
  sleep 30
done

# 4. Verify single version created
kubectl get languageagentversions -l langop.io/agent=s003
# Expected: s003-v1 only

# 5. Verify no ConfigMap explosion
kubectl get configmaps -l langop.io/agent=s003
# Expected: s003-learning-status only (no s003-v1, s003-v2, etc.)
```

## Success Criteria

- ✅ **Functional**: Learning happens exactly once when threshold (10 runs) is reached
- ✅ **Resource Usage**: Single `LanguageAgentVersion` created, not 16+ ConfigMaps
- ✅ **Cost**: 2-3 LLM calls per learning cycle (not 16+)
  - 1 call per optimizable task (identified from traces)
  - Tasks query actual telemetry data, not hardcoded
- ✅ **Counter Reset**: `runsPendingLearning` resets to 0 after successful optimization
- ✅ **No Recursion**: No "OptimizationTriggered after 0 runs" events

## Files Modified

1. [src/controllers/learning_controller.go](../src/controllers/learning_controller.go)
   - Remove `processLearningTrigger()` (lines 1029-1200)
   - Rewrite `identifyTasksForOptimization()` (lines 3767-3816)
   - Add helper functions: `extractTaskSchema()`, `getTaskTraces()`, etc.

2. [src/controllers/learning_controller_test.go](../src/controllers/learning_controller_test.go)
   - Add tests for query-based task identification
   - Add tests for single version creation
   - Add tests verifying no ConfigMap explosion

## Risk Assessment

**Low Risk**:
- Removing unused path (event-based ConfigMap creation) has no impact
- Threshold-based path (LanguageAgentVersion) already works
- Query-based task identification is more correct than hardcoded

**Mitigation**:
- Comprehensive unit tests before deployment
- Integration tests with real agent (s003)
- Manual verification of learning behavior
- Can rollback by reverting commits if issues arise

## Timeline (Aggressive Cleanup)

- **Planning**: 30 minutes (read code, write plan)
- **Implementation**: 3 hours
  - Step 1: 1 hour (delete event-based path - straightforward deletion)
  - Step 2: 1.5 hours (implement query-based task identification)
  - Step 3: 30 minutes (simplify reconciliation after deletions)
- **Testing**: 1.5 hours
  - Unit tests: 45 minutes (fewer functions to test after deletions)
  - Integration tests: 45 minutes
- **Total**: 5 hours (faster due to aggressive cleanup, no backwards compatibility)

## Dependencies

- ✅ Telemetry adapter (SigNoz) working (resolved in #87)
- ✅ Event-driven learning data collection (resolved in #88)
- ✅ Learning status tracking infrastructure exists
- ⚠️ Need to implement trace query functions

## Open Questions

1. **Q**: Should we delete existing individual ConfigMaps (s003-v1, s003-v2, etc.)?
   **A**: Yes, add cleanup in reconciliation to remove legacy ConfigMaps.

2. **Q**: What if `identifyTasksForOptimization()` finds 0 optimizable tasks?
   **A**: Already handled in `triggerOptimization()` lines 275-282 - logs warning and resets counter.

3. **Q**: How to extract task schema from current agent code?
   **A**: Parse Ruby DSL using existing AST validation logic in Ruby gem.

## Notes

- This fixes the **root cause**, not symptoms
- Aligns with DSL v1 organic functions architecture
- Simplifies learning system (single path instead of dual paths)
- Reduces cost (fewer redundant LLM calls)
- Improves reliability (atomic version updates instead of fragmented ConfigMaps)
