# Build the manager binary
FROM golang:1.20 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY ./config.go config.go
COPY ./db.go db.go
COPY ./main.go main.go
COPY ./index.html index.html

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/frontend .

FROM alpine
LABEL Author="Gergely Brautigam"
RUN apk add -u ca-certificates
WORKDIR /app/
COPY --from=builder /workspace/bin/frontend /app/frontend

EXPOSE 8081

ENTRYPOINT [ "/app/frontend" ]
