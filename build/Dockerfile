# Этап сборки (builder)
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git make gcc musl-dev
# RUN go install github.com/pressly/goose/v3/cmd/goose@latest

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/myapp ./cmd/premium_caste/main.go

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/bin/myapp /usr/local/bin/myapp
COPY --from=builder /app/config/config_dev.yaml /etc/myapp/configs/config_dev.yaml
# COPY --from=builder /go/bin/goose /usr/local/bin/goose
# COPY --from=builder /app/migrations /migrations

EXPOSE 8080

CMD ["myapp", "--config", "/etc/myapp/configs/config_dev.yaml"]
