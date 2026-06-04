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
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"

	langopv1alpha1 "github.com/language-operator/language-operator/api/v1alpha1"
)

// dexWebFS embeds the branded Language Operator web assets that the operator
// mounts into every Dex pod. The directory layout matches what Dex's
// frontend.dir loader expects: templates/, static/, themes/<theme>/, and a
// robots.txt at the root.
//
//go:embed all:web
var dexWebFS embed.FS

// dexWebPathSeparator is the substring used in ConfigMap keys to encode the
// '/' character (which Kubernetes does not allow in ConfigMap keys). The
// volume's Items list maps each key back to its real relative path.
const dexWebPathSeparator = "__"

var (
	// dexWebData holds the contents of every embedded file, keyed by
	// path-with-slashes-encoded-as-dexWebPathSeparator.
	dexWebData map[string]string
	// dexWebItems describes how each ConfigMap key maps to a relative path
	// inside the mounted volume, recreating the templates/static/themes
	// directory structure that Dex expects.
	dexWebItems []corev1.KeyToPath
	// dexWebHash is a content hash over the embedded assets, set as a pod
	// template annotation so Dex restarts when the operator ships new assets.
	dexWebHash string
)

func init() {
	data, items, hash, err := loadDexWebAssets()
	if err != nil {
		panic(fmt.Sprintf("language-operator: failed to load embedded dex web assets: %v", err))
	}
	dexWebData = data
	dexWebItems = items
	dexWebHash = hash
}

// loadDexWebAssets walks the embedded web/ tree and returns ConfigMap data,
// the items list that recreates the directory structure on mount, and a
// content hash for the rollout annotation.
func loadDexWebAssets() (map[string]string, []corev1.KeyToPath, string, error) {
	data := map[string]string{}
	var paths []string

	err := fs.WalkDir(dexWebFS, "web", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, readErr := fs.ReadFile(dexWebFS, p)
		if readErr != nil {
			return readErr
		}
		rel := strings.TrimPrefix(p, "web/")
		if strings.Contains(rel, dexWebPathSeparator) {
			return fmt.Errorf("embedded dex asset path %q contains reserved substring %q", rel, dexWebPathSeparator)
		}
		key := strings.ReplaceAll(rel, "/", dexWebPathSeparator)
		data[key] = string(b)
		paths = append(paths, rel)
		return nil
	})
	if err != nil {
		return nil, nil, "", err
	}
	if len(paths) == 0 {
		return nil, nil, "", fmt.Errorf("no embedded dex web assets found")
	}

	sort.Strings(paths)

	items := make([]corev1.KeyToPath, 0, len(paths))
	h := sha256.New()
	for _, rel := range paths {
		key := strings.ReplaceAll(rel, "/", dexWebPathSeparator)
		items = append(items, corev1.KeyToPath{Key: key, Path: rel})

		h.Write([]byte(rel))
		h.Write([]byte{0})
		h.Write([]byte(data[key]))
		h.Write([]byte{0})
	}

	return data, items, fmt.Sprintf("%x", h.Sum(nil)), nil
}

// reconcileDexTemplatesConfigMap ensures the ConfigMap holding the branded
// Dex web assets exists and is up-to-date. Returns the content hash so the
// Deployment can roll its pods when the embedded assets change.
func (r *LanguageClusterReconciler) reconcileDexTemplatesConfigMap(ctx context.Context, cluster *langopv1alpha1.LanguageCluster, namespace string) (string, error) {
	if err := CreateOrUpdateConfigMap(ctx, r.Client, r.Scheme, cluster, DexTemplatesConfigMapName, namespace, dexWebData); err != nil {
		return "", fmt.Errorf("failed to reconcile dex templates ConfigMap: %w", err)
	}
	return dexWebHash, nil
}
