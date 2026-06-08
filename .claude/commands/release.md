# Release: Bump version, tag, and publish a GitHub release

## Arguments

`$ARGUMENTS` is the bump type or an explicit version:

- `patch` — increment patch: `0.1.125` → `0.1.126` *(default if omitted)*
- `minor` — increment minor: `0.1.125` → `0.2.0`
- `major` — increment major: `0.1.125` → `1.0.0`
- `x.y.z` — pin to an exact version (no `v` prefix)

## Context

- Version source of truth: the `version` and `appVersion` fields of **both** umbrella charts — `charts/language-operator/Chart.yaml` and `charts/language-operator-runtimes/Chart.yaml`. All four fields must stay in sync; treat `charts/language-operator/Chart.yaml` as canonical when reading the current version.
- Runtime **subchart** pins (`claude-code`, `openclaw`, `opencode`) inside `charts/language-operator-runtimes/Chart.yaml` are versioned independently and are **not** touched here — use `/update-runtimes` for those.
- Release trigger: pushing a `v*` git tag kicks off CI to build all Docker images and create a GitHub release with the packaged Helm chart as an asset
- CHANGELOG: `CHANGELOG.md` has an `## Unreleased` section at the top; on release, move its content into a new `## v{version} — {date}` section

## Instructions

### Step 1 — Pre-flight checks

Run these in parallel and abort (with a clear message) if any fail:

```bash
# Must be on main branch
git rev-parse --abbrev-ref HEAD

# Working tree must be clean
git status --porcelain

# Local main must be up-to-date with origin
git fetch origin main --dry-run 2>&1 || true
git rev-list HEAD..origin/main --count
```

- If not on `main`: abort — "checkout main before releasing"
- If working tree is dirty: abort — "commit or stash changes before releasing"
- If behind origin/main: abort — "pull latest main before releasing"

### Step 2 — Determine the new version

Read the current version:

```bash
grep '^version:' charts/language-operator/Chart.yaml | awk '{print $2}'
```

Sanity-check that the runtimes chart is currently in sync (it should print the same version):

```bash
grep '^version:' charts/language-operator-runtimes/Chart.yaml | awk '{print $2}'
```

Parse the three semver components (MAJOR.MINOR.PATCH) and apply the bump from `$ARGUMENTS`:
- `patch` (or empty): PATCH += 1
- `minor`: MINOR += 1, PATCH = 0
- `major`: MAJOR += 1, MINOR = 0, PATCH = 0
- explicit `x.y.z`: use as-is (validate it is strictly greater than the current version)

Check that the tag `v{new_version}` does not already exist:
```bash
git tag --list "v{new_version}"
```
If it does, abort with a clear error.

### Step 3 — Generate release notes from commits

Find the previous release tag and collect commits since then:

```bash
# Previous tag (the one before the upcoming release)
git describe --tags --abbrev=0 2>/dev/null || echo "none"

# Commits since that tag, excluding merge commits
git log {prev_tag}..HEAD --oneline --no-merges
```

Format the commits into markdown grouped by conventional commit type:

| Prefix | Section heading |
|--------|----------------|
| `feat:` / `feat(...):`   | **Features** |
| `fix:` / `fix(...):`     | **Bug Fixes** |
| `docs:` / `docs(...):`   | **Documentation** |
| `test:` / `test(...):`   | **Tests** |
| `refactor:` / `refactor(...):` | **Refactoring** |
| `chore:` / `chore(...):`  | **Chores** |
| anything else            | **Other** |

- Strip the conventional commit prefix from each bullet (show only the description).
- Omit empty sections.
- If the previous tag is "none" (first release), use all commits in the repo.

Call this result `{generated_notes}`.

### Step 4 — Show the release plan and ask for confirmation

Print a concise summary before touching anything:

```
Release plan:
  Current version : {current}
  New version     : {new_version}
  Git tag         : v{new_version}

Changes that will be made:
  • charts/language-operator/Chart.yaml            version + appVersion → {new_version}
  • charts/language-operator-runtimes/Chart.yaml   version + appVersion → {new_version}
  • CHANGELOG.md                                   ## Unreleased → ## v{new_version} — {today}
  • git commit         chore: release v{new_version}
  • git tag            v{new_version} (annotated)
  • git push           origin main + tag

Release notes (from {commit_count} commits since {prev_tag}):

{generated_notes}

CI will then build Docker images and create the GitHub release automatically.

Proceed? (yes/no)
```

Wait for the user to confirm before proceeding. If they say no, abort cleanly.

### Step 5 — Update both umbrella `Chart.yaml` files

Use the Edit tool to replace both `version:` and `appVersion:` lines in **each** of
`charts/language-operator/Chart.yaml` **and** `charts/language-operator-runtimes/Chart.yaml`:
- `version: {current}` → `version: {new_version}`
- `appVersion: "{current}"` → `appVersion: "{new_version}"`

Do **not** touch the `dependencies[].version` pins in the runtimes chart — those are the
runtime subchart versions and are managed by `/update-runtimes`.

### Step 6 — Update `CHANGELOG.md`

Read the file and locate `## Unreleased`.

Replace the `## Unreleased` heading with:

```
## Unreleased

---

## v{new_version} — {today}

{generated_notes}
```

**If the Unreleased section already has manual content** (lines between `## Unreleased` and the next `##` heading), place that content *after* the generated notes under the new version heading, separated by a blank line.

**If the Unreleased section is empty**, use only the generated notes as the version section content.

Use today's date in `YYYY-MM-DD` format (available via `currentDate` in context).

### Step 7 — Commit the changes

Stage only the modified files:

```bash
git add charts/language-operator/Chart.yaml charts/language-operator-runtimes/Chart.yaml CHANGELOG.md
```

Commit with a conventional commit message:

```bash
git commit -m "chore: release v{new_version}"
```

### Step 8 — Create an annotated tag

Annotated tags (vs lightweight) are the GitHub best practice for releases — they carry metadata and show up correctly in `git describe`.

```bash
git tag -a "v{new_version}" -m "Release v{new_version}"
```

### Step 9 — Push commit and tag

```bash
git push origin main
git push origin "v{new_version}"
```

### Step 10 — Report and link to CI

After pushing, print:

```
Released v{new_version}

  Tag     : v{new_version}
  Commit  : {short SHA}

CI is now building Docker images and will create the GitHub release.
Watch progress:

  gh run list --branch main --limit 5
  gh run watch $(gh run list --branch main --limit 1 --json databaseId -q '.[0].databaseId')
```

Also show the expected GitHub release URL based on the repo remote.

## Error handling

- If any git command fails, print the error and stop — do not proceed to the next step.
- If the user interrupts mid-process (e.g. after the commit but before the push), note the state clearly so they can recover manually.
- If the tag push fails (e.g. already exists remotely), print the exact `git push` error and stop.
