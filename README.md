# Family expenses

Сервис семейного учёта расходов с Telegram-ботом и веб-панелью.

## Демо

- Telegram-бот: [@FamilyPiganovBot](https://t.me/FamilyPiganovBot)
- Веб-панель: [p01--frontend--lrdkxkkclpsg.code.run](https://p01--frontend--lrdkxkkclpsg.code.run/)

Веб-панель предназначена для главы семьи и по условиям задачи не требует
авторизации. В ней доступны расходы всех членов семьи, а владельца записи глава
семьи выбирает при создании или редактировании расхода. Ограничение доступа
«только к своим расходам» действует в Telegram-боте и определяется Telegram ID
отправителя.

## Возможности

- добавление, просмотр, подсчёт, изменение и удаление расходов через Telegram;
- разбор естественного языка через function calling OpenAI-совместимого LLM API;
- автоматическое определение категории и даты расхода;
- пересчёт иностранных валют в рубли по официальному курсу Банка России;
- CRUD и фильтрация расходов всех членов семьи в веб-панели;
- REST API на Go;
- PostgreSQL и автоматические SQL-миграции;
- React + TypeScript frontend, который раздаётся через nginx;
- health checks, JSON-логирование, восстановление после panic и graceful shutdown.

## Быстрый запуск через Docker

Это рекомендуемый способ локального запуска: отдельно устанавливать Go, Node.js
и PostgreSQL не требуется.

### Требования

- Docker Engine или Docker Desktop;
- Docker Compose v2 (`docker compose`);
- Telegram-токен и ключ LLM-провайдера, если требуется работа бота.

Склонируйте проект и перейдите в его каталог:

```shell
git clone https://github.com/AndreyPiganov/test-task-expenses.git
cd test-task-expenses
```

Создайте локальный файл окружения:

```shell
cp .env.example .env
```

Заполните в `.env` как минимум `TELEGRAM_BOT_TOKEN`, `LLM_BASE_URL`,
`LLM_API_KEY` и `LLM_MODEL`, если хотите проверять Telegram-бота. Для запуска
только REST API и веб-панели эти переменные можно оставить пустыми.

Соберите и запустите все сервисы:

```shell
docker compose up --build
```

Для запуска в фоне:

```shell
docker compose up -d --build
```

После запуска доступны:

- веб-панель — `http://localhost:3000`;
- REST API — `http://localhost:8080`;
- liveness-проверка — `http://localhost:8080/health`;
- readiness-проверка — `http://localhost:8080/ready`.

Проверить backend можно командой:

```shell
curl http://localhost:8080/health
```

Ожидаемый ответ:

```json
{"status":"ok"}
```

Посмотреть логи:

```shell
docker compose logs -f backend
```

Остановить приложение:

```shell
docker compose down
```

Данные PostgreSQL сохраняются в Docker volume `postgres-data` и не удаляются
командой `docker compose down`.

После первого сообщения Telegram-боту пользователь автоматически появится в
списке членов семьи и станет доступен в веб-панели.

> Не запускайте локальный и production backend одновременно с одним и тем же
> `TELEGRAM_BOT_TOKEN`: два экземпляра long polling будут конкурировать за
> обновления. Для разработки лучше создать отдельного бота через
> [@BotFather](https://t.me/BotFather).

## Переменные окружения

Docker Compose автоматически читает `.env` из корня проекта. Файл `.env`
добавлен в `.gitignore`: не коммитьте токены и API-ключи. В репозитории должен
оставаться только безопасный шаблон `.env.example`.

| Переменная | По умолчанию | Назначение |
| --- | --- | --- |
| `DATABASE_URL` | `postgres://expenses:expenses@localhost:5432/expenses?sslmode=disable` | Подключение к PostgreSQL при запуске backend вне Docker Compose. В Compose backend получает внутренний адрес сервиса `postgres` автоматически. |
| `POSTGRES_PORT` | `5432` | Порт PostgreSQL, публикуемый на локальной машине. Можно изменить, если `5432` уже занят. |
| `TELEGRAM_BOT_TOKEN` | пусто | Токен Telegram-бота от `@BotFather`. Если пусто, бот отключён, а REST API продолжает работать. |
| `LLM_BASE_URL` | пусто | Базовый URL OpenAI-совместимого API без `/chat/completions`, например `https://api.openai.com/v1`. |
| `LLM_API_KEY` | пусто | Секретный API-ключ LLM-провайдера. Для локального API без авторизации может быть пустым. |
| `LLM_MODEL` | пусто | Идентификатор модели у выбранного провайдера. |
| `APP_TIMEZONE` | `Europe/Moscow` | IANA-часовой пояс для слов «сегодня», «вчера» и дат расходов. |
| `HTTP_ADDR` | `:8080` | Адрес HTTP-сервера при запуске backend вне Compose. В контейнере используется `:8080`. |

Пример `.env` для OpenAI:

```dotenv
DATABASE_URL=postgres://expenses:expenses@localhost:5432/expenses?sslmode=disable
POSTGRES_PORT=5432
TELEGRAM_BOT_TOKEN=123456789:replace-with-real-token
LLM_BASE_URL=https://api.openai.com/v1
LLM_API_KEY=replace-with-real-key
LLM_MODEL=gpt-4o-mini
APP_TIMEZONE=Europe/Moscow
HTTP_ADDR=:8080
```

## LLM API и поддерживаемые модели

Backend не использует SDK конкретного провайдера. Он отправляет запрос:

```text
POST {LLM_BASE_URL}/chat/completions
Authorization: Bearer {LLM_API_KEY}
```

Провайдер и модель должны поддерживать:

- OpenAI-совместимый Chat Completions API;
- сообщения с ролями `system` и `user`;
- параметр `temperature`;
- function tools в поле `tools`;
- строгие схемы функций (`strict: true`);
- `tool_choice: "required"` и `parallel_tool_calls: false`;
- ответ в `choices[0].message.tool_calls`.

Backend передаёт модели семь инструментов: создание, список, подсчёт, изменение
и удаление расходов, запрос уточнения и ответ на неподдерживаемую тему. Модель
выбирает ровно один инструмент и заполняет его аргументы. Go-код декодирует и
проверяет аргументы, самостоятельно подставляет Telegram ID, выполняет операцию
и формирует ответ. Модель никогда не получает возможность выполнять SQL или
вызывать Go-функции напрямую.

В отличие от прежнего JSON mode, схемы отдельных операций передаются через
стандартное поле `tools`, а не дублируются текстом внутри prompt. Для финансовой
операции используется один LLM-запрос: после выполнения tool call итоговое
сообщение формируется детерминированно в backend. Это уменьшает задержку и не
позволяет второму ответу модели исказить фактически сохранённые данные.

Для OpenAI с текущей реализацией подходят, например, `gpt-4o-mini` и
`gpt-4.1-mini`. Для классификации коротких сообщений обычно достаточно mini-
модели; более крупная модель может лучше разбирать неоднозначные фразы, но будет
дороже и медленнее. Названия, доступность и сроки поддержки моделей меняются,
поэтому актуальный идентификатор следует проверять в
[каталоге моделей OpenAI](https://developers.openai.com/api/docs/models).

Здесь OpenAI-совместимый означает формат HTTP API, а не производителя модели.
Однако совместимости только с `/chat/completions` уже недостаточно: конкретная
модель и обслуживающий её endpoint должны поддерживать перечисленные параметры
function calling. При выполнении этого контракта для переключения провайдера
достаточно заменить три переменные `LLM_*` в `.env` и перезапустить backend.

### DeepSeek напрямую

Создайте API-ключ в DeepSeek и укажите:

```dotenv
LLM_BASE_URL=https://api.deepseek.com
LLM_API_KEY=replace-with-deepseek-key
LLM_MODEL=deepseek-flash
```

Эта конфигурация сработает только с моделью DeepSeek, которая в момент запуска
поддерживает function tools и строгую JSON Schema. Актуальные имена моделей и их
возможности следует проверять в
[документации DeepSeek](https://api-docs.deepseek.com/quick_start/pricing/).

### Модели разных провайдеров через OpenRouter

OpenRouter удобен, если хочется менять модели, не создавая отдельную интеграцию
для каждого поставщика. Например, конфигурация модели Google выглядит так:

```dotenv
LLM_BASE_URL=https://openrouter.ai/api/v1
LLM_API_KEY=replace-with-openrouter-key
LLM_MODEL=google/gemini-2.5-pro
```

Идентификатор имеет формат `поставщик/модель`. Выберите актуальную модель в
[каталоге OpenRouter](https://openrouter.ai/models) и скопируйте её идентификатор
в `LLM_MODEL`. Проверьте наличие параметров `tools`, `tool_choice`,
`parallel_tool_calls` и structured outputs. В OpenRouter поддержка параметров
может зависеть не только от модели, но и от обслуживающего её endpoint.

### Локальная модель через Ollama

Если backend запускается вне Docker командой `make run`, можно использовать
установленную локально Ollama:

```shell
ollama pull llama3.1:8b
```

```dotenv
LLM_BASE_URL=http://localhost:11434/v1
LLM_API_KEY=
LLM_MODEL=llama3.1:8b
```

Ollama предоставляет OpenAI-совместимый `/v1/chat/completions`, но function
calling поддерживают не все локальные модели и версии API. Перед использованием
проверьте, что выбранная модель принимает строгие function tools. Качество
распознавания также зависит от модели и доступных CPU, GPU и памяти. Этот пример
рассчитан на запуск backend на хосте; из Docker-контейнера `localhost` указывает
на сам контейнер, поэтому для Ollama на хосте понадобится отдельная настройка
сети.

Нативный Anthropic Messages API и другие API с отличающимся форматом напрямую
не подойдут. Их можно подключить через OpenAI-совместимый шлюз или после
реализации отдельного клиента в backend.

После изменения `.env` пересоздайте backend:

```shell
docker compose up -d --build backend
docker compose logs -f backend
```

Затем отправьте тестовому боту сообщение `кофе 250`. Ошибка `401` или `403`
обычно означает неверный API-ключ, `404` — неверный `LLM_BASE_URL`, а `400` с
упоминанием `tools`, `strict`, `tool_choice`, `parallel_tool_calls` или
`temperature` — несовместимость выбранной модели с текущим клиентом.

Схема ограничивает форму вызова, но backend всё равно считает аргументы
недоверенными и проверяет названия инструментов, поля, категории, даты, суммы и
принадлежность расхода пользователю. Механика function calling описана в
[официальной документации OpenAI](https://developers.openai.com/api/docs/guides/function-calling).

Если `LLM_BASE_URL` или `LLM_MODEL` пустые, обработчик естественного языка
отключается. REST API и веб-панель при этом продолжают работать.

## Запуск для разработки без пересборки контейнеров

Для этого варианта потребуются:

- Go 1.26 или новее;
- Node.js 22 или новее и npm;
- Docker для локального PostgreSQL либо отдельно установленный PostgreSQL.

Запустите только базу данных:

```shell
docker compose up -d postgres
```

Go-приложение само не загружает `.env`. В Bash или WSL экспортируйте переменные
перед запуском backend:

```shell
set -a
source .env
set +a
make run
```

В другом терминале запустите frontend в dev-режиме:

```shell
cd web
npm ci
npm run dev
```

Vite откроет frontend на `http://localhost:5173` и будет проксировать `/api` в
backend на `http://localhost:8080`.

Правила и карта репозитория для coding agents находятся в
[`AGENTS.md`](./AGENTS.md).

## Telegram-бот

Бот работает через long polling: backend держит запрос `getUpdates`, получает
новые сообщения и подтверждает обработанные обновления через `offset`. При
первом сообщении пользователь добавляется или обновляется в `family_members`.

Поддерживаемые сценарии:

- «кофе 350» — добавить расход;
- «купил наушники за 100 долларов» — пересчитать сумму в рубли по официальному
  курсу Банка России на дату расхода и сохранить рублёвую сумму;
- «покажи траты за неделю» — показать список;
- «сколько потратил на еду в этом месяце» — посчитать сумму;
- «последнее было не 350, а 450» — изменить последнюю запись;
- «удали такси» — найти и удалить однозначно определённую запись.

LLM выбирает function tool и заполняет его JSON-аргументы. Backend самостоятельно
подставляет Telegram ID, проверяет сумму, категорию и дату и только затем
выполняет действие. Операции бота всегда ограничены Telegram ID отправителя
непосредственно в SQL.

Если под запрос подходят несколько расходов, бот ничего не изменяет и просит
уточнить название, дату или явно сказать «последний». Удаление также требует
явного намерения в исходном сообщении пользователя.

Для суммы в иностранной валюте LLM только распознаёт исходную сумму и код
валюты. Пересчёт выполняет backend по данным официального XML API Банка России,
а результат округляется до двух знаков. Если валюта не указана, сумма считается
рублёвой.

## REST API

Доступные методы:

- `GET /health` — liveness-проверка;
- `GET /ready` — readiness-проверка PostgreSQL;
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

Член семьи с указанным `user_id` должен уже существовать. Его запись создаётся
или обновляется автоматически при сообщении Telegram-боту.

Веб-панель обращается к `/api` через nginx. В Docker запросы проксируются во
внутренний сервис `backend`, поэтому отдельная настройка CORS не требуется.

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
