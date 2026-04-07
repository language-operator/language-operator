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

package v1alpha1

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/utils/ptr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyRuntimeDefaults_Ports_AgentEmpty(t *testing.T) {
	agent := &LanguageAgentSpec{}
	rt := &LanguageAgentRuntimeSpec{
		Ports: []AgentPort{
			{Name: "http", Port: 18789, Protocol: corev1.ProtocolTCP, Expose: ptr.To(true)},
		},
	}
	ApplyRuntimeDefaults(agent, rt)
	require.Len(t, agent.Ports, 1)
	assert.Equal(t, int32(18789), agent.Ports[0].Port)
	assert.Equal(t, "http", agent.Ports[0].Name)
	assert.Equal(t, ptr.To(true), agent.Ports[0].Expose)
}

func TestApplyRuntimeDefaults_Ports_AgentWins(t *testing.T) {
	agent := &LanguageAgentSpec{
		Ports: []AgentPort{
			{Name: "api", Port: 9000, Protocol: corev1.ProtocolTCP, Expose: ptr.To(true)},
		},
	}
	rt := &LanguageAgentRuntimeSpec{
		Ports: []AgentPort{
			{Name: "http", Port: 18789, Protocol: corev1.ProtocolTCP, Expose: ptr.To(true)},
		},
	}
	ApplyRuntimeDefaults(agent, rt)
	// Agent's ports must be unchanged — replace semantics, not merge.
	require.Len(t, agent.Ports, 1)
	assert.Equal(t, int32(9000), agent.Ports[0].Port)
	assert.Equal(t, "api", agent.Ports[0].Name)
}

func TestApplyRuntimeDefaults_Ports_RuntimeEmpty(t *testing.T) {
	agent := &LanguageAgentSpec{}
	rt := &LanguageAgentRuntimeSpec{}
	ApplyRuntimeDefaults(agent, rt)
	assert.Empty(t, agent.Ports, "empty runtime should not inject any ports")
}

func TestApplyRuntimeDefaults_ServiceAccountAnnotations_AgentEmpty(t *testing.T) {
	agent := &LanguageAgentSpec{}
	rt := &LanguageAgentRuntimeSpec{
		Deployment: DeploymentSpec{
			ServiceAccountAnnotations: map[string]string{
				"eks.amazonaws.com/role-arn": "arn:aws:iam::123:role/my-role",
			},
		},
	}
	ApplyRuntimeDefaults(agent, rt)
	require.NotNil(t, agent.Deployment.ServiceAccountAnnotations)
	assert.Equal(t, "arn:aws:iam::123:role/my-role", agent.Deployment.ServiceAccountAnnotations["eks.amazonaws.com/role-arn"])
}

func TestApplyRuntimeDefaults_ServiceAccountAnnotations_AgentWins(t *testing.T) {
	agent := &LanguageAgentSpec{
		Deployment: DeploymentSpec{
			ServiceAccountAnnotations: map[string]string{
				"eks.amazonaws.com/role-arn": "arn:aws:iam::123:role/agent-role",
				"agent-only-key":             "agent-value",
			},
		},
	}
	rt := &LanguageAgentRuntimeSpec{
		Deployment: DeploymentSpec{
			ServiceAccountAnnotations: map[string]string{
				"eks.amazonaws.com/role-arn": "arn:aws:iam::123:role/runtime-role",
				"runtime-only-key":           "runtime-value",
			},
		},
	}
	ApplyRuntimeDefaults(agent, rt)
	// Agent key wins on collision.
	assert.Equal(t, "arn:aws:iam::123:role/agent-role", agent.Deployment.ServiceAccountAnnotations["eks.amazonaws.com/role-arn"])
	// Agent-only key preserved.
	assert.Equal(t, "agent-value", agent.Deployment.ServiceAccountAnnotations["agent-only-key"])
	// Runtime-only key merged in.
	assert.Equal(t, "runtime-value", agent.Deployment.ServiceAccountAnnotations["runtime-only-key"])
}

func TestApplyRuntimeDefaults_ServiceAccountAnnotations_RuntimeEmpty(t *testing.T) {
	agent := &LanguageAgentSpec{}
	rt := &LanguageAgentRuntimeSpec{}
	ApplyRuntimeDefaults(agent, rt)
	assert.Nil(t, agent.Deployment.ServiceAccountAnnotations, "empty runtime should not set ServiceAccountAnnotations")
}

func TestApplyRuntimeDefaults_RoleRules_RuntimeOnly(t *testing.T) {
	agent := &LanguageAgentSpec{}
	rt := &LanguageAgentRuntimeSpec{
		Deployment: DeploymentSpec{
			RoleRules: []rbacv1.PolicyRule{
				{APIGroups: []string{""}, Resources: []string{"secrets"}, Verbs: []string{"get"}},
			},
		},
	}
	ApplyRuntimeDefaults(agent, rt)
	require.Len(t, agent.Deployment.RoleRules, 1)
	assert.Equal(t, []string{"secrets"}, agent.Deployment.RoleRules[0].Resources)
}

func TestApplyRuntimeDefaults_RoleRules_RuntimePrepended(t *testing.T) {
	agent := &LanguageAgentSpec{
		Deployment: DeploymentSpec{
			RoleRules: []rbacv1.PolicyRule{
				{APIGroups: []string{""}, Resources: []string{"configmaps"}, Verbs: []string{"create"}},
			},
		},
	}
	rt := &LanguageAgentRuntimeSpec{
		Deployment: DeploymentSpec{
			RoleRules: []rbacv1.PolicyRule{
				{APIGroups: []string{""}, Resources: []string{"secrets"}, Verbs: []string{"get"}},
			},
		},
	}
	ApplyRuntimeDefaults(agent, rt)
	require.Len(t, agent.Deployment.RoleRules, 2)
	// Runtime rule is first.
	assert.Equal(t, []string{"secrets"}, agent.Deployment.RoleRules[0].Resources)
	// Agent rule is second.
	assert.Equal(t, []string{"configmaps"}, agent.Deployment.RoleRules[1].Resources)
}

func TestApplyRuntimeDefaults_RoleRules_AgentOnly(t *testing.T) {
	agent := &LanguageAgentSpec{
		Deployment: DeploymentSpec{
			RoleRules: []rbacv1.PolicyRule{
				{APIGroups: []string{""}, Resources: []string{"configmaps"}, Verbs: []string{"create"}},
			},
		},
	}
	rt := &LanguageAgentRuntimeSpec{}
	ApplyRuntimeDefaults(agent, rt)
	// Agent rules unchanged when runtime has none.
	require.Len(t, agent.Deployment.RoleRules, 1)
	assert.Equal(t, []string{"configmaps"}, agent.Deployment.RoleRules[0].Resources)
}
