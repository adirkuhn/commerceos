FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
RUN go install github.com/pressly/goose/v3/cmd/goose@latest
COPY . .
RUN go build -o /api ./cmd/api && go build -o /relay ./cmd/relay && go build -o /scheduler ./cmd/scheduler

FROM alpine:3.19
# postgresql-client (psql) is only used by the migrate service, to rotate
# commerceos_app's password after goose applies 020_app_role_and_force_rls.sql
# -- see docker-compose.yml's migrate service command.
RUN apk add --no-cache ca-certificates tzdata postgresql-client
COPY --from=builder /api /api
COPY --from=builder /relay /relay
COPY --from=builder /scheduler /scheduler
COPY --from=builder /go/bin/goose /goose
CMD ["/api"]
