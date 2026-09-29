# First-party git client image for LanguageAgent init containers.
#
# The operator runs two init containers from this image: `workspace-seeder` (copies
# spec.workspace seed files onto the workspace PVC) and `repository` (clones
# spec.repository into the workspace). Both run under the operator's hardened pod
# securityContext (runAsNonRoot, runAsUser=1000, fsGroup=101).
#
# Why not alpine/git directly:
#   - alpine/git:latest is a mutable tag, and the repository init container has the
#     agent's git credentials mounted. The base is pinned by digest here and the
#     resulting image is tagged per operator release (language-operator/language-operator#917).
#   - alpine/git has no passwd entry for uid 1000. OpenSSH refuses to start as a uid it
#     cannot resolve ("No user exists for uid 1000"), so every SSH clone failed. This
#     image adds that user with HOME=/tmp (language-operator/language-operator#908).
#
# Bump the base by editing the tag AND the digest together:
#   docker buildx imagetools inspect alpine/git:<tag>
FROM alpine/git:v2.54.0@sha256:832b1cd1a271509f3d5272a1a62d4cb2ab1a53426ebde1f6c2cb7349f907dc6f

# uid/gid match controllers.LangopUserID / LangopGroupID. -H: /tmp already exists.
RUN addgroup -g 101 langop \
    && adduser -D -H -u 1000 -G langop -h /tmp -s /bin/sh langop

ENV HOME=/tmp
WORKDIR /tmp
USER 1000

# Kubernetes overrides this with container.command; the ENTRYPOINT is for local `docker run`.
ENTRYPOINT ["git"]
