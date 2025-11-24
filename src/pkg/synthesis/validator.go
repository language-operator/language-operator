package synthesis

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-logr/logr"
)

// TaskValidationError represents a validation error for task-based agents
type TaskValidationError struct {
	Type     string `json:"type"`
	Task     string `json:"task,omitempty"`
	Field    string `json:"field,omitempty"`
	Line     int    `json:"line,omitempty"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // "error", "warning", "info"
}

// TaskValidator provides validation for DSL v1 task-based agents
type TaskValidator struct {
	logger logr.Logger
}

// NewTaskValidator creates a new task validator
func NewTaskValidator(logger logr.Logger) *TaskValidator {
	return &TaskValidator{
		logger: logger,
	}
}

// ValidateTaskAgent performs comprehensive validation of a task-based agent
func (v *TaskValidator) ValidateTaskAgent(ctx context.Context, code string) ([]TaskValidationError, error) {
	var errors []TaskValidationError

	// Parse the agent DSL code to extract structure
	agent, err := v.ParseAgentStructure(code)
	if err != nil {
		return nil, fmt.Errorf("failed to parse agent structure: %w", err)
	}

	// Validate task definitions
	taskErrors := v.validateTaskDefinitions(agent)
	errors = append(errors, taskErrors...)

	// Validate main block
	mainErrors := v.validateMainBlock(agent)
	errors = append(errors, mainErrors...)

	// Validate type consistency
	typeErrors := v.validateTypeConsistency(agent)
	errors = append(errors, typeErrors...)

	// Validate symbolic task safety
	safetyErrors := v.validateSymbolicTaskSafety(agent)
	errors = append(errors, safetyErrors...)

	// Run existing schema validation as well
	schemaViolations, err := ValidateGeneratedCodeAgainstSchema(ctx, code)
	if err != nil {
		v.logger.Info("Schema validation failed", "error", err.Error())
		// Don't fail validation if schema validation has issues - it's supplemental
	} else {
		// Convert schema violations to task validation errors
		for _, violation := range schemaViolations {
			errors = append(errors, TaskValidationError{
				Type:     "schema",
				Field:    violation.Property,
				Line:     violation.Location,
				Message:  violation.Message,
				Severity: "error",
			})
		}
	}

	return errors, nil
}

// AgentStructure represents the parsed structure of an agent
type AgentStructure struct {
	Name        string
	Description string
	Mode        string
	Tasks       map[string]*TaskDefinition
	MainBlock   *MainBlock
}

// TaskDefinition represents a task definition with type information
type TaskDefinition struct {
	Name         string
	Instructions string
	Inputs       map[string]string // input_name -> type
	Outputs      map[string]string // output_name -> type
	IsSymbolic   bool              // true if it has a do |inputs| block
	CodeBlock    string            // the actual Ruby code for symbolic tasks
	Line         int
}

// MainBlock represents the main execution block
type MainBlock struct {
	TaskCalls   []TaskCall
	ReturnValue string
	CodeBlock   string
	Line        int
}

// TaskCall represents a call to a task
type TaskCall struct {
	TaskName string
	Inputs   map[string]string // input_name -> variable/value
	Variable string            // variable name where result is stored
	Line     int
}

// parseAgentStructure extracts the agent structure from DSL code
func (v *TaskValidator) ParseAgentStructure(code string) (*AgentStructure, error) {
	agent := &AgentStructure{
		Tasks: make(map[string]*TaskDefinition),
	}

	lines := strings.Split(code, "\n")

	// Extract agent name and basic info
	agentNameRegex := regexp.MustCompile(`agent\s+"([^"]+)"`)
	if matches := agentNameRegex.FindStringSubmatch(code); len(matches) > 1 {
		agent.Name = matches[1]
	}

	// Extract description
	descRegex := regexp.MustCompile(`description\s+"([^"]+)"`)
	if matches := descRegex.FindStringSubmatch(code); len(matches) > 1 {
		agent.Description = matches[1]
	}

	// Extract mode
	modeRegex := regexp.MustCompile(`mode\s+:(\w+)`)
	if matches := modeRegex.FindStringSubmatch(code); len(matches) > 1 {
		agent.Mode = matches[1]
	}

	// Parse tasks
	err := v.parseTaskDefinitions(code, lines, agent)
	if err != nil {
		return nil, err
	}

	// Parse main block
	err = v.parseMainBlock(code, lines, agent)
	if err != nil {
		return nil, err
	}

	return agent, nil
}

