package validation

import (
	"fmt"
	"strings"
)

// ValidateImageRegistry checks if the image registry is in the allowed list.
// It handles docker.io defaults, short names, and cloud provider patterns.
//
// Examples:
//   - "nginx" -> extracts "docker.io" (default registry)
//   - "docker.io/nginx" -> extracts "docker.io"
//   - "gcr.io/project/image" -> extracts "gcr.io"
//   - "123456789.dkr.ecr.us-west-2.amazonaws.com/image" -> extracts "123456789.dkr.ecr.us-west-2.amazonaws.com"
func ValidateImageRegistry(image string, allowedRegistries []string) error {
	if image == "" {
		return fmt.Errorf("image reference cannot be empty")
	}

	registry := extractRegistry(image)

	// Check if registry matches any allowed registry (exact or wildcard)
	for _, allowed := range allowedRegistries {
		if matchesRegistry(registry, allowed) {
			return nil
		}
	}

	return fmt.Errorf("registry %s not in whitelist: %v", registry, allowedRegistries)
}

// extractRegistry extracts the registry from an image reference.
// Handles various formats:
//   - "nginx" or "library/nginx" -> "docker.io" (Docker Hub default)
//   - "docker.io/nginx" -> "docker.io"
//   - "gcr.io/project/image:tag" -> "gcr.io"
//   - "localhost:5000/image" -> "localhost:5000"
//   - "[::1]:5000/image" -> "[::1]:5000" (IPv6)
//   - "[2001:db8::1]:8080/app" -> "[2001:db8::1]:8080" (IPv6 with port)
func extractRegistry(image string) string {
	// Remove tag or digest if present
	// Tags: image:tag
	// Digests: image@sha256:...
	if idx := strings.Index(image, "@"); idx != -1 {
		image = image[:idx]
	}

	// Handle IPv6 addresses: [IPv6]:port format
	if strings.HasPrefix(image, "[") {
		if closeBracket := strings.Index(image, "]"); closeBracket != -1 {
			// Find the first slash after the closing bracket to separate registry from path
			remainder := image[closeBracket+1:]
			if slashIdx := strings.Index(remainder, "/"); slashIdx != -1 {
				// Extract registry portion: [IPv6]:port up to first slash
				return image[:closeBracket+1+slashIdx]
			}
			// If no slash found, entire string is the registry
			return image
		}
		// If no closing bracket found, treat as malformed but continue with normal parsing
	}

	// Original IPv4/hostname logic for non-bracketed addresses
	if idx := strings.Index(image, ":"); idx != -1 {
		// Check if colon is part of port number (e.g., localhost:5000)
		// or a tag separator. Port comes before first slash.
		slashIdx := strings.Index(image, "/")
		if slashIdx == -1 || idx < slashIdx {
			// Colon is part of registry (port number), keep it
		} else {
			// Colon is tag separator, remove tag
			image = image[:idx]
		}
	}

	// Split into parts
	parts := strings.Split(image, "/")

	// If there's only one part (e.g., "nginx"), it's a Docker Hub short name
	if len(parts) == 1 {
		return "docker.io"
	}

	// If there are two parts, check if first part looks like a registry
	// Registries contain dots, colons (ports), or are "localhost"
	if len(parts) == 2 {
		firstPart := parts[0]
		if strings.Contains(firstPart, ".") || strings.Contains(firstPart, ":") || firstPart == "localhost" {
			return firstPart
		}
		// Otherwise it's Docker Hub implicit (e.g., "library/nginx")
		return "docker.io"
	}

	// Three or more parts: first part is the registry
	return parts[0]
}

// matchesRegistry checks if a registry matches an allowed pattern.
// Supports wildcards with "*." prefix for subdomains.
//
// Examples:
//   - registry="docker.io", allowed="docker.io" -> true
//   - registry="us.gcr.io", allowed="*.gcr.io" -> true
//   - registry="gcr.io", allowed="*.gcr.io" -> true (wildcard matches base domain)
//   - registry="123456.dkr.ecr.us-west-2.amazonaws.com", allowed="*.amazonaws.com" -> true
func matchesRegistry(registry, allowed string) bool {
	// Exact match
	if registry == allowed {
		return true
	}

	// Wildcard match: *.example.com matches anything.example.com or example.com
	if strings.HasPrefix(allowed, "*.") {
		baseDomain := allowed[2:] // Remove "*."

		// Match base domain exactly
		if registry == baseDomain {
			return true
		}

		// Match subdomains
		if strings.HasSuffix(registry, "."+baseDomain) {
			return true
		}
	}

	return false
}
