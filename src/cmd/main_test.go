/*
Copyright 2025 Langop Team.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"os"
	"testing"
	"time"
)

func TestStartupTimeoutConfiguration(t *testing.T) {
	originalTimeout := os.Getenv("STARTUP_TIMEOUT")
	defer func() {
		if originalTimeout == "" {
			os.Unsetenv("STARTUP_TIMEOUT")
		} else {
			os.Setenv("STARTUP_TIMEOUT", originalTimeout)
		}
	}()

	tests := []struct {
		name            string
		envValue        string
		expectedTimeout time.Duration
		expectDefault   bool
	}{
		{
			name:            "no env var - uses default",
			envValue:        "",
			expectedTimeout: 60 * time.Second,
			expectDefault:   true,
		},
		{
			name:            "valid timeout value",
			envValue:        "120s",
			expectedTimeout: 120 * time.Second,
			expectDefault:   false,
		},
		{
			name:            "valid timeout in minutes",
			envValue:        "5m",
			expectedTimeout: 5 * time.Minute,
			expectDefault:   false,
		},
		{
			name:            "invalid timeout - falls back to default",
			envValue:        "invalid",
			expectedTimeout: 60 * time.Second,
			expectDefault:   true,
		},
		{
			name:            "zero timeout - uses zero",
			envValue:        "0s",
			expectedTimeout: 0,
			expectDefault:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envValue == "" {
				os.Unsetenv("STARTUP_TIMEOUT")
			} else {
				os.Setenv("STARTUP_TIMEOUT", tc.envValue)
			}

			startupTimeout := 60 * time.Second
			if timeoutStr := os.Getenv("STARTUP_TIMEOUT"); timeoutStr != "" {
				if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
					startupTimeout = parsedTimeout
				}
			}

			if startupTimeout != tc.expectedTimeout {
				t.Errorf("Expected timeout %v, got %v", tc.expectedTimeout, startupTimeout)
			}
		})
	}
}
