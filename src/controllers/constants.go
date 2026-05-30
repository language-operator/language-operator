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

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// FinalizerName is the operator finalizer added to all managed resources.
	FinalizerName = "langop.io/finalizer"

	// GatewayResourceName is the name used for gateway Deployment, Service, and ConfigMap.
	GatewayResourceName = "gateway"

	// LangopUserID is the user ID for the langop user (matches Dockerfile).
	LangopUserID = 1000
	// LangopGroupID is the group ID for the langop group.
	LangopGroupID = 101

	// DexResourceName is the name used for Dex OIDC provider resources.
	DexResourceName = "auth"
	// DexConfigMapName is the name of the Dex configuration ConfigMap.
	DexConfigMapName = "auth-dex-config"
	// DexClientSecretName is the name of the Secret holding the shared oauth2-proxy client secret.
	DexClientSecretName = "auth-client-secret"
	// DexClientSecretKey is the key within auth-client-secret holding the secret value.
	DexClientSecretKey = "secret"
	// DexPort is the port Dex listens on inside the pod.
	DexPort = int32(5556)
	// OAuth2ProxySuffix is the suffix appended to agent names for oauth2-proxy resources.
	OAuth2ProxySuffix = "-oauth2-proxy"
	// OAuth2ProxyPort is the port oauth2-proxy listens on.
	OAuth2ProxyPort = int32(4180)
	// OAuth2ProxyCookieSecretKey is the key within the per-agent cookie-secret Secret.
	OAuth2ProxyCookieSecretKey = "cookie-secret"
)

// RemoveFinalizer removes the operator finalizer from obj and updates it.
func RemoveFinalizer(ctx context.Context, c client.Client, obj client.Object) error {
	if controllerutil.ContainsFinalizer(obj, FinalizerName) {
		controllerutil.RemoveFinalizer(obj, FinalizerName)
		if err := c.Update(ctx, obj); err != nil {
			log.FromContext(ctx).Error(err, "Failed to remove finalizer")
			return err
		}
	}
	return nil
}
