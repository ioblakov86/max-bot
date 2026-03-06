<?php
/**
 * Скрипт для обновления статьи в Joomla через PHP
 * Загрузите этот файл на сервер в корень сайта и выполните через браузер:
 * https://ваш-сайт.ru/update_article.php?id=1025&token=YOUR_TOKEN
 *
 * Или через POST:
 * curl -X POST "https://ваш-сайт.ru/update_article.php?token=YOUR_TOKEN" 
 *      -d "id=1025" -d "articletext=<p>Новое содержимое</p>"
 *
 * После успешного обновления удалите файл с сервера!
 */

// Секретный токен (должен совпадать с тем, что вы передаете в запросе)
define('SECRET_TOKEN', 'dljhfDJLFHdsfsdamnjdszzcg234234BJEWWEF');

// Подключаем Joomla
define('_JEXEC', 1);
define('JPATH_BASE', __DIR__);
require_once JPATH_BASE . '/includes/defines.php';
require_once JPATH_BASE . '/includes/framework.php';

// Проверяем токен
$token = isset($_GET['token']) ? $_GET['token'] : (isset($_POST['token']) ? $_POST['token'] : '');
if ($token !== SECRET_TOKEN) {
    echo "Ошибка: неверный токен доступа\n";
    exit(1);
}

// Получаем ID статьи
$article_id = isset($_GET['id']) ? (int)$_GET['id'] : (isset($_POST['id']) ? (int)$_POST['id'] : 0);
if ($article_id <= 0) {
    echo "Ошибка: не указан ID статьи\n";
    exit(1);
}

// Получаем новое содержимое (из POST или GET)
// articletext будет сохранен в introtext
$new_articletext = isset($_POST['articletext']) ? $_POST['articletext'] : (isset($_GET['articletext']) ? $_GET['articletext'] : null);

if (version_compare(JVERSION, '4.0', 'lt')) {
    // Joomla 3.x
    $app = JFactory::getApplication('administrator');
    $db = JFactory::getDbo();

    // Получаем текущую статью
    $query = $db->getQuery(true)
        ->select($db->quoteName(['id', 'title', 'introtext', 'fulltext']))
        ->from($db->quoteName('#__content'))
        ->where($db->quoteName('id') . ' = ' . (int)$article_id);

    $db->setQuery($query);
    $article = $db->loadObject();

    if (!$article) {
        echo "Ошибка: статья с ID {$article_id} не найдена\n";
        exit(1);
    }

    echo "=== Статья до обновления ===\n";
    echo "ID: {$article->id}\n";
    echo "Заголовок: {$article->title}\n";
    echo "Длина introtext: " . strlen($article->introtext) . " символов\n";
    echo "Длина fulltext: " . strlen($article->fulltext) . " символов\n";
    echo "\n";

    // Если содержимое не передано, используем текущее + маркер
    if ($new_articletext === null) {
        $timestamp = date('Y-m-d H:i:s');
        $update_marker = "<!-- PHP_UPDATE_TEST_" . $timestamp . " -->";
        $new_articletext = $article->introtext . "\n\n{$update_marker}";
    }

    // Обновляем статью через базу данных
    $timestamp = date('Y-m-d H:i:s');
    $query = $db->getQuery(true)
        ->update($db->quoteName('#__content'))
        ->set($db->quoteName('introtext') . ' = ' . $db->quote($new_articletext))
        ->set($db->quoteName('modified') . ' = ' . $db->quote($timestamp))
        ->where($db->quoteName('id') . ' = ' . (int)$article_id);

    $db->setQuery($query);

    try {
        $db->execute();
        echo "✅ Статья успешно обновлена!\n";
        echo "Длина нового содержимого: " . strlen($new_articletext) . " символов\n";

        // Проверяем результат
        $query = $db->getQuery(true)
            ->select($db->quoteName('introtext'))
            ->from($db->quoteName('#__content'))
            ->where($db->quoteName('id') . ' = ' . (int)$article_id);

        $db->setQuery($query);
        $result = $db->loadResult();

        if ($result === $new_articletext) {
            echo "✅ Содержимое успешно записано в introtext!\n";
        } else {
            echo "❌ Содержимое НЕ совпадает!\n";
            echo "Ожидалось: " . strlen($new_articletext) . " символов\n";
            echo "Получено: " . strlen($result) . " символов\n";
        }

    } catch (Exception $e) {
        echo "Ошибка при обновлении: " . $e->getMessage() . "\n";
        exit(1);
    }

} else {
    // Joomla 4.x
    echo "Этот скрипт предназначен для Joomla 3.x\n";
    exit(1);
}

echo "\nПосле успешного обновления удалите файл update_article.php с сервера!\n";
