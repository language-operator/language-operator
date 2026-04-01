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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyRuntimeDefaults_Ports_AgentEmpty(t *testing.T) {
	agent := &LanguageAgentSpec{}
	rt := &LanguageAgentRuntimeSpec{
		Ports: []AgentPort{
			{Name: "http", Port: 18789, Protocol: corev1.ProtocolTCP, Expose: true},
		},
	}
	ApplyRuntimeDefaults(agent, rt)
	require.Len(t, agent.Ports, 1)
	assert.Equal(t, int32(18789), agent.Ports[0].Port)
	assert.Equal(t, "http", agent.Ports[0].Name)
	assert.True(t, agent.Ports[0].Expose)
}

func TestApplyRuntimeDefaults_Ports_AgentWins(t *testing.T) {
	agent := &LanguageAgentSpec{
		Ports: []AgentPort{
			{Name: "api", Port: 9000, Protocol: corev1.ProtocolTCP, Expose: true},
		},
	}
	rt := &LanguageAgentRuntimeSpec{
		Ports: []AgentPort{
			{Name: "http", Port: 18789, Protocol: corev1.ProtocolTCP, Expose: true},
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
