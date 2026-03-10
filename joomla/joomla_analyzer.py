#!/usr/bin/env python3
"""
Joomla Analyzer - скрипт для анализа и обновления статей Joomla

Использование:
    # Анализ изменений (возвращает JSON с планируемыми изменениями)
    python3 joomla_analyzer.py analyze --json '<JSON от бота>'

    # Применение изменений
    python3 joomla_analyzer.py apply --json '<JSON от бота>'

JSON формат:
{
    "Employee": {"Position": "...", "FullName": "Фамилия ИО"},
    "AbsenceType": "Больничный" | "Отпуск",
    "Dates": {"StartDate": "...", "EndDate": "..."},
    "Status": "Начало" | "Окончание" | "Продолжение"
}
"""

import requests
from bs4 import BeautifulSoup
import json
import sys
import os
import re
import argparse
from typing import List, Dict, Any, Optional, Tuple

# Загружаем config если есть
try:
    import config
except ImportError:
    # Если config.py нет, используем переменные окружения
    class EnvConfig:
        SITE_URL = os.getenv('JOOMLA_SITE_URL', 'https://plk32.ru')
        ADMIN_URL = os.getenv('JOOMLA_ADMIN_URL', 'https://plk32.ru/administrator')
        USERNAME = os.getenv('JOOMLA_USERNAME', '')
        PASSWORD = os.getenv('JOOMLA_PASSWORD', '')
        API_TOKEN = os.getenv('JOOMLA_API_TOKEN', '')
        ARTICLE_IDS = [int(x.strip()) for x in os.getenv('JOOMLA_ARTICLE_IDS', '1025,1027,1028,1029').split(',')]
        REQUEST_TIMEOUT = 30
        VERIFY_SSL = True
        USER_AGENT = 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36'
    
    config = EnvConfig()


def login(session) -> Optional[str]:
    """Аутентификация в Joomla админке"""
    login_url = f"{config.ADMIN_URL}/index.php"
    
    try:
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
    except Exception as e:
        print(f"Ошибка аутентификации: {e}", file=sys.stderr)
        return None


def get_article_content(session, article_id: int) -> Optional[str]:
    """Получение содержимого статьи через форму редактирования"""
    try:
        edit_url = f"{config.ADMIN_URL}/index.php?option=com_content&task=article.edit&id={article_id}"
        response = session.get(edit_url, timeout=config.REQUEST_TIMEOUT, verify=config.VERIFY_SSL)
        soup = BeautifulSoup(response.text, 'html.parser')

        # Ищем содержимое в textarea jform_articletext
        article_textarea = soup.find('textarea', {'id': 'jform_articletext'})

        if article_textarea and article_textarea.text:
            return article_textarea.text.strip()
        return None
    except Exception as e:
        print(f"Ошибка получения статьи {article_id}: {e}", file=sys.stderr)
        return None


def update_article(session, article_id: int, content: str, csrf_token: str) -> bool:
    """Обновление содержимого статьи"""
    try:
        # Получаем форму редактирования для получения всех необходимых полей
        edit_url = f"{config.ADMIN_URL}/index.php?option=com_content&task=article.edit&id={article_id}"
        response = session.get(edit_url, timeout=config.REQUEST_TIMEOUT, verify=config.VERIFY_SSL)
        soup = BeautifulSoup(response.text, 'html.parser')

        # Собираем данные формы
        form_data = {}
        
        # Находим все input поля
        for input_tag in soup.find_all('input', {'type': ['hidden', 'number']}):
            name = input_tag.get('name')
            value = input_tag.get('value', '')
            if name:
                form_data[name] = value

        # Находим CSRF токен в форме
        if not csrf_token:
            for input_tag in soup.find_all('input', {'name': lambda x: x and len(x) == 32 and x.isalnum()}):
                csrf_token = input_tag['name']
                form_data[csrf_token] = '1'
                break

        # Устанавливаем новое содержимое
        form_data['jform[articletext]'] = content
        form_data['task'] = 'article.save'

        # Отправляем форму
        save_url = f"{config.ADMIN_URL}/index.php?option=com_content&task=article.save"
        response = session.post(save_url, data=form_data, timeout=config.REQUEST_TIMEOUT, verify=config.VERIFY_SSL)

        return "Статья успешно сохранена" in response.text or "article.edit" not in response.text
    except Exception as e:
        print(f"Ошибка обновления статьи {article_id}: {e}", file=sys.stderr)
        return False


