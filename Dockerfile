# Используем официальный образ Go для сборки
FROM golang:1.21-alpine AS builder

# Устанавливаем зависимости
RUN apk add --no-cache git

# Устанавливаем рабочую директорию в контейнере
WORKDIR /app

# Копируем go.mod и go.sum для загрузки зависимостей
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код в контейнер
COPY . .

# Собираем бинарный файл для Linux
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o max-bot main.go

# Второй этап сборки - используем минимальный образ
FROM alpine:latest

# Устанавливаем зависимости
RUN apk --no-cache add ca-certificates

# Создаем директорию для работы приложения
WORKDIR /root/

# Копируем бинарный файл из первого образа
COPY --from=builder /app/max-bot .

# Копируем .env.example как шаблон
COPY --from=builder /app/.env.example .

# Открываем порт (если потребуется в будущем)
EXPOSE 8080

# Запускаем бота
CMD ["./max-bot"]