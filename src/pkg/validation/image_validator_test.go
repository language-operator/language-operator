package validation

import (
	"strings"
	"testing"
	"time"
)

func TestValidateImageRegistry(t *testing.T) {
	allowedRegistries := []string{
		"docker.io",
		"gcr.io",
		"*.gcr.io",
		"quay.io",
		"ghcr.io",
		"registry.k8s.io",
		"codeberg.org",
		"gitlab.com",
		"*.amazonaws.com",
		"*.azurecr.io",
	}

	tests := []struct {
		name      string
		image     string
		allowed   []string
		wantError bool
	}{
		// Docker Hub cases
		{
			name:      "short name resolves to docker.io",
			image:     "nginx",
			allowed:   allowedRegistries,
			wantError: false,
		},
		{
			name:      "short name with tag",
			image:     "nginx:latest",
			allowed:   allowedRegistries,
			wantError: false,
		},
		{
			name:      "library image",
			image:     "library/nginx",
			allowed:   allowedRegistries,
			wantError: false,
		},
		{
			name:      "fully qualified docker.io",
			image:     "docker.io/nginx",
			allowed:   allowedRegistries,
			wantError: false,
		},
		{
			name:      "docker.io with org and tag",
			image:     "docker.io/library/nginx:1.21",
			allowed:   allowedRegistries,
			wantError: false,
		},

		// GCR cases
		{
			name:      "gcr.io base domain",
			image:     "gcr.io/project/image",
			allowed:   allowedRegistries,
			wantError: false,
		},
		{
			name:      "us.gcr.io subdomain",
			image:     "us.gcr.io/project/image:v1.0",
			allowed:   allowedRegistries,
			wantError: false,
		},
		{
			name:      "eu.gcr.io subdomain",
			image:     "eu.gcr.io/project/image",
			allowed:   allowedRegistries,
			wantError: false,
		},

		// Other registries
		{
			name:      "quay.io",
			image:     "quay.io/organization/image:latest",
			allowed:   allowedRegistries,
			wantError: false,
		},
		{
			name:      "ghcr.io",
			image:     "ghcr.io/user/repo:tag",
			allowed:   allowedRegistries,
			wantError: false,
		},
		{
			name:      "registry.k8s.io",
			image:     "registry.k8s.io/kube-apiserver:v1.28.0",
			allowed:   allowedRegistries,
			wantError: false,
		},
		{
			name:      "codeberg.org",
			image:     "codeberg.org/user/image",
			allowed:   allowedRegistries,
			wantError: false,
		},
		{
			name:      "gitlab.com",
			image:     "gitlab.com/group/project/image:tag",
			allowed:   allowedRegistries,
			wantError: false,
		},

		// AWS ECR cases
		{
			name:      "ecr us-west-2",
			image:     "123456789012.dkr.ecr.us-west-2.amazonaws.com/myapp:latest",
			allowed:   allowedRegistries,
			wantError: false,
		},
		{
			name:      "ecr eu-central-1",
			image:     "987654321098.dkr.ecr.eu-central-1.amazonaws.com/service",
			allowed:   allowedRegistries,
			wantError: false,
		},

		// Azure ACR cases
		{
			name:      "azurecr.io",
			image:     "myregistry.azurecr.io/myapp:v2",
			allowed:   allowedRegistries,
			wantError: false,
		},

		// Digest format
		{
			name:      "image with digest",
			image:     "docker.io/nginx@sha256:abcdef1234567890",
			allowed:   allowedRegistries,
			wantError: false,
		},
		{
			name:      "gcr with digest",
			image:     "gcr.io/project/image@sha256:1234567890abcdef",
			allowed:   allowedRegistries,
			wantError: false,
		},

		// Port numbers
		{
			name:      "localhost with port",
			image:     "localhost:5000/myimage",
			allowed:   []string{"localhost:5000"},
			wantError: false,
		},
		{
			name:      "custom registry with port",
			image:     "registry.example.com:443/image:tag",
			allowed:   []string{"registry.example.com:443"},
			wantError: false,
		},

		// IPv6 addresses
		{
			name:      "ipv6 localhost",
			image:     "[::1]:5000/myimage",
			allowed:   []string{"[::1]:5000"},
			wantError: false,
		},
		{
			name:      "ipv6 full address",
			image:     "[2001:db8::1]:8080/app:latest",
			allowed:   []string{"[2001:db8::1]:8080"},
			wantError: false,
		},
		{
			name:      "ipv6 without port",
			image:     "[::1]/image",
			allowed:   []string{"[::1]"},
			wantError: false,
		},
		{
			name:      "ipv6 rejected",
			image:     "[::1]:5000/image",
			allowed:   allowedRegistries,
			wantError: true,
		},

		// Rejection cases
		{
			name:      "untrusted registry",
			image:     "untrusted.io/malicious/image",
			allowed:   allowedRegistries,
			wantError: true,
		},
		{
			name:      "empty image",
			image:     "",
			allowed:   allowedRegistries,
			wantError: true,
		},
		{
			name:      "random registry",
			image:     "random-registry.com/image:tag",
			allowed:   allowedRegistries,
			wantError: true,
		},
		{
			name:      "localhost not in whitelist",
			image:     "localhost:5000/image",
			allowed:   allowedRegistries,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateImageRegistry(tt.image, tt.allowed)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateImageRegistry(%q) error = %v, wantError %v", tt.image, err, tt.wantError)
			}
		})
	}
}