def extract_name_from_td(td_tag) -> Tuple[str, str, str]:
    """
    Извлекает фамилию, имя, отчество из <td> тега
    Возвращает (фамилия, полное_имя, оригинальный_текст)
    """
    text = td_tag.get_text(strip=True)
    
    # Разбиваем по <br> - фамилия обычно первая
    parts = []
    for content in td_tag.children:
        if hasattr(content, 'name') and content.name == 'br':
            parts.append('')
        elif hasattr(content, 'strip'):
            stripped = content.strip()
            if stripped:
                parts.append(stripped)
    
    if not parts:
        parts = text.replace('<br>', '\n').split('\n')
    
    # Фильтруем пустые и убираем текст из span (пометки о больничном)
    clean_parts = []
    for part in parts:
        # Пропускаем текст внутри span с пометками
        if 'На больничном' in part or 'В отпуске' in part:
            continue
        part = part.strip()
        if part:
            clean_parts.append(part)
    
    if len(clean_parts) >= 1:
        surname = clean_parts[0].strip()
        full_name = ' '.join(clean_parts[:3]) if len(clean_parts) >= 3 else ' '.join(clean_parts)
        return surname, full_name, text
    
    return '', '', text


def find_doctor_in_html(soup: BeautifulSoup, full_name: str) -> List[Tuple[Any, str]]:
    """
    Ищет врача в HTML по фамилии и инициалам
    Возвращает список (td_element, article_id)
    """
    results = []
    
    # Извлекаем фамилию (первое слово до пробела)
    name_parts = full_name.strip().split()
    if not name_parts:
        return results
    
    surname = name_parts[0].lower()
    
    # Ищем во всех <td> тегах
    for td in soup.find_all('td'):
        td_text = td.get_text(' ', strip=True).lower()
        
        # Проверяем наличие фамилии
        if surname not in td_text:
            continue
        
        # Проверяем, что это действительно врач (есть <br> после фамилии)
        br_count = len(td.find_all('br'))
        if br_count < 1:
            continue
        
        results.append(td)
    
    return results


def normalize_initials(initials: str) -> List[str]:
    """
    Генерирует различные форматы инициалов для поиска
    """
    variants = []
    
    # Убираем точки и пробелы
    clean = initials.replace('.', '').replace(' ', '')
    
    if len(clean) >= 2:
        # Формат "ИО"
        variants.append(clean[:2].upper())
        # Формат "И.О."
        variants.append(f"{clean[0]}.{clean[1]}.")
        # Формат "И. О."
        variants.append(f"{clean[0]}. {clean[1]}.")
    
    return variants


def create_absence_span(absence_type: str, dates: Dict[str, str]) -> str:
    """
    Создаёт HTML для пометки о больничном/отпуске
    """
    if absence_type.lower() == 'больничный':
        text = "На больничном"
    elif absence_type.lower() == 'отпуск':
        text = "В отпуске"
        if dates.get('StartDate') and dates.get('EndDate'):
            text = f"В отпуске с {dates['StartDate']} по {dates['EndDate']}"
        elif dates.get('StartDate'):
            text = f"В отпуске с {dates['StartDate']}"
        elif dates.get('EndDate'):
            text = f"В отпуске до {dates['EndDate']}"
    else:
        text = f"{absence_type}"
    
    return f'<br /><span style="font-weight: bold; font-style: italic; color: red;">{text}</span>'


def add_absence_to_td(td_element, absence_span: str) -> str:
    """
    Добавляет пометку о больничном/отпуске в <td> элемент
    """
    # Проверяем, есть ли уже такая пометка
    td_html = str(td_element)
    
    # Ищем существующие span с пометками
    existing_span = td_element.find('span', style=lambda x: x and 'color: red' in x if x else False)
    
    if existing_span:
        # Уже есть пометка - не добавляем дубль
        return None
    
    # Вставляем перед закрывающим </td>
    # Находим позицию для вставки - после последнего <br> или перед </td>
    td_content = str(td_element)
    
    # Вставляем перед </td>
    modified = td_content.replace('</td>', f'{absence_span}\n</td>', 1)
    
    return modified


def remove_absence_from_td(td_element) -> str:
    """
    Убирает или комментирует пометку о больничном/отпуске из <td>
    """
    td_html = str(td_element)
    
    # Находим span с пометкой
    span = td_element.find('span', style=lambda x: x and 'color: red' in x if x else False)
    
    if not span:
        return None  # Нет пометки для удаления
    
    # Комментируем span вместо удаления
    span_html = str(span)
    commented = f'<!-- {span_html} -->'
    
    modified = td_html.replace(span_html, commented, 1)
    
    return modified


