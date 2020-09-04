# syntax=docker/dockerfile:experimental

FROM golang:1.14 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

ADD *.go ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o mutator .

# Use distroless as minimal base image to package the cmplet binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
LABEL source_repository="https://github.com/sapcc/pull-secrets-injector"
WORKDIR /
COPY --from=builder /workspace/mutator .
USER nonroot:nonroot
ENTRYPOINT ["/mutator"]
