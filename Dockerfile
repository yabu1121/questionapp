# syntax=docker/dockerfile:1

FROM golang:1.27-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 \
    GOOS=linux \
    go build \
    -trimpath \
    -o /out/jev-api \
    ./cmd/api


FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /app

COPY --from=builder --chown=nonroot:nonroot \
    /out/jev-api \
    /app/jev-api

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/app/jev-api"]
