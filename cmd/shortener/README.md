# API Сервиса сокращения URL

## Эндпоинты

### 1. Сокращение URL (текстовый формат)
```
POST /
Content-Type: text/plain

Тело запроса: URL для сокращения
Пример: http://example.com

Ответ (201 Created):
Content-Type: text/plain
http://localhost:8080/abcd1234
```

### 2. Сокращение URL (JSON формат)
```
POST /api/shorten
Content-Type: application/json

Тело запроса:
{
    "url": "http://example.com"
}

Ответ (201 Created):
Content-Type: application/json
{
    "result": "http://localhost:8080/abcd1234"
}
```

### 3. Пакетное сокращение URL
```
POST /api/shorten/batch
Content-Type: application/json

Тело запроса:
[
    {
        "correlation_id": "1",
        "original_url": "http://example.com"
    },
    {
        "correlation_id": "2",
        "original_url": "http://another.com"
    }
]

Ответ (201 Created):
Content-Type: application/json
[
    {
        "correlation_id": "1",
        "short_url": "http://localhost:8080/abcd1234"
    },
    {
        "correlation_id": "2",
        "short_url": "http://localhost:8080/efgh5678"
    }
]
```

### 4. Получение оригинального URL
```
GET /{shortID}
Пример: GET /abcd1234

Ответ (307 Temporary Redirect):
Location: http://example.com
```

### 5. Получение всех URL пользователя
```
GET /api/user/urls
Cookie: user_id=<encrypted_user_id>

Ответ (200 OK):
Content-Type: application/json
[
    {
        "short_url": "http://localhost:8080/abcd1234",
        "original_url": "http://example.com"
    }
]

Ответ при отсутствии URL (204 No Content)
```

### 6. Удаление URL пользователя
```
DELETE /api/user/urls
Cookie: user_id=<encrypted_user_id>
Content-Type: application/json

Тело запроса:
["abcd1234", "efgh5678"]

Ответ (202 Accepted)
```

### 7. Проверка работоспособности
```
GET /ping

Ответ (200 OK) - если БД доступна
Ответ (500 Internal Server Error) - если БД недоступна
```

## Авторизация

Все запросы (кроме первого запроса нового пользователя) должны содержать куку `user_id`. 
Кука устанавливается автоматически при первом запросе пользователя.

## Коды ответов

- 200 OK - успешный запрос
- 201 Created - URL успешно создан
- 202 Accepted - запрос на удаление принят
- 204 No Content - нет данных для ответа
- 307 Temporary Redirect - редирект на оригинальный URL
- 400 Bad Request - неверный запрос
- 401 Unauthorized - отсутствует или неверная кука авторизации
- 404 Not Found - URL не найден
- 409 Conflict - URL уже существует
- 410 Gone - URL был удален
- 500 Internal Server Error - внутренняя ошибка сервера