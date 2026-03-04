FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /remote-executor ./cmd/executor/

FROM alpine:3.19
RUN apk --no-cache add ca-certificates bash
COPY --from=builder /remote-executor /usr/local/bin/remote-executor
COPY scripts/ /opt/remote-executor/scripts/
COPY configs/ /etc/remote-executor/
EXPOSE 8443
VOLUME ["/opt/remote-executor/scripts", "/var/lib/remote-executor/records", "/etc/remote-executor/certs"]
ENTRYPOINT ["/usr/local/bin/remote-executor"]
CMD ["--config", "/etc/remote-executor/executor.yaml", "--scripts-dir", "/opt/remote-executor/scripts"]
