package synthesis

import (
	"testing"
)

func TestDetectTemporalIntent(t *testing.T) {
	tests := []struct {
		name         string
		instructions string
		expected     TemporalIntent
	}{
		// One-shot cases
		{
			name:         "explicit run once",
			instructions: "Run once to analyze the codebase",
			expected:     OneShot,
		},
		{
			name:         "one time instruction",
			instructions: "One time migration of data",
			expected:     OneShot,
		},
		{
			name:         "single time execution",
			instructions: "Execute this single time to setup the environment",
			expected:     OneShot,
		},
		{
			name:         "just once",
			instructions: "Just once, check if the service is running",
			expected:     OneShot,
		},

		// Scheduled cases
		{
			name:         "every hour",
			instructions: "Post a fact every hour",
			expected:     Scheduled,
		},
		{
			name:         "daily execution",
			instructions: "Run daily backups at midnight",
			expected:     Scheduled,
		},
		{
			name:         "hourly task",
			instructions: "Check status hourly",
			expected:     Scheduled,
		},
		{
			name:         "weekly report",
			instructions: "Generate weekly reports",
			expected:     Scheduled,
		},
		{
			name:         "monthly summary",
			instructions: "Send monthly summary emails",
			expected:     Scheduled,
		},
		{
			name:         "cron schedule",
			instructions: "Run on a cron schedule",
			expected:     Scheduled,
		},
		{
			name:         "at midnight",
			instructions: "Execute at midnight",
			expected:     Scheduled,
		},
		{
			name:         "periodically",
			instructions: "Check periodically for updates",
			expected:     Scheduled,
		},
		{
			name:         "each day",
			instructions: "Review logs each day",
			expected:     Scheduled,
		},

		// Continuous cases
		{
			name:         "provides service",
			instructions: "Provides fun facts about Ruby programming",
			expected:     Continuous,
		},
		{
			name:         "monitors system",
			instructions: "Monitor system health and alert on issues",
			expected:     Continuous,
		},
		{
			name:         "responds to requests",
			instructions: "Answer questions about documentation",
			expected:     Continuous,
		},
		{
			name:         "watches for changes",
			instructions: "Watch for file changes and process them",
			expected:     Continuous,
		},
		{
			name:         "empty instructions",
			instructions: "",
			expected:     Continuous,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectTemporalIntent(tt.instructions)
			if result != tt.expected {
				t.Errorf("detectTemporalIntent(%q) = %v, want %v",
					tt.instructions, result, tt.expected)
			}
		})
	}
}

func TestTemporalIntentString(t *testing.T) {
	tests := []struct {
		intent   TemporalIntent
		expected string
	}{
		{OneShot, "One-shot"},
		{Scheduled, "Scheduled"},
		{Continuous, "Continuous"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.intent.String()
			if result != tt.expected {
				t.Errorf("TemporalIntent(%d).String() = %q, want %q",
					tt.intent, result, tt.expected)
			}
		})
	}
}

// TestValidateSecurity has been removed - validation is now in pkg/validation/ruby_validator_test.go
// The AST-based validator is tested there with comprehensive bypass tests
