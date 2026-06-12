package controllers

import (
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
)

// MCP stdio→Streamable-HTTP bridge wiring.
//
// For transport=stdio tools the operator runs the user's stdio MCP command under a pinned,
// persistent bridge (supergateway --stateful) that exposes Streamable HTTP at /mcp and a
// /health endpoint. The bridge keeps ONE long-lived child, so there's no per-request spawn.
// Because tool pods run with readOnlyRootFilesystem and as a non-root user, the operator also
// mounts writable scratch dirs (a HOME for npm/uv caches and /tmp) so npx/uvx can run.

const (
	// mcpBridgeHome is the writable HOME the bridge container runs with.
	mcpBridgeHome = "/home/mcp"
	// mcpBridgeTmpPath is where the tmp scratch volume is mounted.
	mcpBridgeTmpPath = "/tmp"
	// mcpBridgeVolumePrefix names the bridge scratch volumes in a service-mode tool pod (one
	// tool per pod). Sidecar tools use a per-tool prefix since they share the agent pod.
	mcpBridgeVolumePrefix = "mcp-bridge"
)

func bridgeCacheVolumeName(prefix string) string { return prefix + "-cache" }
func bridgeTmpVolumeName(prefix string) string   { return prefix + "-tmp" }

// resolveMCPBridgeImage returns the configured bridge image, falling back to the pinned default.
func resolveMCPBridgeImage(configured string) string {
	if configured != "" {
		return configured
	}
	return langopv1alpha1.DefaultMCPBridgeImage
}

// bridgeContainerCommand is the bridge entrypoint (overrides the image's default).
func bridgeContainerCommand() []string {
	return []string{"supergateway"}
}

// buildBridgeArgs assembles the supergateway args that wrap the tool's stdio command and serve
// it over stateful Streamable HTTP at /mcp (+ /health) on the given port. The arg order is
// deterministic so CreateOrUpdate does not see perpetual diffs.
func buildBridgeArgs(stdioCommand []string, port int32) []string {
	return []string{
		"--stdio", strings.Join(stdioCommand, " "),
		"--outputTransport", "streamableHttp",
		"--stateful",
		"--streamableHttpPath", "/mcp",
		"--healthEndpoint", "/health",
		"--port", strconv.Itoa(int(port)),
		"--logLevel", "info",
	}
}

// bridgeCacheEnv points npm/uv/XDG caches and HOME at the writable scratch volume, so npx/uvx
// work despite readOnlyRootFilesystem. Order is fixed for diff stability.
func bridgeCacheEnv() []corev1.EnvVar {
	return []corev1.EnvVar{
		{Name: "HOME", Value: mcpBridgeHome},
		{Name: "npm_config_cache", Value: mcpBridgeHome + "/.npm"},
		{Name: "XDG_CACHE_HOME", Value: mcpBridgeHome + "/.cache"},
		{Name: "XDG_DATA_HOME", Value: mcpBridgeHome + "/.local/share"},
		{Name: "UV_CACHE_DIR", Value: mcpBridgeHome + "/.uv"},
	}
}

// bridgeVolumes returns the writable scratch volumes a stdio bridge container needs. The prefix
// keeps volume names unique when several stdio sidecars share one agent pod.
func bridgeVolumes(prefix string) []corev1.Volume {
	return []corev1.Volume{
		{Name: bridgeCacheVolumeName(prefix), VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		{Name: bridgeTmpVolumeName(prefix), VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
	}
}

// bridgeVolumeMounts mounts the scratch volumes into the bridge container.
func bridgeVolumeMounts(prefix string) []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{Name: bridgeCacheVolumeName(prefix), MountPath: mcpBridgeHome},
		{Name: bridgeTmpVolumeName(prefix), MountPath: mcpBridgeTmpPath},
	}
}

// defaultBridgeReadinessProbe probes the bridge's /health endpoint. The failure threshold is
// generous because a stdio server's first boot may cold-fetch its package via npx/uvx.
func defaultBridgeReadinessProbe(port int32) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/health",
				Port: intstr.FromInt(int(port)),
			},
		},
		InitialDelaySeconds: 5,
		PeriodSeconds:       10,
		TimeoutSeconds:      3,
		FailureThreshold:    12, // ~120s grace for cold npx/uvx fetch
	}
}

// isStdioTool reports whether the tool uses the stdio transport.
func isStdioTool(tool *langopv1alpha1.LanguageTool) bool {
	return tool.Spec.Transport == "stdio"
}

// stdioCommandOf returns the tool's stdio command, or nil if unset.
func stdioCommandOf(tool *langopv1alpha1.LanguageTool) []string {
	if tool.Spec.Stdio != nil {
		return tool.Spec.Stdio.Command
	}
	return nil
}
