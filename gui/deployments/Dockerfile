FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /remote-gui ./cmd/gui/

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /remote-gui /usr/local/bin/remote-gui
COPY configs/ /etc/remote-gui/
VOLUME ["/etc/remote-gui/certs"]
ENTRYPOINT ["/usr/local/bin/remote-gui"]
CMD ["--config", "/etc/remote-gui/gui.yaml"]
