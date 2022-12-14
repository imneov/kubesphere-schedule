# Build the manager binary
FROM golang:1.18 as builder

ARG GOPROXY
WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.sum Makefile ./
COPY api api/
COPY cmd cmd/
COPY pkg pkg/
COPY vendor vendor/

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
#RUN go mod download
# Build
RUN CGO_ENABLED=0 GO111MODULE=on go build -a -o controller-manager cmd/controller/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/controller-manager .
USER nonroot:nonroot

ENTRYPOINT ["/controller-manager"]
