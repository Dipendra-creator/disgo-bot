# syntax=docker/dockerfile:1

# --- build stage ---
FROM golang:1.26-alpine AS build

# git is needed to resolve the discordgo master pseudo-version.
RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Cache dependencies separately from source for faster rebuilds.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

# Static, stripped binary. CGO off so it runs on distroless/scratch.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -trimpath -ldflags='-s -w' \
    -o /out/disgo ./cmd/bot

# --- runtime stage ---
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=build /out/disgo /app/disgo
# Ship the example so operators can mount a real config over it.
COPY --from=build /src/configs/config.example.yaml /app/configs/config.example.yaml

USER nonroot:nonroot

# Health (8080) and metrics (9090).
EXPOSE 8080 9090

ENTRYPOINT ["/app/disgo"]