// parseTaskDefinitions extracts task definitions from the code
func (v *TaskValidator) parseTaskDefinitions(code string, lines []string, agent *AgentStructure) error {
	// More robust task parsing - handle multi-line task definitions
	
	// First, find all task blocks (both neural and symbolic)
	taskBlocks := v.extractTaskBlocks(code)
	
	for _, block := range taskBlocks {
		task := v.parseTaskBlock(block, lines)
		if task != nil {
			agent.Tasks[task.Name] = task
		}
	}

	return nil
}

// extractTaskBlocks finds all task definitions in the code
func (v *TaskValidator) extractTaskBlocks(code string) []string {
	var blocks []string
	
	// Find task statements using a simple approach
	lines := strings.Split(code, "\n")
	var currentBlock []string
	inTaskBlock := false
	blockDepth := 0
	
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		
		// Check if this line starts a task
		if strings.HasPrefix(trimmedLine, "task :") {
			// If we were already in a block, save it
			if inTaskBlock && len(currentBlock) > 0 {
				blocks = append(blocks, strings.Join(currentBlock, "\n"))
			}
			// Start new block
			currentBlock = []string{line}
			inTaskBlock = true
			blockDepth = 0
			// Check if this line contains " do"
			if strings.Contains(line, " do") {
				blockDepth = 1
			}
			continue
		}
		
		if inTaskBlock {
			currentBlock = append(currentBlock, line)
			
			// Count do/end pairs for symbolic tasks
			if strings.Contains(trimmedLine, " do") {
				blockDepth++
			}
			if strings.Contains(trimmedLine, "end") {
				blockDepth--
				if blockDepth <= 0 {
					// End of task block
					blocks = append(blocks, strings.Join(currentBlock, "\n"))
					currentBlock = nil
					inTaskBlock = false
				}
			}
			
			// For neural tasks (no do block), end when we hit another task or main
			if blockDepth == 0 && (strings.HasPrefix(trimmedLine, "task :") || 
				strings.HasPrefix(trimmedLine, "main ") ||
				strings.HasPrefix(trimmedLine, "end") ||
				trimmedLine == "") {
				// End of neural task
				if !strings.HasPrefix(trimmedLine, "task :") && len(currentBlock) > 1 { // Don't include next task in this block
					currentBlock = currentBlock[:len(currentBlock)-1]
				}
				if len(currentBlock) > 0 {
					blocks = append(blocks, strings.Join(currentBlock, "\n"))
				}
				currentBlock = nil
				inTaskBlock = false
			}
		}
	}
	
	// Don't forget the last block
	if inTaskBlock && len(currentBlock) > 0 {
		blocks = append(blocks, strings.Join(currentBlock, "\n"))
	}
	
	return blocks
}