func TestExtractRegistry(t *testing.T) {
	tests := []struct {
		name     string
		image    string
		expected string
	}{
		// Docker Hub
		{"short name", "nginx", "docker.io"},
		{"short name with tag", "nginx:latest", "docker.io"},
		{"library image", "library/nginx", "docker.io"},
		{"docker.io explicit", "docker.io/nginx", "docker.io"},
		{"docker.io with org", "docker.io/library/nginx", "docker.io"},
		{"docker.io with tag", "docker.io/nginx:1.21", "docker.io"},

		// Other registries
		{"gcr.io", "gcr.io/project/image", "gcr.io"},
		{"us.gcr.io", "us.gcr.io/project/image", "us.gcr.io"},
		{"quay.io", "quay.io/org/image", "quay.io"},
		{"ghcr.io", "ghcr.io/user/repo", "ghcr.io"},
		{"registry.k8s.io", "registry.k8s.io/pause:3.9", "registry.k8s.io"},
		{"codeberg.org", "codeberg.org/user/image", "codeberg.org"},
		{"gitlab.com", "gitlab.com/group/project", "gitlab.com"},

		// Cloud providers
		{"ecr", "123456789012.dkr.ecr.us-west-2.amazonaws.com/app", "123456789012.dkr.ecr.us-west-2.amazonaws.com"},
		{"acr", "myregistry.azurecr.io/app", "myregistry.azurecr.io"},

		// Special formats
		{"digest", "nginx@sha256:abc123", "docker.io"},
		{"digest with registry", "gcr.io/project/image@sha256:def456", "gcr.io"},
		{"tag and path", "quay.io/org/image:v1.0", "quay.io"},

		// Port numbers
		{"localhost with port", "localhost:5000/image", "localhost:5000"},
		{"registry with port", "registry.example.com:443/image", "registry.example.com:443"},
		{"registry with port and tag", "registry.example.com:443/image:tag", "registry.example.com:443"},

		// IPv6 addresses
		{"ipv6 localhost", "[::1]/image", "[::1]"},
		{"ipv6 localhost with port", "[::1]:5000/image", "[::1]:5000"},
		{"ipv6 with port and tag", "[2001:db8::1]:8080/app:latest", "[2001:db8::1]:8080"},
		{"ipv6 full address", "[2001:0db8:85a3::8a2e:0370:7334]:9090/service", "[2001:0db8:85a3::8a2e:0370:7334]:9090"},
		{"ipv6 without port", "[::1]/org/image", "[::1]"},
		{"ipv6 nested path", "[::1]:5000/org/project/image", "[::1]:5000"},
		{"ipv6 with digest", "[::1]:5000/image@sha256:abc", "[::1]:5000"},
		{"ipv6 compressed", "[::ffff:192.0.2.1]:8080/app", "[::ffff:192.0.2.1]:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRegistry(tt.image)
			if got != tt.expected {
				t.Errorf("extractRegistry(%q) = %q, want %q", tt.image, got, tt.expected)
			}
		})
	}
}

