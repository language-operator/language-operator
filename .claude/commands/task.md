# Task Commmand

## Inputs

- :task str - name of task to run
- :persona str (optional) - persona to assume

## Instructions

You have been instructed to read and execute a task.  The task definitions are found in:
- ./requirements/tasks/:task.md, for example "./requirements/tasks/prioritize.md".

## Writing Guidelines

### Avoid unfalsifiable qualifiers

Please avoid the word "comprehensive" and similar unfalsifiable qualifiers like:
- "Complete"
- "Full" 
- "Thorough"
- "Extensive"
- "Robust"

Instead, use specific, verifiable descriptions:
- "Tests X scenarios: A, B, C"
- "Handles 3 error cases: timeout, invalid input, network failure"
- "Validates 5 cron fields: minute, hour, day, month, weekday"

## Precision Guidelines

### 1. Quantify instead of approximating
- ❌ "Several test cases" → ✅ "5 test cases"
- ❌ "Many files" → ✅ "12 files in src/controllers/"
- ❌ "Large performance improvement" → ✅ "Reduced API calls from 100/min to 1/5min"

### 2. Define boundaries explicitly
- ❌ "Handles errors" → ✅ "Handles timeout and 404 errors; does NOT handle auth failures"
- ❌ "Compatible with Kubernetes" → ✅ "Compatible with Kubernetes 1.23-1.28"
- ❌ "Works with all registries" → ✅ "Works with Docker Hub, ECR, GCR; not tested with Harbor"

### 3. Specify failure modes
- ❌ "Graceful error handling" → ✅ "Returns 400 on invalid input, 503 on upstream timeout"
- ❌ "Safe deployment" → ✅ "Zero downtime if rollback within 5 minutes"

### 4. Make assumptions explicit
- ❌ "Should work fine" → ✅ "Assumes kubectl 1.23+, admin permissions, internet access"
- ❌ "Standard configuration" → ✅ "Default namespace, no custom RBAC, 2GB+ memory"

### 5. Define "done" precisely
- ❌ "Fix the validation" → ✅ "Reject cron strings with invalid minute field (>59)"
- ❌ "Improve performance" → ✅ "Reduce reconciliation time from 200ms to <50ms"

### 6. Time-bound statements
- ❌ "Eventually consistent" → ✅ "Consistent within 30 seconds"
- ❌ "Quick to deploy" → ✅ "Deploys in <2 minutes"