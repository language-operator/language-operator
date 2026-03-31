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
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// validateClusterMembership verifies a LanguageCluster exists for the given namespace.
// Used by all resource webhooks to enforce namespace membership.
func validateClusterMembership(ctx context.Context, c client.Client, namespace string) error {
	cluster := &LanguageCluster{}
	err := c.Get(ctx, types.NamespacedName{Name: namespace}, cluster)
	if err == nil {
		return nil
	}
	if apierrors.IsNotFound(err) {
		return fmt.Errorf("namespace %q is not managed by a LanguageCluster: no cluster %q exists", namespace, namespace)
	}
	return fmt.Errorf("failed to check LanguageCluster for namespace %q: %w", namespace, err)
}
