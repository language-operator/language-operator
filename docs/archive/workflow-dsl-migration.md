# Workflow DSL Migration Archive

## Overview

This directory contains archived files and documentation from the DSL v1 migration that removed the old workflow/step model in favor of the task/main model.

## Archived Files

- `synthesizer.go.backup` - Previous version of the synthesizer with workflow-related template logic

## Migration Summary

As part of Issue #27 (Remove old workflow synthesis templates), the following changes were made:

### Test Files Updated

1. **template_test.go**:
   - Removed `TestTemplateGeneratesValidWorkflow` function
   - Added `TestTemplateGeneratesValidTaskMain` function
   - Updated DSL method lists to use DSL v1 patterns (task, main, execute_task, etc.)

2. **schema_test.go**:
   - Updated test cases to use task/main DSL v1 syntax instead of workflow/step
   - Replaced workflow test examples with organic function patterns

3. **ruby_validator_test.go**:
   - Updated agent test cases to use task/main model
   - Updated performance test to use DSL v1 syntax with symbolic task implementations

### Deprecated Concepts Removed

- `workflow` block definitions
- `step` directive usage in test code  
- `depends_on` references in method lists
- `objectives` array pattern (replaced with `instructions`)

### DSL v1 Concepts Introduced in Tests

- `task` definitions with inputs/outputs schemas
- `main` blocks with imperative control flow
- `execute_task` function calls
- `instructions` for natural language descriptions
- Organic function patterns (neural and symbolic task implementations)

## Historical Context

The workflow/step model was part of DSL v0 and has been superseded by DSL v1's organic function model as described in `requirements/proposals/dsl-v1.md`. The task/main model provides:

- Better contract stability through input/output schemas
- Progressive synthesis from neural to symbolic implementations
- Cleaner imperative control flow
- Improved learning capabilities

## Reference

For current DSL documentation, see:
- DSL v1 proposal: `requirements/proposals/dsl-v1.md` 
- Current synthesis templates in `src/pkg/synthesis/`
- Language operator gem documentation

---

*Archived during Issue #27 implementation - November 2025*