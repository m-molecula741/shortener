# gRPC поддержка для сервиса сокращения URL

В соответствии с техническим заданием добавлена возможность делать запросы к серверу по протоколу gRPC. Все хендлеры доступны по gRPC и функционируют идентично уже имеющимся HTTP хендлерам.

### Реализованные gRPC методы

1. **ShortenURL** - аналог `POST /` (сокращение URL в текстовом формате)
2. **ShortenURLJSON** - аналог `POST /api/shorten` (сокращение URL в JSON формате)
3. **ExpandURL** - аналог `GET /{shortID}` (получение оригинального URL)
4. **ShortenBatch** - аналог `POST /api/shorten/batch` (пакетное сокращение URL)
5. **GetUserURLs** - аналог `GET /api/user/urls` (получение URL пользователя)
6. **DeleteUserURLs** - аналог `DELETE /api/user/urls` (удаление URL пользователя)
7. **Ping** - аналог `GET /ping` (проверка работоспособности)
8. **GetStats** - аналог `GET /api/internal/stats` (получение статистики с проверкой доверенной подсети)

### Конфигурация gRPC сервера

Добавлены новые параметры конфигурации аналогично HTTP серверу:

#### Флаги командной строки:
- `-g string` - адрес gRPC сервера (по умолчанию "localhost:3200")
- `-grpc` - включить/выключить gRPC сервер (по умолчанию true)

#### Переменные окружения:
- `GRPC_ADDRESS` - адрес gRPC сервера
- `ENABLE_GRPC` - включить/выключить gRPC сервер

#### JSON конфигурация:
```json
{
  "grpc_address": "localhost:3200",
  "enable_grpc": true
}
```

### Архитектура

HTTP и gRPC хендлеры являются фасадами к общему коду с бизнес-логикой:

```
HTTP Controller ─┐
                 ├─► URLService (общая бизнес-логика)
gRPC Server ─────┘
```

### Запуск сервера

Сервер запускает HTTP и gRPC серверы параллельно:

```bash
# Запуск с параметрами по умолчанию
./shortener

# HTTP сервер: localhost:8080
# gRPC сервер: localhost:3200

# Кастомная конфигурация
./shortener -a "localhost:8081" -g "localhost:3201"
```

### Логи запуска

При запуске сервера в логах будет отображаться информация о запуске обоих серверов:

```
INF gRPC server will be started address=localhost:3200
INF Starting gRPC server address=localhost:3200  
INF Starting HTTP server address=localhost:8080
```

### Тестирование

Добавлены unit тесты для всех gRPC методов в `internal/app/grpc/server_test.go`.

Запуск тестов:
```bash
go test ./internal/app/grpc -v
```