# Используем официальный образ Go для сборки
FROM golang:1.21-alpine AS builder

# Устанавливаем зависимости
RUN apk add --no-cache git

# Устанавливаем рабочую директорию в контейнере
WORKDIR /app

# Копируем go.mod и go.sum (если они существуют)
COPY go.mod go.sum ./

# Если go.mod не существует, инициализируем модуль
RUN if [ ! -f go.mod ]; then go mod init max-bot; fi

# Копируем все исходные файлы
COPY main.go ./
COPY bot/ ./bot/
COPY handlers/ ./handlers/
COPY utils/ ./utils/

# Загружаем зависимости (если go.mod существует) или создаем tidy
RUN if [ -f go.mod ]; then go mod download; else go mod tidy; fi

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
COPY .env.example . 2>/dev/null || echo ".env.example not found"

# Открываем порт (если потребуется в будущем)
EXPOSE 8080

# Запускаем бота
CMD ["./max-bot"]
