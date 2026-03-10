# Используем официальный образ Go для сборки
FROM golang:1.24-alpine AS builder

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

# Второй этап сборки - используем минимальный образ с Python
FROM alpine:latest

# Устанавливаем зависимости: Python, SSL сертификаты, зависимости для BeautifulSoup
RUN apk --no-cache add ca-certificates python3 py3-pip py3-beautifulsoup4 py3-requests

# Создаем директорию для работы приложения
WORKDIR /root/

# Копируем бинарный файл из первого образа
COPY --from=builder /app/max-bot .

# Копируем файл промпта из первого образа
COPY --from=builder /app/prompt.txt .

# Копируем Joomla утилиты
COPY --from=builder /app/joomla/ ./joomla/

# Запускаем бота
CMD ["./max-bot"]