func TestMatchesRegistry(t *testing.T) {
	tests := []struct {
		name     string
		registry string
		allowed  string
		expected bool
	}{
		// Exact matches
		{"exact match", "docker.io", "docker.io", true},
		{"exact mismatch", "docker.io", "gcr.io", false},

		// Wildcard matches
		{"wildcard base domain", "gcr.io", "*.gcr.io", true},
		{"wildcard subdomain", "us.gcr.io", "*.gcr.io", true},
		{"wildcard multi-level subdomain", "us-west1.gcr.io", "*.gcr.io", true},
		{"wildcard ECR", "123456789012.dkr.ecr.us-west-2.amazonaws.com", "*.amazonaws.com", true},
		{"wildcard ACR", "myregistry.azurecr.io", "*.azurecr.io", true},

		// Wildcard non-matches
		{"wildcard different domain", "gcr.io", "*.amazonaws.com", false},
		{"wildcard partial match", "gcr.io.fake.com", "*.gcr.io", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesRegistry(tt.registry, tt.allowed)
			if got != tt.expected {
				t.Errorf("matchesRegistry(%q, %q) = %v, want %v", tt.registry, tt.allowed, got, tt.expected)
			}
		})
	}
}

// TestValidationPerformance ensures validation is fast (<1ms per validation)
func TestValidationPerformance(t *testing.T) {
	allowedRegistries := []string{
		"docker.io",
		"gcr.io",
		"*.gcr.io",
		"quay.io",
		"ghcr.io",
		"registry.k8s.io",
		"codeberg.org",
		"gitlab.com",
		"*.amazonaws.com",
		"*.azurecr.io",
	}

	testImages := []string{
		"nginx",
		"docker.io/library/nginx:latest",
		"gcr.io/project/image",
		"us.gcr.io/project/image:v1.0",
		"123456789012.dkr.ecr.us-west-2.amazonaws.com/app:latest",
		"myregistry.azurecr.io/app",
		"quay.io/org/image:tag",
		"ghcr.io/user/repo@sha256:abc123",
	}

	iterations := 1000
	start := time.Now()

	for i := 0; i < iterations; i++ {
		for _, image := range testImages {
			_ = ValidateImageRegistry(image, allowedRegistries)
		}
	}

	elapsed := time.Since(start)
	avgPerValidation := elapsed / time.Duration(iterations*len(testImages))

	// Performance target: <1ms per validation
	if avgPerValidation > time.Millisecond {
		t.Errorf("Performance target not met: average %v per validation (target: <1ms)", avgPerValidation)
	}

	t.Logf("Performance: %v per validation (%d validations in %v)", avgPerValidation, iterations*len(testImages), elapsed)
}

// TestErrorMessages ensures error messages are clear and helpful
func TestErrorMessages(t *testing.T) {
	allowedRegistries := []string{"docker.io", "gcr.io"}

	err := ValidateImageRegistry("untrusted.io/image", allowedRegistries)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "untrusted.io") {
		t.Errorf("error message should contain rejected registry 'untrusted.io', got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "docker.io") {
		t.Errorf("error message should contain allowed registries, got: %s", errMsg)
	}

	// Test empty image error
	err = ValidateImageRegistry("", allowedRegistries)
	if err == nil {
		t.Fatal("expected error for empty image, got nil")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error message should mention empty image, got: %s", err.Error())
	}
}
