# Update Runtimes

Bring the runtime subchart pins in `charts/language-operator-runtimes` up to their
latest published versions.

## Context

The umbrella chart `charts/language-operator-runtimes` pulls three runtime adapters as
OCI subcharts from `oci://ghcr.io/language-operator/charts`:

| Subchart | Source repo | Published from |
|----------|-------------|----------------|
| `claude-code` | `language-operator/claude-code-adapter` | its `chart/Chart.yaml` `version` |
| `openclaw`    | `language-operator/openclaw-adapter`    | its `chart/Chart.yaml` `version` |
| `opencode`    | `language-operator/opencode-adapter`    | its `chart/Chart.yaml` `version` |

Each adapter repo publishes its chart independently (via its own `release-chart.yaml`
workflow: `helm package chart && helm push`), so the umbrella's pins drift behind the
registry over time. This command resyncs the pins.

**Scope:** only the `dependencies[].version` entries in
`charts/language-operator-runtimes/Chart.yaml` (and the resulting `Chart.lock`).

**Out of scope:**
- The runtime **images/source** — those live in their own repos and update themselves.
- The umbrella **chart version** (`version`/`appVersion`) — subchart versions are
  independent of it. The umbrella version is bumped by `/release`, which versions both
  `charts/language-operator` and `charts/language-operator-runtimes` in lockstep.

> `Chart.lock` is committed; the pulled `charts/*.tgz` are gitignored.

---

## Step 1 — Snapshot current pins

Show the versions currently pinned in `Chart.yaml` and locked in `Chart.lock`:

```bash
echo "=== Chart.yaml dependencies ==="
grep -A2 '^  - name:' charts/language-operator-runtimes/Chart.yaml
echo "=== Chart.lock ==="
cat charts/language-operator-runtimes/Chart.lock
```

Keep this "before" snapshot for the final report.

---

## Step 2 — Discover latest published versions

Query the ghcr OCI registry for the highest published semver tag of each subchart:

```bash
for chart in claude-code openclaw opencode; do
  token=$(curl -fsSL "https://ghcr.io/token?scope=repository:language-operator/charts/${chart}:pull" | jq -r .token)
  echo -n "${chart}: "
  curl -fsSL -H "Authorization: Bearer ${token}" \
    "https://ghcr.io/v2/language-operator/charts/${chart}/tags/list" \
    | jq -r '.tags[]' | grep -E '^[0-9]+\.[0-9]+\.[0-9]+$' | sort -V | tail -1 \
    || echo "FAILED"
done
```

- Non-semver tags (e.g. `latest`) are filtered out; `sort -V | tail -1` picks the highest.
- If a registry call fails (auth/private/offline), fall back to
  `helm show chart oci://ghcr.io/language-operator/charts/<chart> --version <x>` and
  report that automatic discovery could not run for that subchart — do not guess.

---

## Step 3 — Compare and update pins

For each runtime where the latest published version is **greater** than the pinned
version, use the Edit tool to bump the `version:` under that dependency block in
`charts/language-operator-runtimes/Chart.yaml`. For example:

```yaml
  - name: opencode
    version: "0.1.2"        # was "0.1.0"
    repository: oci://ghcr.io/language-operator/charts
    condition: opencode.enabled
```

If every subchart is already current, print "All runtime pins up to date." and skip to
the final report — make no further changes.

---

## Step 4 — Refresh lock and fetch

Re-resolve dependencies from the updated `Chart.yaml`. This rewrites `Chart.lock` and
re-fetches the `.tgz` archives into `charts/language-operator-runtimes/charts/`:

```bash
helm dependency update charts/language-operator-runtimes
```

(Use `update`, not `build` — `build` only installs whatever is already locked.)

---

## Step 5 — Validate

Lint and render the umbrella chart to catch any breaking change in the new subcharts:

```bash
cd charts/language-operator-runtimes && helm lint . && helm template . --debug >/dev/null
```

If lint or template fails, investigate the new subchart version (a runtime may have
changed its values schema). Report the failure rather than committing a broken pin.

---

## Step 6 — Show diff and commit

```bash
git -C charts/language-operator-runtimes diff --stat
```

Stage only the tracked files (the `.tgz` archives are gitignored):

```bash
git add charts/language-operator-runtimes/Chart.yaml charts/language-operator-runtimes/Chart.lock
```

If there are staged changes, commit:

```bash
git commit -m "chore: update runtime subchart pins"
```

---

## Step 7 — Final report

Print a before→after summary:

```
Runtime pins updated.

  claude-code : <before> → <after>
  openclaw    : <before> → <after>
  opencode    : <before> → <after>

Run `make upgrade-runtimes` to redeploy the runtimes with the new versions.
```

Note any subcharts that were already current. If nothing changed, say so and exit.
