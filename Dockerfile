FROM quay.io/konveyor/crane-reverse-proxy:latest as crane-reverse-proxy
FROM quay.io/konveyor/crane-secret-service:latest as crane-secret-service
FROM quay.io/konveyor/crane-ui-plugin:latest as crane-ui-plugin
FROM quay.io/konveyor/crane-runner:latest as crane-runner

# Build the manager binary
FROM quay.io/konveyor/builder as builder

WORKDIR /go/src/github.com/konveyor/crane-operator
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o /go/src/manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM registry.access.redhat.com/ubi8-minimal
WORKDIR /
COPY --from=builder /go/src/manager .
COPY --from=crane-reverse-proxy /deploy.yaml crane-reverse-proxy.yaml
COPY --from=crane-secret-service /deploy.yaml crane-secret-service.yaml
COPY --from=crane-ui-plugin /deploy.yaml crane-ui-plugin.yaml
COPY --from=crane-runner /deploy.yaml crane-runner.yaml

USER 65532:65532

ENTRYPOINT ["/manager"]