// parseTaskBlock parses a single task block
func (v *TaskValidator) parseTaskBlock(block string, lines []string) *TaskDefinition {
	// Extract task name
	taskNameRegex := regexp.MustCompile(`task\s+:(\w+)`)
	nameMatch := taskNameRegex.FindStringSubmatch(block)
	if len(nameMatch) < 2 {
		return nil
	}
	taskName := nameMatch[1]
	
	// Extract instructions
	instructionsRegex := regexp.MustCompile(`instructions:\s*["']([^"']+)["']`)
	instructions := ""
	if instrMatch := instructionsRegex.FindStringSubmatch(block); len(instrMatch) > 1 {
		instructions = instrMatch[1]
	}
	
	// Extract inputs
	inputsRegex := regexp.MustCompile(`inputs:\s*\{([^}]*)\}`)
	inputs := make(map[string]string)
	if inputMatch := inputsRegex.FindStringSubmatch(block); len(inputMatch) > 1 {
		inputs = parseTypeHash(inputMatch[1])
	}
	
	// Extract outputs
	outputsRegex := regexp.MustCompile(`outputs:\s*\{([^}]*)\}`)
	outputs := make(map[string]string)
	if outputMatch := outputsRegex.FindStringSubmatch(block); len(outputMatch) > 1 {
		outputs = parseTypeHash(outputMatch[1])
	}
	
	// Check if it's a symbolic task (has do |inputs| block)
	isSymbolic := strings.Contains(block, " do |")
	codeBlock := ""
	if isSymbolic {
		// Extract the code between do |inputs| and end
		doRegex := regexp.MustCompile(`(?s)do\s*\|[^|]*\|\s*(.+?)\s*end`)
		if codeMatch := doRegex.FindStringSubmatch(block); len(codeMatch) > 1 {
			codeBlock = strings.TrimSpace(codeMatch[1])
		}
	}
	
	// Find line number
	lineNum := 0
	for i, line := range lines {
		if strings.Contains(line, "task :"+taskName) {
			lineNum = i + 1
			break
		}
	}
	
	return &TaskDefinition{
		Name:         taskName,
		Instructions: instructions,
		Inputs:       inputs,
		Outputs:      outputs,
		IsSymbolic:   isSymbolic,
		CodeBlock:    codeBlock,
		Line:         lineNum,
	}
}

// parseMainBlock extracts the main block from the code
func (v *TaskValidator) parseMainBlock(code string, lines []string, agent *AgentStructure) error {
	// Find main block using line-by-line parsing
	codeLines := strings.Split(code, "\n")
	var mainBlockLines []string
	inMainBlock := false
	blockDepth := 0
	lineNum := 0
	
	for i, line := range codeLines {
		trimmedLine := strings.TrimSpace(line)
		
		// Check if this line starts a main block
		if strings.HasPrefix(trimmedLine, "main do") || strings.Contains(trimmedLine, "main do |") {
			inMainBlock = true
			blockDepth = 1
			lineNum = i + 1
			continue // Don't include the "main do" line itself
		}
		
		if inMainBlock {
			// Count do/end pairs
			if strings.Contains(trimmedLine, " do") {
				blockDepth++
			}
			if strings.Contains(trimmedLine, "end") {
				blockDepth--
				if blockDepth <= 0 {
					// End of main block
					break
				}
			}
			mainBlockLines = append(mainBlockLines, line)
		}
	}
	
	if !inMainBlock {
		return nil // No main block found
	}

	mainBlock := &MainBlock{
		CodeBlock: strings.Join(mainBlockLines, "\n"),
		Line:      lineNum,
	}

	// Parse task calls from main block
	v.parseTaskCalls(mainBlock.CodeBlock, mainBlock)

	agent.MainBlock = mainBlock
	return nil
}

// parseTaskCalls extracts task calls from a code block
func (v *TaskValidator) parseTaskCalls(codeBlock string, mainBlock *MainBlock) {
	// Find execute_task calls
	taskCallRegex := regexp.MustCompile(`(\w+)\s*=\s*execute_task\(\s*:(\w+)(?:,\s*inputs:\s*\{([^}]*)\})?\s*\)`)
	matches := taskCallRegex.FindAllStringSubmatch(codeBlock, -1)
	
	for _, match := range matches {
		variable := match[1]
		taskName := match[2]
		inputsStr := match[3]

		taskCall := TaskCall{
			TaskName: taskName,
			Variable: variable,
			Inputs:   parseInputs(inputsStr),
		}

		mainBlock.TaskCalls = append(mainBlock.TaskCalls, taskCall)
	}
}

// parseTypeHash parses type definitions like "name: 'string', count: 'integer'"
func parseTypeHash(str string) map[string]string {
	result := make(map[string]string)
	if str == "" {
		return result
	}

	// Parse key-value pairs
	pairs := strings.Split(str, ",")
	for _, pair := range pairs {
		parts := strings.Split(strings.TrimSpace(pair), ":")
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.Trim(strings.TrimSpace(parts[1]), `'"`)
			result[key] = value
		}
	}
	return result
}

// parseInputs parses input assignments like "name: variable, count: 42"
func parseInputs(str string) map[string]string {
	result := make(map[string]string)
	if str == "" {
		return result
	}

	pairs := strings.Split(str, ",")
	for _, pair := range pairs {
		parts := strings.Split(strings.TrimSpace(pair), ":")
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			result[key] = value
		}
	}
	return result
}

