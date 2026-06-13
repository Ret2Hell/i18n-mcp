# argument for Go version
ARG GO_VERSION=1.26.4

FROM golang:${GO_VERSION}-alpine@sha256:7a3e50096189ad57c9f9f865e7e4aa8585ed1585248513dc5cda498e2f41812c AS builder

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

CMD ["/bin/i18n-mcp"]
