# git

First-party **git client** image for LanguageAgent init containers.

The operator runs two init containers from this image, both under the hardened pod
securityContext (`runAsNonRoot`, `runAsUser: 1000`, `fsGroup: 101`):

- `workspace-seeder` — copies `spec.workspace.initialFiles` / `seedConfigMapRef` onto the
  workspace PVC (seed-once).
- `repository` — clones `spec.repository` into the workspace (clone-once), with the
  credentials Secret mounted read-only at `/var/run/secrets/langop.io/git`.

## Why this image

- **Pinned.** `alpine/git:latest` is a mutable tag, and the `repository` container has the
  agent's git token or SSH key in reach. The base here is pinned by digest, and the operator
  chart pins this image to its own release tag (`config.git.tag` defaults to the chart
  `appVersion`), so it only changes when you bump the chart.
- **SSH works as uid 1000.** `alpine/git` has no passwd entry for uid 1000, and OpenSSH refuses
  to run as a uid it cannot resolve (`No user exists for uid 1000`), so every SSH clone failed.
  This image adds a `langop` user (uid 1000, gid 101, `HOME=/tmp`).

Override with `config.git.repository` / `config.git.tag` (Helm) or `--git-image` (operator
flag). Any image with `git`, `ssh`, `/bin/sh` and a passwd entry for uid 1000 works.

## Build / develop

```bash
make build        # build ghcr.io/language-operator/git:latest
make dev          # build + import into k3s
make test         # verify uid 1000 resolves and git/ssh are present
make push         # publish
```

For the local `make dev` loop the operator chart's `values.local.yaml` sets `config.git.tag: latest`
so the imported image is used instead of the appVersion-pinned tag.

Bump the base by editing the tag and digest together in the `Dockerfile`
(`docker buildx imagetools inspect alpine/git:<tag>` prints the index digest).
