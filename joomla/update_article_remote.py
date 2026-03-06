#!/usr/bin/env python3
"""
Скрипт для обновления статьи на сервере Joomla через PHP API

Использование:
    python3 update_article_remote.py [article_id] [content_file]

Примеры:
    # Тест подключения (без изменения содержимого)
    python3 update_article_remote.py 1025

    # Обновить статью из файла
    python3 update_article_remote.py 1025 schedule.html
"""

import requests
import sys
import os
import config


def update_article(article_id, articletext=None):
    """
    Обновить статью на сервере

    Args:
        article_id: ID статьи
        articletext: Новое содержимое (HTML) или None для теста

    Returns:
        bool: True если успешно
    """
    url = f"{config.SITE_URL}/update_article.php"

    params = {
        'token': config.API_TOKEN,
        'id': article_id
    }

    data = {
        'token': config.API_TOKEN,
        'id': str(article_id)
    }

    if articletext is not None:
        data['articletext'] = articletext

    try:
        response = requests.post(
            url,
            params=params,
            data=data,
            timeout=config.REQUEST_TIMEOUT,
            verify=config.VERIFY_SSL
        )

        print("=" * 60)
        print("Ответ сервера:")
        print("=" * 60)
        print(response.text)
        print("=" * 60)

        if "Статья успешно обновлена!" in response.text:
            print("\nОбновление прошло успешно!")
            return True
        else:
            print("\nОшибка обновления!")
            return False

    except requests.exceptions.RequestException as e:
        print(f"Ошибка соединения: {e}")
        return False


def main():
    if len(sys.argv) < 2:
        print("Использование:")
        print(f"  {sys.argv[0]} [article_id] [content_file]")
        print("\nПримеры:")
        print(f"  {sys.argv[0]} 1025  # Тест без изменения содержимого")
        print(f"  {sys.argv[0]} 1025 schedule.html  # Обновить из файла")
        sys.exit(1)

    article_id = sys.argv[1]
    content_file = sys.argv[2] if len(sys.argv) > 2 else None

    print(f"Обновление статьи ID: {article_id}")

    articletext = None
    if content_file:
        if not os.path.exists(content_file):
            print(f"Файл '{content_file}' не найден!")
            sys.exit(1)

        with open(content_file, 'r', encoding='utf-8') as f:
            articletext = f.read()
        print(f"Загружено содержимое из '{content_file}' ({len(articletext)} символов)")

    success = update_article(article_id, articletext)
    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()
