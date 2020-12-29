# Use golang as intermediate image for building
FROM golang:1.15.6-alpine3.12 as builder

# Copy source to gopath
WORKDIR /go/src/github.com/Kenneth-KT/k8s-rds-autoscaler
COPY pkg/ pkg/
COPY cmd/ cmd/

# Fetch dependency
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod vendor -v

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o autoscaler github.com/Kenneth-KT/k8s-rds-autoscaler/cmd/autoscaler

# Copy the compiled binary into a thin image
FROM alpine:3.8
WORKDIR /root/
COPY --from=builder /go/src/github.com/Kenneth-KT/k8s-rds-autoscaler/autoscaler .
ENTRYPOINT ["./autoscaler"]
