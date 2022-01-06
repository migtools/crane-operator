# Build the manager binary
FROM quay.io/konveyor/builder as builder

WORKDIR /go/src/github.com/konveyor/mtk-operator
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
COPY deploy/artifacts/manifests.yaml manifests.yaml
COPY deploy/artifacts/crane-ui-plugin.yaml crane-ui-plugin.yaml

USER 65532:65532

ENTRYPOINT ["/manager"]