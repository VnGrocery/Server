FROM golang:1.26.1-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/server ./cmd/server

FROM alpine:3.22

RUN adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/server /app/server

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/server"]
