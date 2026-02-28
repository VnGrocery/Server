FROM golang:1.26.1-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
COPY vendor ./vendor

COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOFLAGS=-mod=vendor go build -o /out/server ./cmd/server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates && update-ca-certificates
RUN adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/server /app/server

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/server"]
