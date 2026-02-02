# Используем официальный образ Go для сборки
FROM golang:1.21-alpine AS builder

# Устанавливаем зависимости
RUN apk add --no-cache git

# Устанавливаем рабочую директорию в контейнере
WORKDIR /app

# Копируем только main.go сначала
COPY main.go ./
COPY bot/ ./bot/
COPY handlers/ ./handlers/
COPY utils/ ./utils/

# Инициализируем модуль Go
RUN go mod init max-bot

# Добавляем необходимые зависимости
RUN go get github.com/joho/godotenv
RUN go get github.com/max-messenger/max-bot-api-client-go
RUN go get github.com/max-messenger/max-bot-api-client-go/schemes

# Завершаем настройку модуля
RUN go mod tidy

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

# Открываем порт (если потребуется в будущем)
EXPOSE 8080

# Запускаем бота
CMD ["./max-bot"]
