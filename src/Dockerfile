# Build the manager binary
FROM golang:1.23 as builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/ cmd/
COPY api/ api/
COPY controllers/ controllers/
COPY pkg/ pkg/
COPY scripts/ scripts/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager cmd/main.go

# Use base image with Ruby support for AST validation
FROM git.theryans.io/language-operator/base:latest

WORKDIR /app

# Copy manager binary from builder
COPY --from=builder /workspace/manager .

# Install language-operator gem for AST validation
RUN gem install language-operator --source https://git.theryans.io/api/packages/language-operator/rubygems

# Copy AST validator wrapper script
COPY --from=builder /workspace/scripts/validate-ruby-code.rb /usr/local/bin/validate-ruby-code.rb
RUN chmod +x /usr/local/bin/validate-ruby-code.rb

# Switch to langop user (created in base image)
USER langop:langop

ENTRYPOINT ["/app/manager"]