def analyze_changes(json_data: Dict[str, Any], articles: Dict[int, str]) -> Dict[str, Any]:
    """
    Анализирует изменения и возвращает план действий
    """
    result = {
        'success': True,
        'employee': json_data.get('Employee', {}),
        'changes': [],
        'errors': []
    }
    
    employee = json_data.get('Employee', {})
    full_name = employee.get('FullName', '')
    absence_type = json_data.get('AbsenceType', '')
    dates = json_data.get('Dates', {})
    status = json_data.get('Status', '')
    
    # Отладочное логирование
    print(f"DEBUG: Received JSON - full_name='{full_name}', status='{status}'", file=sys.stderr)
    print(f"DEBUG: Articles loaded: {list(articles.keys())}", file=sys.stderr)
    
    if not full_name:
        result['success'] = False
        result['errors'].append('Не указано ФИО сотрудника')
        return result
    
    if status == 'Продолжение':
        result['success'] = True
        result['message'] = 'Статус "Продолжение" - изменения не требуются'
        return result
    
    for article_id, content in articles.items():
        soup = BeautifulSoup(content, 'html.parser')
        found_doctors = find_doctor_in_html(soup, full_name)
        
        if not found_doctors:
            result['errors'].append(f'Врач {full_name} не найден в статье {article_id}')
            continue
        
        for td in found_doctors:
            change = {
                'article_id': article_id,
                'doctor': td.get_text(' ', strip=True)[:50],
                'action': None,
                'old_html': str(td),
                'new_html': None
            }
            
            if status == 'Начало':
                absence_span = create_absence_span(absence_type, dates)
                new_html = add_absence_to_td(td, absence_span)
                if new_html:
                    change['action'] = 'add'
                    change['new_html'] = new_html
                    result['changes'].append(change)
            
            elif status == 'Окончание':
                new_html = remove_absence_from_td(td)
                if new_html:
                    change['action'] = 'remove'
                    change['new_html'] = new_html
                    result['changes'].append(change)
    
    if not result['changes'] and not result['errors']:
        result['errors'].append('Нет изменений для применения')
    
    return result


def apply_changes(json_data: Dict[str, Any], changes: List[Dict]) -> Dict[str, Any]:
    """
    Применяет изменения к статьям
    """
    result = {
        'success': True,
        'updated_articles': [],
        'errors': []
    }
    
    session = requests.Session()
    session.headers.update({'User-Agent': config.USER_AGENT})
    
    # Логин
    csrf_token = login(session)
    if not csrf_token:
        result['success'] = False
        result['errors'].append('Не удалось аутентифицироваться')
        return result
    
    # Группируем изменения по статьям
    articles_changes = {}
    for change in changes:
        article_id = change['article_id']
        if article_id not in articles_changes:
            articles_changes[article_id] = []
        articles_changes[article_id].append(change)
    
    # Применяем изменения к каждой статье
    for article_id, article_changes in articles_changes.items():
        # Получаем текущее содержимое
        content = get_article_content(session, article_id)
        if not content:
            result['errors'].append(f'Не удалось получить статью {article_id}')
            continue
        
        # Применяем все изменения к содержимому
        for change in article_changes:
            old_html = change['old_html']
            new_html = change['new_html']
            if new_html:
                content = content.replace(old_html, new_html, 1)
        
        # Сохраняем статью
        if update_article(session, article_id, content, csrf_token):
            result['updated_articles'].append(article_id)
        else:
            result['errors'].append(f'Не удалось обновить статью {article_id}')
    
    return result


def main():
    parser = argparse.ArgumentParser(description='Joomla Analyzer')
    parser.add_argument('command', choices=['analyze', 'apply'], help='Команда: analyze или apply')
    parser.add_argument('--json', required=True, help='JSON данные от бота')
    parser.add_argument('--changes', help='JSON с изменениями (для apply)')

    args = parser.parse_args()

    try:
        json_data = json.loads(args.json)
    except json.JSONDecodeError as e:
        print(json.dumps({'success': False, 'errors': [f'Ошибка парсинга JSON: {e}']}))
        sys.exit(1)

    # Отладка: выводим полученный JSON
    print(f"DEBUG: Parsed JSON - full_name='{json_data.get('Employee', {}).get('FullName', '')}'", file=sys.stderr)

    if args.command == 'analyze':
        # Получаем содержимое всех статей
        articles = {}
        session = requests.Session()
        session.headers.update({'User-Agent': config.USER_AGENT})

        print(f"DEBUG: Starting Joomla login to {config.ADMIN_URL}", file=sys.stderr)
        csrf_token = login(session)
        if not csrf_token:
            print(f"DEBUG: Login failed", file=sys.stderr)
            print(json.dumps({'success': False, 'errors': ['Не удалось аутентифицироваться']}))
            sys.exit(1)
        
        print(f"DEBUG: Login successful, CSRF token: {csrf_token[:10]}...", file=sys.stderr)

        article_ids = getattr(config, 'ARTICLE_IDS', [1025, 1027, 1028, 1029])
        print(f"DEBUG: Loading articles: {article_ids}", file=sys.stderr)
        
        for article_id in article_ids:
            content = get_article_content(session, article_id)
            if content:
                articles[article_id] = content
                print(f"DEBUG: Loaded article {article_id} ({len(content)} chars)", file=sys.stderr)
            else:
                print(f"DEBUG: Failed to load article {article_id}", file=sys.stderr)

        result = analyze_changes(json_data, articles)
        print(json.dumps(result, ensure_ascii=False, indent=2))
    
    elif args.command == 'apply':
        if not args.changes:
            print(json.dumps({'success': False, 'errors': ['Требуется параметр --changes']}))
            sys.exit(1)
        
        try:
            changes = json.loads(args.changes)
        except json.JSONDecodeError as e:
            print(json.dumps({'success': False, 'errors': [f'Ошибка парсинга JSON изменений: {e}']}))
            sys.exit(1)
        
        result = apply_changes(json_data, changes)
        print(json.dumps(result, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
