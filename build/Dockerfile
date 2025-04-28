# Этап сборки (builder)
FROM golang:1.24-alpine AS builder

# Установка зависимостей для сборки
RUN apk add --no-cache git make gcc musl-dev

# Рабочая директория
WORKDIR /app

# Копируем файлы зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь код
COPY . .

# Сборка приложения
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/myapp ./cmd/premium_caste/main.go

# Финальный этап (минимальный образ)
FROM alpine:3.21

# Установка runtime зависимостей (если нужны)
RUN apk add --no-cache ca-certificates tzdata

# Копируем только бинарник из builder-этапа
COPY --from=builder /app/bin/myapp /usr/local/bin/myapp

# Дополнительные файлы (конфиги, миграции и т.д.)
COPY --from=builder /app/config/config_dev.yaml /etc/myapp/configs/config_dev.yaml

# Порт приложения
EXPOSE 8080

# Команда запуска
CMD ["myapp", "--config", "/etc/myapp/configs/config_dev.yaml"]
