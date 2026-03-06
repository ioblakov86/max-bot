# Joomla Article Manager

Инструмент для обновления статей на сайте Joomla 3.10.12 через PHP API.

## [-] Назначение

Обновление содержимого статей Joomla (например, расписания врачей) через прямое обновление базы данных с помощью PHP-скрипта.

## [-] Быстрый старт

### 1. Настройка

1. Скопируйте конфигурацию:
   ```bash
   cp config_example.py config.py
   ```

2. Отредактируйте `config.py`:
   ```python
   SITE_URL = "https://your-site.com"
   API_TOKEN = "your_secret_token"
   ```

### 2. Загрузка PHP-скрипта на сервер

Загрузите файл `update_article.php` в корень сайта Joomla:
```bash
# Через FTP/SFTP или файловый менеджер хостинга
# Путь: /path/to/www/update_article.php
```

**Важно:** Токен в `update_article.php` (строка 15) должен совпадать с `API_TOKEN` в `config.py`.

### 3. Использование

```bash
# Получить содержимое статьи для редактирования
python3 get_article_content.py 1025 schedule.html

# Тест подключения (обновление маркером)
python3 update_article_remote.py 1025

# Обновить статью из файла
python3 update_article_remote.py 1025 schedule.html
```

## [-] Файлы проекта

| Файл | Назначение |
|------|------------|
| `update_article_remote.py` | Python-скрипт для обновления статей |
| `get_article_content.py` | Скрипт для получения содержимого статьи |
| `update_article.php` | PHP-скрипт для загрузки на сервер |
| `config.py` | Конфигурация (токен, URL) |
| `config_example.py` | Шаблон конфигурации |

## [-] Конфигурация

### config.py

```python
SITE_URL = "https://your-site.com"      # URL сайта
ADMIN_URL = "https://your-site.com/administrator"  # Админка
API_TOKEN = "your_secret_token"         # Токен доступа
REQUEST_TIMEOUT = 30                     # Таймаут запросов
VERIFY_SSL = True                        # Проверка SSL
```

### update_article.php

Измените токен в начале файла (строка 15):
```php
define('SECRET_TOKEN', 'your_secret_token');
```

## [-] Примеры

### Рабочий процесс обновления статьи

1. **Получить текущее содержимое:**
   ```bash
   python3 get_article_content.py 1025 schedule.html
   ```

2. **Отредактировать файл `schedule.html`:**
   - Открыть в любом текстовом редакторе
   - Внести изменения (добавить пометки об отпуске/больничном)
   - Сохранить в UTF-8

3. **Обновить статью на сервере:**
   ```bash
   python3 update_article_remote.py 1025 schedule.html
   ```

### Обновление расписания врачей

1. Подготовьте HTML-файл с новым содержимым:
```html
<table class="uk-table">
    <tr>
        <td>Иванов Иван Иванович</td>
        <td>На ЭЛН с 05.03.2026</td>
    </tr>
</table>
```

2. Обновите статью:
```bash
python3 update_article_remote.py 1025 schedule.html
```

### Массовое обновление

```bash
#!/bin/bash
for id in 1025 1026 1027; do
    python3 update_article_remote.py $id schedule.html
done
```

### Через CURL напрямую

```bash
# Обновить статью
curl -X POST "https://your-site.com/update_article.php?token=YOUR_TOKEN&id=1025" \
     -d "articletext=<p>Новое содержимое</p>"
```

## [-] Безопасность

### ВНИМАНИЕ: Важно!

После завершения работы **удалите `update_article.php` с сервера**!

Этот скрипт даёт прямой доступ к обновлению статей без дополнительной аутентификации.

### Рекомендации

1. Используйте сложный токен (минимум 32 символа)
2. Не храните токен в публичном доступе
3. Удаляйте скрипт после использования
4. Для постоянного использования рассмотрите Joomla Web Services API

## [-] Требования

- Python 3.6+
- Библиотека `requests`

Установка зависимостей:
```bash
pip install -r requirements.txt
```

## [-] Структура статьи

Содержимое сохраняется в поле `introtext` таблицы `#__content`.

Формат HTML:
```html
<script>
    // JavaScript для интерактивности (опционально)
</script>
<p>Описание</p>
<table>
    <tbody>
        <tr>
            <td>Данные</td>
        </tr>
    </tbody>
</table>
```

## [-] Решение проблем

### "Ошибка: неверный токен доступа"
Проверьте, что токен в `config.py` совпадает с `SECRET_TOKEN` в `update_article.php`.

### "Статья с ID не найдена"
Убедитесь, что статья с таким ID существует в базе данных Joomla.

### "Ошибка соединения"
Проверьте доступность сайта и SSL-сертификаты.

### Содержимое не обновляется
- Убедитесь, что файл сохранён в UTF-8
- Проверьте размер содержимого (лимит PHP)

## [-] Лицензия

MIT

---

**Версия:** 1.0  
**Joomla:** 3.10.12  
**Последнее обновление:** 2026-03-05
