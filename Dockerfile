# Используем официальный образ Go для сборки
FROM golang:1.23-alpine AS builder

# Устанавливаем зависимости
RUN apk add --no-cache git

# Устанавливаем рабочую директорию в контейнере
WORKDIR /app

# Копируем все файлы
COPY . .

# Инициализируем модуль (если не существует)
RUN go mod init max-bot || true
RUN go mod tidy

# Собираем бинарный файл для Linux
RUN CGO_ENABLED=0 GOOS=linux go build -o max-bot main.go

# Второй этап сборки - используем минимальный образ
FROM alpine:latest

# Устанавливаем зависимости
RUN apk --no-cache add ca-certificates

# Создаем директорию для работы приложения
WORKDIR /root/

# Копируем бинарный файл из первого образа
COPY --from=builder /app/max-bot .

# Запускаем бота
CMD ["./max-bot"]
