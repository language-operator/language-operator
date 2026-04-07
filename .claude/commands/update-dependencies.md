# Update Dependencies

Bring all packages, Docker base images, and adapter dependencies up to date.

## Scope

| Area | Files | Tool |
|------|-------|------|
| Go modules | `src/go.mod`, `src/go.sum` | `go get -u`, `go mod tidy` |
| Node — adapters | `components/agents/*/package.json` | `npm update` |
| Node — hooks | `.claude/hooks/package.json` | `npm update` |
| Docker base images | `Dockerfile`, `components/*/Dockerfile` | manual edit |
| Go build tools | `src/Makefile` vars | manual edit |
| GitHub Actions | `.github/workflows/*.yaml` | manual edit |

---

## Step 1 — Snapshot current versions

Run in parallel:

```bash
cd src && grep -E '^(go |require)' go.mod | head -5
grep 'CONTROLLER_TOOLS_VERSION\|CRD_REF_DOCS_VERSION\|ENVTEST_K8S_VERSION' src/Makefile
grep '^FROM' Dockerfile components/agents/*/Dockerfile components/model-gateway/Dockerfile
grep -rh 'uses:' .github/workflows/*.yaml | sort -u
```

Print a compact "before" snapshot. Keep this in mind for the final diff report.

---

## Step 2 — Update Go modules

```bash
cd src && go get -u ./...
cd src && go mod tidy
```

- If `go get -u` pulls a new **major** k8s API version (e.g. `k8s.io/api` jumps from v0.29 to v0.32+), pause and report — a major k8s bump may require controller-runtime alignment and API changes.
- After tidy, verify no unexpected major-version jumps in `go.mod` by diffing against the snapshot from Step 1.

---

## Step 3 — Update Node packages

Run in parallel for each component that has a `package.json`:

```bash
cd components/agents/openclaw-adapter  && npm update && npm install
cd components/agents/opencode-adapter  && npm update && npm install
cd components/agents/claude-code-adapter && npm update && npm install
cd components/agents/claude-code-server  && npm update && npm install
cd .claude/hooks                         && npm update && npm install
```

The `npm update` command upgrades packages within semver range declared in `package.json`. It will not cross major version boundaries. If any package was bumped to a new version, note it.

---

## Step 4 — Check Docker base images

Read and display the current `FROM` lines:

```bash
grep '^FROM' Dockerfile components/agents/*/Dockerfile components/model-gateway/Dockerfile
```

For each base image, check the latest available patch/minor version by pulling the image manifest:

```bash
# Check latest node LTS major (currently 24)
docker manifest inspect node:24-alpine 2>/dev/null | grep -c 'digest' || echo "docker not available"

# Check golang latest for major in use (currently 1.25)
docker pull --quiet golang:1.25 2>&1 | tail -1 || true

# Check python 3.11
docker pull --quiet python:3.11-slim 2>&1 | tail -1 || true
```

**What to look for:**
- `golang:1.25` — if Go released a new 1.x minor (e.g. 1.26), update `Dockerfile` builder stage AND `go.mod` `go` directive, AND `src/Makefile` envtest version, AND `.github/workflows/docs.yaml` `go-version`.
- `node:24-alpine` — if Node 24 is no longer LTS or a new LTS major is released (e.g. Node 26), update all four adapter Dockerfiles.
- `python:3.11-slim` — if a newer 3.x patch is available, note it (image tag `3.11-slim` auto-tracks patches, so no change needed unless bumping minor).
- `gcr.io/distroless/static:nonroot` — no change needed; `nonroot` tag always tracks latest.

Report which images need manual version bumps and show the exact `FROM` line to replace. Do **not** edit Dockerfiles automatically — show the proposed change and ask for confirmation.

---

## Step 5 — Check Go build tool versions

Read the current pins:

```bash
grep -E 'CONTROLLER_TOOLS_VERSION|CRD_REF_DOCS_VERSION|ENVTEST_K8S_VERSION' src/Makefile
```

Check latest releases:

```bash
# controller-gen
gh release list -R kubernetes-sigs/controller-tools --limit 5

# crd-ref-docs
gh release list -R elastic/crd-ref-docs --limit 5

# envtest k8s — list recent k8s release tags
gh release list -R kubernetes/kubernetes --limit 10 | grep -v 'alpha\|beta\|rc'
```

For each tool, if a newer version exists:
- Show current vs latest
- Show the exact `src/Makefile` line to update
- Ask for confirmation before editing

---

## Step 6 — Check GitHub Actions versions

```bash
grep -rh 'uses:' .github/workflows/*.yaml | sort -u
```

For the most commonly pinned actions, check latest major:

```bash
gh release list -R actions/checkout          --limit 3
gh release list -R actions/setup-go          --limit 3
gh release list -R actions/setup-python      --limit 3
gh release list -R docker/build-push-action  --limit 3
gh release list -R docker/setup-buildx-action --limit 3
gh release list -R docker/login-action       --limit 3
gh release list -R docker/metadata-action    --limit 3
gh release list -R peaceiris/actions-gh-pages --limit 3
```

Report which actions have a newer major version available. Show the specific `uses:` lines that need updating. Do **not** edit automatically — present the diffs and ask for confirmation.

---

## Step 7 — Run tests

After all automated updates (Go modules, npm packages), run tests to verify nothing broke:

```bash
cd src && make test
```

If tests fail:
- Show the failing output.
- Investigate whether the failure is due to a dependency change (API rename, removed function, etc.).
- Fix the breakage before proceeding — do **not** revert the dependency update unless the fix would be unreasonably complex.

---

## Step 8 — Commit automated changes

Stage only the files modified by automated updates (Go modules + npm lock files):

```bash
git add src/go.mod src/go.sum
git add components/agents/openclaw-adapter/package-lock.json
git add components/agents/opencode-adapter/package-lock.json
git add components/agents/claude-code-adapter/package-lock.json
git add components/agents/claude-code-server/package-lock.json
git add .claude/hooks/package-lock.json
```

Check what changed:
```bash
git diff --cached --stat
```

If there are staged changes, commit:
```bash
git commit -m "chore: update go and npm dependencies"
```

If `src/Makefile`, Dockerfiles, or workflow files were also approved and edited in earlier steps, include them in this commit with an appropriate message.

---

## Step 9 — Final report

Print a summary:

```
Dependency update complete.

Automated changes committed:
  • Go modules: <N packages updated, list notable ones>
  • npm: <list packages updated per component>

Pending manual review:
  • Docker base images: <list images with suggested bumps>
  • Go build tools: <list tools with newer versions>
  • GitHub Actions: <list actions with newer major versions>

Run `make dev` to rebuild and redeploy with updated dependencies.
```

If everything was already up to date, report that clearly and exit.
