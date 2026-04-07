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

package controllers

import (
	"context"
	"testing"

	"github.com/language-operator/language-operator/controllers/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDeleteConfigMap_Exists(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "my-cm", Namespace: "default"},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cm).Build()

	err := DeleteConfigMap(context.Background(), c, "my-cm", "default")
	require.NoError(t, err)

	// Verify deletion
	got := &corev1.ConfigMap{}
	getErr := c.Get(context.Background(), client.ObjectKey{Name: "my-cm", Namespace: "default"}, got)
	assert.True(t, client.IgnoreNotFound(getErr) == nil && getErr != nil, "expected not-found after deletion")
}

func TestDeleteConfigMap_AlreadyGone(t *testing.T) {
	scheme := testutil.SetupTestScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()

	// ConfigMap does not exist — must return nil, not an error
	err := DeleteConfigMap(context.Background(), c, "missing-cm", "default")
	require.NoError(t, err)
}
