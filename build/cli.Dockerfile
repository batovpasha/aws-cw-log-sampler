FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk update && \
    apk --no-cache add \
        ca-certificates \
        git \
        tzdata && \
    /usr/sbin/update-ca-certificates

COPY cmd/cli cmd/cli
COPY internal/ internal/
COPY go.mod go.mod
COPY go.sum go.sum

ARG VERSION
RUN : "${VERSION:?VERSION build arg is required}"

RUN CGO_ENABLED=0 go build -a -tags netgo,osusergo \
    -ldflags "-extldflags '-static' -s -w" \
    -ldflags "-X main.version=${VERSION}" \
    -o cli ./cmd/cli

FROM scratch

WORKDIR /app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/cli cli

ENTRYPOINT ["/app/cli"]