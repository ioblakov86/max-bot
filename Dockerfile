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

# Аргументы для передачи переменных окружения при сборке
ARG JOOMLA_SITE_URL=https://plk32.ru
ARG JOOMLA_ADMIN_URL=https://plk32.ru/administrator
ARG JOOMLA_USERNAME=
ARG JOOMLA_PASSWORD=
ARG JOOMLA_API_TOKEN=

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

# Создаём config.py из аргументов сборки
RUN echo "# Auto-generated config from build args" > joomla/config.py && \
    echo "SITE_URL = '${JOOMLA_SITE_URL}'" >> joomla/config.py && \
    echo "ADMIN_URL = '${JOOMLA_ADMIN_URL}'" >> joomla/config.py && \
    echo "USERNAME = '${JOOMLA_USERNAME}'" >> joomla/config.py && \
    echo "PASSWORD = '${JOOMLA_PASSWORD}'" >> joomla/config.py && \
    echo "API_TOKEN = '${JOOMLA_API_TOKEN}'" >> joomla/config.py && \
    echo "REQUEST_TIMEOUT = 30" >> joomla/config.py && \
    echo "VERIFY_SSL = True" >> joomla/config.py && \
    echo "USER_AGENT = 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36'" >> joomla/config.py && \
    echo "ARTICLE_IDS = [1025, 1027, 1028, 1029]" >> joomla/config.py

# Запускаем бота
CMD ["./max-bot"]
