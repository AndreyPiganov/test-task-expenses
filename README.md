# Family expenses

Сервис семейного учёта расходов с Telegram-ботом и веб-панелью.

Веб-панель предназначена для главы семьи и по условиям задачи не требует
авторизации. В ней доступны расходы всех членов семьи, а владельца записи глава
семьи выбирает при создании или редактировании расхода. Ограничение доступа
«только к своим расходам» действует в Telegram-боте и определяется Telegram ID
отправителя.

В проект входят Go backend, Telegram-бот и React-веб-панель:

- `GET /health` — liveness-проверка;
- `GET /ready` — readiness-проверка;
- JSON-логирование запросов;
- восстановление после panic в HTTP-обработчике;
- graceful shutdown по `SIGINT` и `SIGTERM`;
- подключение к PostgreSQL и автоматические SQL-миграции;
- доменная модель расхода `Expense`;
- просмотр, фильтрация, создание, редактирование и удаление расходов в браузере.

## Запуск локально

Требуется Go 1.26 или новее.

```shell
docker compose up -d postgres
make run
```

Сервер слушает `http://localhost:8080`. Адрес и часовой пояс можно изменить:

```shell
HTTP_ADDR=:9090 APP_TIMEZONE=UTC make run
```

## Запуск в Docker

```shell
docker compose up --build
```

Compose поднимает PostgreSQL, дожидается его готовности, применяет миграции,
запускает backend и собирает веб-панель. Данные базы хранятся в volume
`postgres-data`.

После запуска доступны:

- веб-панель — `http://localhost:3000`;
- REST API — `http://localhost:8080`;
- проверка backend — `http://localhost:8080/health`.

Веб-панель обращается к `/api` через nginx. В Docker запросы проксируются во
внутренний сервис `backend`, поэтому отдельная настройка CORS не нужна.

Проверка:

```shell
curl http://localhost:8080/health
```

Ожидаемый ответ:

```json
{"status":"ok"}
```

## REST API

Доступные методы:

- `GET /api/v1/members` — члены семьи;
- `GET /api/v1/expenses` — расходы с фильтрами;
- `POST /api/v1/expenses` — создание расхода;
- `GET /api/v1/expenses/{id}` — получение расхода;
- `PATCH /api/v1/expenses/{id}` — частичное изменение;
- `DELETE /api/v1/expenses/{id}` — удаление.

Фильтры списка: `user_id`, `category`, `date_from`, `date_to`, `limit`, `offset`.
Дата передаётся как `YYYY-MM-DD`, денежная сумма — десятичной строкой.

Пример создания:

```shell
curl -X POST http://localhost:8080/api/v1/expenses \
  -H 'Content-Type: application/json' \
  -d '{
    "user_id": 123456789,
    "amount": "350.00",
    "category": "еда",
    "spent_at": "2026-10-16",
    "comment": "кофе"
  }'
```

Член семьи с указанным `user_id` должен уже существовать. После подключения
Telegram-бота его запись будет создаваться автоматически при первом сообщении.

## Telegram-бот

Укажите токен, полученный для бота в Telegram:

```dotenv
TELEGRAM_BOT_TOKEN=123456:replace-with-real-token
```

После этого запустите приложение:

```shell
docker compose up --build
```

Бот работает через long polling: backend держит запрос `getUpdates`, получает
новые сообщения и подтверждает обработанные обновления через `offset`. При
первом сообщении пользователь автоматически добавляется в `family_members`.

Если `TELEGRAM_BOT_TOKEN` пустой, Telegram-часть отключается, но REST API
продолжает работать.

### Подключение LLM

Для разбора сообщений настройте OpenAI-совместимый Chat Completions API:

```dotenv
LLM_BASE_URL=https://provider.example/v1
LLM_API_KEY=replace-with-real-key
LLM_MODEL=provider-model-name
```

LLM преобразует сообщение в JSON-команду. Backend самостоятельно подставляет
Telegram ID, проверяет сумму, категорию и дату и только затем сохраняет расход.

Поддерживаемые сценарии:

- «кофе 350» — добавить расход;
- «покажи траты за неделю» — показать список;
- «сколько потратил на еду в этом месяце» — посчитать сумму;
- «последнее было не 350, а 450» — изменить последнюю запись;
- «удали такси» — найти и удалить однозначно определённую запись.

Операции бота всегда ограничены Telegram ID отправителя непосредственно в SQL.
Если под запрос подходят несколько расходов, бот ничего не изменяет и просит
уточнить название, дату или явно сказать «последний».

## Проверки

```shell
make test
make vet
make fmt
```

Интеграционный тест миграций запускается при наличии отдельной тестовой базы:

```shell
cd backend
TEST_DATABASE_URL=postgres://expenses:expenses@localhost:5432/expenses_test?sslmode=disable \
  go test ./internal/database -run TestOpenAndMigrate
```
