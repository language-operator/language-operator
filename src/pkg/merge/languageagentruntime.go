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

package merge

import (
	"maps"

	corev1 "k8s.io/api/core/v1"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
)

// ApplyRuntimeDefaults merges runtime defaults into an agent spec.
// The agent spec should be a deep copy — the stored agent object must never be
// mutated with runtime-derived values.
//
// Merge rules:
//   - Scalars: runtime value used only when the agent field is zero/nil (agent wins).
//   - Maps: merged key-by-key; agent keys win on collision.
//   - Lists: runtime items prepended, agent items appended (runtime-first ordering).
func ApplyRuntimeDefaults(agent *langopv1alpha1.LanguageAgentSpec, rt *langopv1alpha1.LanguageAgentRuntimeSpec) {
	// --- Top-level scalar fields ---

	if agent.Image == "" {
		agent.Image = rt.Image
	}
	// Ports: replace semantics — runtime ports apply only when agent has none.
	// Ports define the agent's network identity and must not be additively merged.
	if len(agent.Ports) == 0 && len(rt.Ports) > 0 {
		agent.Ports = make([]langopv1alpha1.AgentPort, len(rt.Ports))
		copy(agent.Ports, rt.Ports)
	}
	// --- ExecutionSpec (agent wins if non-zero/non-nil) ---
	// Agents run as Argo Workflows, so the runtime's replicas/autoscaling are not
	// merged — they have no meaning for a Workflow and the webhook rejects them.

	e := &agent.Execution
	re := &rt.Execution

	if e.Mode == "" {
		e.Mode = re.Mode
	}
	if e.Schedule == "" {
		e.Schedule = re.Schedule
	}
	if e.Timezone == "" {
		e.Timezone = re.Timezone
	}
	if e.ActiveDeadlineSeconds == nil && re.ActiveDeadlineSeconds != nil {
		v := *re.ActiveDeadlineSeconds
		e.ActiveDeadlineSeconds = &v
	}
	if e.TTLSecondsAfterFinished == nil && re.TTLSecondsAfterFinished != nil {
		v := *re.TTLSecondsAfterFinished
		e.TTLSecondsAfterFinished = &v
	}
	if e.ConcurrencyPolicy == "" {
		e.ConcurrencyPolicy = re.ConcurrencyPolicy
	}
	if e.Suspend == nil && re.Suspend != nil {
		v := *re.Suspend
		e.Suspend = &v
	}
	if e.RetryLimit == nil && re.RetryLimit != nil {
		v := *re.RetryLimit
		e.RetryLimit = &v
	}

	// --- DeploymentSpec scalar fields (agent wins if non-zero/non-nil) ---

	d := &agent.Deployment
	r := &rt.Deployment

	if d.ImagePullPolicy == "" {
		d.ImagePullPolicy = r.ImagePullPolicy
	}
	if len(d.ImagePullSecrets) == 0 && len(r.ImagePullSecrets) > 0 {
		d.ImagePullSecrets = make([]corev1.LocalObjectReference, len(r.ImagePullSecrets))
		copy(d.ImagePullSecrets, r.ImagePullSecrets)
	}
	if d.Resources.Requests == nil && d.Resources.Limits == nil {
		d.Resources = r.Resources
	}
	if len(d.NodeSelector) == 0 && len(r.NodeSelector) > 0 {
		d.NodeSelector = make(map[string]string, len(r.NodeSelector))
		maps.Copy(d.NodeSelector, r.NodeSelector)
	}
	if d.Affinity == nil && r.Affinity != nil {
		a := *r.Affinity
		d.Affinity = &a
	}
	if len(d.Tolerations) == 0 && len(r.Tolerations) > 0 {
		d.Tolerations = make([]corev1.Toleration, len(r.Tolerations))
		copy(d.Tolerations, r.Tolerations)
	}
	if len(d.TopologySpreadConstraints) == 0 && len(r.TopologySpreadConstraints) > 0 {
		d.TopologySpreadConstraints = make([]corev1.TopologySpreadConstraint, len(r.TopologySpreadConstraints))
		copy(d.TopologySpreadConstraints, r.TopologySpreadConstraints)
	}
	if d.ServiceAccountName == "" {
		d.ServiceAccountName = r.ServiceAccountName
	}
	// ServiceAccountAnnotations: merge key-by-key; agent keys win on collision.
	if len(r.ServiceAccountAnnotations) > 0 {
		if d.ServiceAccountAnnotations == nil {
			d.ServiceAccountAnnotations = make(map[string]string, len(r.ServiceAccountAnnotations))
		}
		for k, v := range r.ServiceAccountAnnotations {
			if _, exists := d.ServiceAccountAnnotations[k]; !exists {
				d.ServiceAccountAnnotations[k] = v
			}
		}
	}
	if d.SecurityContext == nil && r.SecurityContext != nil {
		sc := *r.SecurityContext
		d.SecurityContext = &sc
	}
	if len(d.Command) == 0 && len(r.Command) > 0 {
		d.Command = append([]string{}, r.Command...)
	}
	if len(d.Args) == 0 && len(r.Args) > 0 {
		d.Args = append([]string{}, r.Args...)
	}
	if d.LivenessProbe == nil && r.LivenessProbe != nil {
		p := *r.LivenessProbe
		d.LivenessProbe = &p
	}
	if d.ReadinessProbe == nil && r.ReadinessProbe != nil {
		p := *r.ReadinessProbe
		d.ReadinessProbe = &p
	}
	if d.StartupProbe == nil && r.StartupProbe != nil {
		p := *r.StartupProbe
		d.StartupProbe = &p
	}
	if d.ServiceType == "" {
		d.ServiceType = r.ServiceType
	}

	// --- DeploymentSpec map fields (merge; agent keys win on collision) ---

	if len(r.ServiceAnnotations) > 0 {
		if d.ServiceAnnotations == nil {
			d.ServiceAnnotations = make(map[string]string)
		}
		for k, v := range r.ServiceAnnotations {
			if _, exists := d.ServiceAnnotations[k]; !exists {
				d.ServiceAnnotations[k] = v
			}
		}
	}
	if len(r.PodAnnotations) > 0 {
		if d.PodAnnotations == nil {
			d.PodAnnotations = make(map[string]string)
		}
		for k, v := range r.PodAnnotations {
			if _, exists := d.PodAnnotations[k]; !exists {
				d.PodAnnotations[k] = v
			}
		}
	}
	if len(r.PodLabels) > 0 {
		if d.PodLabels == nil {
			d.PodLabels = make(map[string]string)
		}
		for k, v := range r.PodLabels {
			if _, exists := d.PodLabels[k]; !exists {
				d.PodLabels[k] = v
			}
		}
	}

	// --- Credentials: runtime-first, agent-appended, deduped by Name (agent wins) ---

	if len(rt.Credentials) > 0 {
		seen := make(map[string]struct{}, len(agent.Credentials))
		for _, c := range agent.Credentials {
			seen[c.Name] = struct{}{}
		}
		merged := make([]langopv1alpha1.CredentialSpec, 0, len(rt.Credentials)+len(agent.Credentials))
		for _, c := range rt.Credentials {
			if _, exists := seen[c.Name]; !exists {
				merged = append(merged, *c.DeepCopy())
			}
		}
		agent.Credentials = append(merged, agent.Credentials...)
	}

	// --- DeploymentSpec list fields (runtime-first, agent-appended) ---

	if len(r.InitContainers) > 0 {
		d.InitContainers = append(r.InitContainers, d.InitContainers...)
	}
	if len(r.Env) > 0 {
		d.Env = append(r.Env, d.Env...)
	}
	if len(r.EnvFrom) > 0 {
		d.EnvFrom = append(r.EnvFrom, d.EnvFrom...)
	}
	if len(r.Volumes) > 0 {
		d.Volumes = append(r.Volumes, d.Volumes...)
	}
	if len(r.VolumeMounts) > 0 {
		d.VolumeMounts = append(r.VolumeMounts, d.VolumeMounts...)
	}
	// RoleRules: runtime-first, agent-appended (same pattern as InitContainers, Env, Volumes).
	if len(r.RoleRules) > 0 {
		d.RoleRules = append(r.RoleRules, d.RoleRules...)
	}
}