// validateTaskDefinitions validates all task definitions
func (v *TaskValidator) validateTaskDefinitions(agent *AgentStructure) []TaskValidationError {
	var errors []TaskValidationError

	for taskName, task := range agent.Tasks {
		// Validate task has either instructions OR symbolic code block
		if task.Instructions == "" && !task.IsSymbolic {
			errors = append(errors, TaskValidationError{
				Type:     "task_definition",
				Task:     taskName,
				Line:     task.Line,
				Message:  "Task must have either 'instructions' (neural) or code block (symbolic)",
				Severity: "error",
			})
		}

		// Validate inputs and outputs are defined
		if len(task.Inputs) == 0 && len(task.Outputs) == 0 {
			errors = append(errors, TaskValidationError{
				Type:     "task_definition",
				Task:     taskName,
				Line:     task.Line,
				Message:  "Task must define inputs and/or outputs schema",
				Severity: "warning",
			})
		}

		// Validate type names are valid
		for inputName, inputType := range task.Inputs {
			if !isValidDSLType(inputType) {
				errors = append(errors, TaskValidationError{
					Type:     "type_definition",
					Task:     taskName,
					Field:    inputName,
					Line:     task.Line,
					Message:  fmt.Sprintf("Invalid input type '%s' for field '%s'. Valid types: string, integer, number, boolean, array, hash, any", inputType, inputName),
					Severity: "error",
				})
			}
		}

		for outputName, outputType := range task.Outputs {
			if !isValidDSLType(outputType) {
				errors = append(errors, TaskValidationError{
					Type:     "type_definition",
					Task:     taskName,
					Field:    outputName,
					Line:     task.Line,
					Message:  fmt.Sprintf("Invalid output type '%s' for field '%s'. Valid types: string, integer, number, boolean, array, hash, any", outputType, outputName),
					Severity: "error",
				})
			}
		}
	}

	return errors
}

// validateMainBlock validates the main execution block
func (v *TaskValidator) validateMainBlock(agent *AgentStructure) []TaskValidationError {
	var errors []TaskValidationError

	if agent.MainBlock == nil {
		errors = append(errors, TaskValidationError{
			Type:     "main_block",
			Message:  "Agent must have a 'main' block to define execution flow",
			Severity: "error",
		})
		return errors
	}

	// Validate that all called tasks are defined
	for _, taskCall := range agent.MainBlock.TaskCalls {
		if _, exists := agent.Tasks[taskCall.TaskName]; !exists {
			errors = append(errors, TaskValidationError{
				Type:     "task_call",
				Task:     taskCall.TaskName,
				Line:     agent.MainBlock.Line,
				Message:  fmt.Sprintf("Called task '%s' is not defined", taskCall.TaskName),
				Severity: "error",
			})
		}
	}

	return errors
}

// validateTypeConsistency validates type consistency across task calls
func (v *TaskValidator) validateTypeConsistency(agent *AgentStructure) []TaskValidationError {
	var errors []TaskValidationError

	if agent.MainBlock == nil {
		return errors
	}

	// Track variable types as they flow through the main block
	variableTypes := make(map[string]map[string]string)

	for _, taskCall := range agent.MainBlock.TaskCalls {
		task, exists := agent.Tasks[taskCall.TaskName]
		if !exists {
			continue // This error is caught elsewhere
		}

		// Validate input types match expected
		for inputName, inputValue := range taskCall.Inputs {
			_, exists := task.Inputs[inputName]
			if !exists {
				errors = append(errors, TaskValidationError{
					Type:     "type_mismatch",
					Task:     taskCall.TaskName,
					Field:    inputName,
					Line:     agent.MainBlock.Line,
					Message:  fmt.Sprintf("Input '%s' not defined in task '%s'", inputName, taskCall.TaskName),
					Severity: "error",
				})
				continue
			}

			// Check if input comes from a variable (previous task output)
			if strings.Contains(inputValue, "[:") {
				// This looks like variable[:field] access - validate the field exists
				v.validateVariableAccess(inputValue, variableTypes, &errors, taskCall.TaskName, inputName, agent.MainBlock.Line)
			}
		}

		// Record output types for this variable
		if taskCall.Variable != "" {
			variableTypes[taskCall.Variable] = task.Outputs
		}
	}

	return errors
}

