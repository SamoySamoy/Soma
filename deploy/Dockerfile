# syntax=docker/dockerfile:1

# --- Web app ---------------------------------------------------------------
FROM node:24-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci
COPY api/openapi.yaml /src/api/openapi.yaml
COPY web/ ./
RUN npm run build

# --- Go binary -------------------------------------------------------------
FROM golang:1.27-alpine AS go
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
COPY --from=web /src/web/dist ./web/dist
ARG VERSION=dev
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath \
      -ldflags "-s -w -X github.com/SamoySamoy/Soma/internal/buildinfo.Version=${VERSION}" \
      -o /out/soma ./cmd/soma

# --- Runtime: static binary on distroless, non-root ------------------------
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=go /out/soma /soma
ENV SOMA_HTTP_ADDR=0.0.0.0:8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/soma"]
CMD ["serve"]
