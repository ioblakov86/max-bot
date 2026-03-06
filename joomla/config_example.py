"""
Пример файла конфигурации для управления статьями Joomla
Скопируйте этот файл в config.py и укажите свои значения
"""

# Параметры подключения к сайту Joomla
SITE_URL = "https://your-site.com"
ADMIN_URL = "https://your-site.com/administrator"

# Параметры аутентификации
USERNAME = "your_admin_username"
PASSWORD = "your_admin_password"

# Токен для доступа к PHP API обновления статей
# Должен совпадать с SECRET_TOKEN в update_article.php
API_TOKEN = "your_secret_token_here"

# Параметры сессии
REQUEST_TIMEOUT = 30  # Таймаут для HTTP-запросов в секундах

# Параметры безопасности
VERIFY_SSL = True  # Проверять SSL-сертификаты при HTTPS-запросах
