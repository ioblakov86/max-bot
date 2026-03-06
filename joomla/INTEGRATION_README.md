# Joomla Integration для Max Bot

Интеграция бота с CMS Joomla для автоматического обновления информации о больничных и отпусках врачей на сайте plk32.ru.

## Архитектура

```
┌─────────────┐     ┌──────────────┐     ┌─────────────────┐
│  Max Bot    │────►│   Joomla     │────►│  Сайт Joomla    │
│  (Go)       │     │  Analyzer    │     │  (plk32.ru)     │
│             │     │  (Python)    │     │                 │
└─────────────┘     └──────────────┘     └─────────────────┘
```

## Компоненты

### 1. Python-скрипт (joomla/joomla_analyzer.py)

Скрипт для анализа и обновления статей Joomla.

**Команды:**

```bash
# Анализ изменений (возвращает JSON с планируемыми изменениями)
python3 joomla_analyzer.py analyze --json '<JSON от бота>'

# Применение изменений
python3 joomla_analyzer.py apply --json '<JSON от бота>' --changes '<JSON изменений>'
```

**Формат JSON от бота:**
```json
{
    "Employee": {
        "Position": "терапевт участковый Т.о N2",
        "FullName": "Асеев НН"
    },
    "AbsenceType": "Больничный",
    "Dates": {
        "StartDate": "",
        "EndDate": "06.03.26"
    },
    "Status": "Окончание"
}
```

### 2. Go-пакет (joomla/client.go)

Интеграция с Python-скриптом через exec.Command.

**Основные функции:**
- `Analyze(analysis)` - анализ изменений
- `Apply(analysis, changes)` - применение изменений

### 3. Обработчик (handlers/handler.go)

Обновлённый обработчик сообщений бота.

**Поток обработки:**
1. Бот получает сообщение → AI анализ → JSON
2. Вызов `JoomlaClient.Analyze()` для получения планируемых изменений
3. Отправка админу сообщения с кнопками "Принять"/"Отмена" и списком изменений
4. При нажатии "Принять" → вызов `JoomlaClient.Apply()`
5. Отправка результата админу

## Настройка

### Переменные окружения (.env)

```env
# Joomla Integration
JOOMLA_ARTICLE_IDS=1025,1027,1028,1029
JOOMLA_SITE_URL=https://plk32.ru
JOOMLA_ADMIN_URL=https://plk32.ru/administrator
JOOMLA_USERNAME=admin
JOOMLA_PASSWORD=your_password
JOOMLA_API_TOKEN=your_api_token
```

### Требования

- Python 3.6+
- Библиотеки: `requests`, `beautifulsoup4`
- Доступ к админке Joomla

## Логика работы

### Поиск врача

1. Из JSON извлекается фамилия (первое слово из FullName)
2. Поиск в статьях по фамилии (регистронезависимо)
3. Если найдено несколько врачей с одинаковой фамилией - обновляются все

### Формат записей

**Больничный (без дат):**
```html
<br /><span style="font-weight: bold; font-style: italic; color: red;">На больничном</span>
```

**Отпуск (с датами):**
```html
<br /><span style="font-weight: bold; font-style: italic; color: red;">В отпуске с 01.01.2026 по 31.01.2026</span>
```

### Статусы

- **Начало** - добавление пометки
- **Окончание** - комментирование пометки `<!-- ... -->`
- **Продолжение** - игнорируется

## Примеры

### Анализ больничного

**Входной JSON:**
```json
{
    "Employee": {"FullName": "Асеев НН"},
    "AbsenceType": "Больничный",
    "Status": "Начало"
}
```

**Результат анализа:**
```json
{
    "success": true,
    "changes": [
        {
            "article_id": 1029,
            "doctor": "Асеев Николай Николаевич",
            "action": "add",
            "new_html": "<td>Асеев<br>Николай<br>Николаевич<br /><span style=\"...\">На больничном</span></td>"
        }
    ]
}
```

### Окончание больничного

**Входной JSON:**
```json
{
    "Employee": {"FullName": "Асеев НН"},
    "AbsenceType": "Больничный",
    "Status": "Окончание"
}
```

**Действие:**
```html
<!-- Было -->
<td>Асеев<br>Николай<br>Николаевич
    <br /><span style="...">На больничном</span>
</td>

<!-- Стало -->
<td>Асеев<br>Николай<br>Николаевич
    <!-- <br /><span style="...">На больничном</span> -->
</td>
```

## Обработка ошибок

- **Нет ID статьи** - не ищем в ней
- **Врач не найден** - пропускаем, сообщаем админу
- **Не удалось подключиться** - 3 попытки, затем сообщение админу
- **Логирование** - все действия логируются

## Структура файлов

```
joomla/
├── joomla_analyzer.py      # Основной скрипт анализа/обновления
├── client.go               # Go-клиент для интеграции
├── get_article_content.py  # Скрипт получения содержимого
├── update_article_remote.py # Скрипт обновления
├── update_article.php      # PHP-скрипт на сервере Joomla
├── config_example.py       # Шаблон конфигурации
└── README.md               # Этот файл
```

## Тестирование

### Проверка Python-скрипта

```bash
# Проверка доступности Python
python3 --version

# Проверка зависимостей
pip3 install -r requirements.txt

# Тест анализа (пример JSON)
python3 joomla_analyzer.py analyze --json '{"Employee":{"FullName":"Асеев НН"},"AbsenceType":"Больничный","Dates":{},"Status":"Начало"}'
```

### Проверка Go-кода

```bash
# Сборка
go build -o bin/max-bot ./main.go

# Запуск
./bin/max-bot
```

## Безопасность

- Не коммитьте `config.py` с реальными учётными данными
- Используйте HTTPS
- Регулярно меняйте пароли
