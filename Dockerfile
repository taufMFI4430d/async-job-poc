FROM node:24-alpine AS ui-builder

WORKDIR /src/web

COPY web/package.json web/package-lock.json ./
RUN npm ci --ignore-scripts

COPY web/ ./
RUN npm run build


FROM golang:1.24.10-alpine AS go-builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/

RUN CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags="-s -w" \
        -o /out/api \
        ./cmd/api \
    && CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags="-s -w" \
        -o /out/worker \
        ./cmd/worker


FROM alpine:3.22 AS runtime-base

RUN apk add --no-cache ca-certificates \
    && addgroup -S app \
    && adduser -S -G app app

WORKDIR /app
USER app


FROM runtime-base AS api

COPY --from=go-builder /out/api /app/app
COPY --from=ui-builder /src/web/dist /app/web/dist

ENTRYPOINT ["/app/app"]


FROM runtime-base AS worker

COPY --from=go-builder /out/worker /app/app

ENTRYPOINT ["/app/app"]
