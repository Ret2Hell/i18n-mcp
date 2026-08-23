# argument for Go version
ARG GO_VERSION=1.26.7

FROM golang:${GO_VERSION}-alpine@sha256:28d89ee9cc0ff9fec75c82ca201e6bf7fdf9a679d4b7b24dfa04f2bb766bb468 AS builder

# Create user in builder (these tools don't exist in scratch)
RUN adduser -D -g '' -u 1000 appuser

WORKDIR /app

ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/

RUN CGO_ENABLED=0 GOOS=linux go build \
  -ldflags="-s -w -X github.com/Ret2Hell/i18n-mcp/internal/version.Version=${VERSION} -X github.com/Ret2Hell/i18n-mcp/internal/version.Commit=${COMMIT} -X github.com/Ret2Hell/i18n-mcp/internal/version.Date=${DATE}" \
  -installsuffix 'static' \
  -o bin/i18n-mcp \
  ./cmd/i18n-mcp

FROM scratch AS final

WORKDIR /app

# Copy the static binary
COPY --from=builder /app/bin/i18n-mcp /bin/i18n-mcp

# Copy user database so we can run as non-root by name
COPY --from=builder /etc/passwd /etc/passwd

USER appuser

ENTRYPOINT ["/bin/i18n-mcp"]
