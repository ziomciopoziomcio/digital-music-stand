# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY contracts/go.mod contracts/go.sum* ./contracts/
COPY server/go.mod server/go.sum ./server/

RUN go work init ./contracts ./server

COPY contracts/ ./contracts/

RUN cd server && go mod download

COPY server/ ./server/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server-bin ./server/cmd/api

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/server-bin /app/server-bin
COPY --from=builder /app/server/internal/admin/web /app/server/internal/admin/web

EXPOSE 8080 50051

ENTRYPOINT ["/app/server-bin"]