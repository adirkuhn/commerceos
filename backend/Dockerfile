# The builder runs on the build machine's own platform and cross-compiles for
# the target (TARGETOS/TARGETARCH, set by buildx: linux/arm64 for the Ampere
# server) -- no emulation needed, the binaries are pure Go (CGO off).
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder
ARG TARGETOS=linux
ARG TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
# goose pinned to the version in go.mod. go install puts a cross-compiled
# binary under /go/bin/<os>_<arch>/, a native one in /go/bin/.
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.24.3 && \
    mv "/go/bin/${GOOS}_${GOARCH}/goose" /goose 2>/dev/null || mv /go/bin/goose /goose
COPY . .
RUN go build -o /api ./cmd/api && go build -o /relay ./cmd/relay && go build -o /scheduler ./cmd/scheduler && \
    go build -o /backfill-woo-sale-prices ./cmd/backfill-woo-sale-prices && \
    go build -o /backfill-outbox-replay ./cmd/backfill-outbox-replay && \
    go build -o /backfill-webhook-dispatch ./cmd/backfill-webhook-dispatch && \
    go build -o /repair-stock-review-flags ./cmd/repair-stock-review-flags && \
    go build -o /repair-partially-received-pos ./cmd/repair-partially-received-pos && \
    go build -o /repair-unmanaged-woo-stock ./cmd/repair-unmanaged-woo-stock && \
    go build -o /webhook-processor ./cmd/webhook-processor && \
    go build -o /webhook-forwarder ./cmd/webhook-forwarder

FROM alpine:3.19
# postgresql-client (psql) is only used by the migrate service, to rotate
# commerceos_app's password after goose applies 020_app_role_and_force_rls.sql
# -- see docker-compose.yml's migrate service command.
RUN apk add --no-cache ca-certificates tzdata postgresql-client
COPY --from=builder /api /api
COPY --from=builder /relay /relay
COPY --from=builder /scheduler /scheduler
COPY --from=builder /backfill-woo-sale-prices /backfill-woo-sale-prices
COPY --from=builder /backfill-outbox-replay /backfill-outbox-replay
COPY --from=builder /backfill-webhook-dispatch /backfill-webhook-dispatch
COPY --from=builder /repair-stock-review-flags /repair-stock-review-flags
COPY --from=builder /repair-partially-received-pos /repair-partially-received-pos
COPY --from=builder /repair-unmanaged-woo-stock /repair-unmanaged-woo-stock
COPY --from=builder /webhook-processor /webhook-processor
COPY --from=builder /webhook-forwarder /webhook-forwarder
COPY --from=builder /goose /goose
# Baked in so a published image migrates by itself (deploy/prod has no
# source checkout); the dev compose still mounts the live folder over it.
COPY infrastructure/db/migrations /migrations
CMD ["/api"]
