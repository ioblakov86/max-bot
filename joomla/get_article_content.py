#!/usr/bin/env python3
"""
Скрипт для получения содержимого статьи из Joomla и сохранения в HTML-файл

Использование:
    python3 get_article_content.py [article_id] [output_file]

Примеры:
    # Получить статью 1025 в schedule.html
    python3 get_article_content.py 1025 schedule.html

    # Получить статью в другой файл
    python3 get_article_content.py 1007 article.html
"""

import requests
from bs4 import BeautifulSoup
import json
import sys
import config


def login(session):
    """Аутентификация в Joomla админке"""
    login_url = f"{config.ADMIN_URL}/index.php"
    response = session.get(login_url, timeout=config.REQUEST_TIMEOUT, verify=config.VERIFY_SSL)
    soup = BeautifulSoup(response.text, 'html.parser')

    # Получаем CSRF токен
    csrf_token = None
    script_tag = soup.find('script', {'class': 'joomla-script-options'})
    if script_tag:
        try:
            json_data = json.loads(script_tag.string)
            csrf_token = json_data.get('csrf.token')
        except:
            pass

    if not csrf_token:
        token_elem = soup.find('input', {'name': lambda x: x and len(x) == 32 and x.isalnum()})
        if token_elem:
            csrf_token = token_elem['name']

    login_data = {
        'username': config.USERNAME,
        'passwd': config.PASSWORD,
        'lang': 'ru-RU',
        'option': 'com_login',
        'task': 'login',
    }
    if csrf_token:
        login_data[csrf_token] = '1'

    session.post(login_url, data=login_data, timeout=config.REQUEST_TIMEOUT, verify=config.VERIFY_SSL)
    return csrf_token


def get_article_content(session, article_id):
    """Получение содержимого статьи через форму редактирования"""
    edit_url = f"{config.ADMIN_URL}/index.php?option=com_content&task=article.edit&id={article_id}"
    response = session.get(edit_url, timeout=config.REQUEST_TIMEOUT, verify=config.VERIFY_SSL)
    soup = BeautifulSoup(response.text, 'html.parser')

    # Ищем содержимое в textarea jform_articletext
    article_textarea = soup.find('textarea', {'id': 'jform_articletext'})
    
    if article_textarea and article_textarea.text:
        return article_textarea.text.strip()
    
    # Альтернативно: ищем в данных страницы
    return None


def save_to_file(content, filename):
    """Сохранение содержимого в файл"""
    with open(filename, 'w', encoding='utf-8') as f:
        f.write(content)


def main():
    if len(sys.argv) < 2:
        print("Использование:")
        print(f"  {sys.argv[0]} [article_id] [output_file]")
        print("\nПримеры:")
        print(f"  {sys.argv[0]} 1025 schedule.html")
        print(f"  {sys.argv[0]} 1007 article.html")
        sys.exit(1)

    article_id = sys.argv[1]
    output_file = sys.argv[2] if len(sys.argv) > 2 else 'schedule.html'

    print(f"Получение статьи ID: {article_id}")

    session = requests.Session()
    session.headers.update({'User-Agent': config.USER_AGENT})

    # Логин
    print("Аутентификация в Joomla...")
    login(session)
    print("Успешный вход!")

    # Получение содержимого
    print(f"Получение содержимого статьи {article_id}...")
    content = get_article_content(session, article_id)

    if content:
        print(f"Получено {len(content)} символов")

        # Сохранение в файл
        save_to_file(content, output_file)
        print(f"Содержимое сохранено в '{output_file}'")

        # Краткая статистика
        print(f"\nСтатистика:")
        print(f"   Размер: {len(content)} символов")
        print(f"   Строк: {content.count(chr(10)) + 1}")
        print(f"   Тегов <tr>: {content.count('<tr>')}")

    else:
        print("Не удалось получить содержимое статьи")
        sys.exit(1)


if __name__ == "__main__":
    main()
