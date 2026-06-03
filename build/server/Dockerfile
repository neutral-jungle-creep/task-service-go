# syntax=docker/dockerfile:1.7

# -------- common build --------
FROM golang:1.26.4-alpine AS builder

ENV BIN_FILE=/opt/app/bin/server
ENV CODE_DIR=/build

RUN apk add --no-cache git ca-certificates

WORKDIR ${CODE_DIR}

COPY go.mod go.sum ./
RUN go mod download

COPY . ${CODE_DIR}

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o ${BIN_FILE} ./cmd

# -------- builder for goose --------
FROM golang:1.26.4-alpine AS goose-builder
RUN apk add --no-cache git
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.22.1

# -------- runtime for app --------
FROM alpine:3.20 AS app

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S app -G app

COPY --from=builder /opt/app/bin/server /opt/app/bin/server

USER app
EXPOSE 8888
ENTRYPOINT ["/opt/app/bin/server"]

# -------- runtime for migrations --------
FROM alpine:3.20 AS migrate

RUN apk add --no-cache ca-certificates tzdata

COPY --from=goose-builder /go/bin/goose /usr/local/bin/goose
COPY db/migrations /migrations

ENV GOOSE_DRIVER=postgres
ENV GOOSE_MIGRATION_DIR=/migrations

ENTRYPOINT ["goose", "-dir", "/migrations", "up"]