// validateVariableAccess validates that variable field access is correct
func (v *TaskValidator) validateVariableAccess(inputValue string, variableTypes map[string]map[string]string, errors *[]TaskValidationError, taskName, inputName string, line int) {
	// Parse variable access like "result[:data]"
	accessRegex := regexp.MustCompile(`(\w+)\[:(\w+)\]`)
	matches := accessRegex.FindStringSubmatch(inputValue)
	if len(matches) != 3 {
		return // Not a variable access pattern we recognize
	}

	variable := matches[1]
	field := matches[2]

	variableType, exists := variableTypes[variable]
	if !exists {
		*errors = append(*errors, TaskValidationError{
			Type:     "undefined_variable",
			Task:     taskName,
			Field:    inputName,
			Line:     line,
			Message:  fmt.Sprintf("Variable '%s' not defined before use in task '%s'", variable, taskName),
			Severity: "error",
		})
		return
	}

	_, fieldExists := variableType[field]
	if !fieldExists {
		*errors = append(*errors, TaskValidationError{
			Type:     "undefined_field",
			Task:     taskName,
			Field:    inputName,
			Line:     line,
			Message:  fmt.Sprintf("Field '%s' not defined in variable '%s' (available: %v)", field, variable, getKeys(variableType)),
			Severity: "error",
		})
	}
}

// validateSymbolicTaskSafety validates that symbolic task blocks are safe
func (v *TaskValidator) validateSymbolicTaskSafety(agent *AgentStructure) []TaskValidationError {
	var errors []TaskValidationError

	for taskName, task := range agent.Tasks {
		if !task.IsSymbolic {
			continue
		}

		// Check for dangerous Ruby methods in symbolic tasks
		dangerousPatterns := []string{
			"system(",
			"exec(",
			"eval(",
			"`", // backticks for command execution
			"fork(",
			"spawn(",
			"open(", // File.open can be dangerous
			"load(",
			"require(", // Should not dynamically require
		}

		for _, pattern := range dangerousPatterns {
			if strings.Contains(task.CodeBlock, pattern) {
				errors = append(errors, TaskValidationError{
					Type:     "security",
					Task:     taskName,
					Line:     task.Line,
					Message:  fmt.Sprintf("Symbolic task contains potentially dangerous method: %s", pattern),
					Severity: "error",
				})
			}
		}

		// Validate that symbolic tasks use only allowed methods
		allowedMethods := []string{
			"execute_tool(",
			"execute_task(",
			"execute_llm(",
			"logger",
		}

		// This is a basic check - in a production system you'd want more sophisticated AST parsing
		hasAllowedMethod := false
		for _, method := range allowedMethods {
			if strings.Contains(task.CodeBlock, method) {
				hasAllowedMethod = true
				break
			}
		}

		// Check if it's just simple Ruby code without method calls
		simpleRubyPatterns := []string{"=", "+", "-", "*", "/", "[", "]", "{", "}", "if", "else", "end"}
		hasSimpleRuby := false
		for _, pattern := range simpleRubyPatterns {
			if strings.Contains(task.CodeBlock, pattern) {
				hasSimpleRuby = true
				break
			}
		}

		if !hasAllowedMethod && !hasSimpleRuby {
			errors = append(errors, TaskValidationError{
				Type:     "method_usage",
				Task:     taskName,
				Line:     task.Line,
				Message:  "Symbolic task should use execute_tool(), execute_task(), execute_llm(), or simple Ruby operations",
				Severity: "warning",
			})
		}
	}

	return errors
}

// isValidDSLType checks if a type is valid according to the DSL
func isValidDSLType(t string) bool {
	validTypes := []string{"string", "integer", "number", "boolean", "array", "hash", "any"}
	for _, valid := range validTypes {
		if t == valid {
			return true
		}
	}
	return false
}

// getKeys returns the keys of a map as a slice
func getKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}