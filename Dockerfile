FROM golang:1.24.10-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./

RUN go mod download

COPY . .

ARG APP_NAME=api

RUN case "$APP_NAME" in \
        api|worker) ;; \
        *) echo "APP_NAME must be api or worker" && exit 1 ;; \
    esac \
    && CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags="-s -w" \
        -o /out/app \
        "./cmd/${APP_NAME}"


FROM alpine:3.22 AS runtime

RUN apk add --no-cache ca-certificates \
    && addgroup -S app \
    && adduser -S -G app app

WORKDIR /app

COPY --from=builder /out/app /app/app

USER app

ENTRYPOINT ["/app/app"]