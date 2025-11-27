package main

import (
	"fmt"
	"github.com/language-operator/language-operator/pkg/validation"
)

func main() {
	// Test the specific malformed IPv6 case from issue #65
	testCases := []struct {
		name    string
		image   string
		allowed []string
	}{
		{
			name:    "malformed IPv6 from issue #65",
			image:   "[::1:5000/image",
			allowed: []string{"[::1]:5000"},
		},
		{
			name:    "another malformed case",
			image:   "[::1/image",
			allowed: []string{"[::1]"},
		},
		{
			name:    "valid IPv6 for comparison",
			image:   "[::1]:5000/image",
			allowed: []string{"[::1]:5000"},
		},
	}

	for _, tc := range testCases {
		fmt.Printf("Testing: %s\n", tc.name)
		fmt.Printf("  Image: %s\n", tc.image)
		fmt.Printf("  Allowed: %v\n", tc.allowed)
		
		err := validation.ValidateImageRegistry(tc.image, tc.allowed)
		if err != nil {
			fmt.Printf("  Result: REJECTED (✓) - %s\n", err.Error())
		} else {
			fmt.Printf("  Result: ALLOWED (✗) - validation passed\n")
		}
		fmt.Println()
	}
}