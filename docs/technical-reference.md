> Технический справочник. Краткое представление и запуск: [README](../README.md). Конкретный результат сборки и тестов проверяйте в Actions.

# IS Budget

Полноценный прототип для управления проектными финансами:
- учет доходов по дням,
- учет фактических и плановых расходов,
- многоуровневые налоги и комиссии,
- многоуровневые ручные правила распределения прибыли,
- прогнозирование дохода через отдельный analytics-сервис,
- командная работа с RBAC.

В текущем коде присутствуют finance pipeline и workflow graph. Финансовые операции и управление зависимостями задач разделены.

## Стек

- `frontend`: Next.js 14, React, TypeScript
- `backend`: Go, chi, pgx, sqlc, golang-migrate, JWT, RBAC
- `analytics`: FastAPI, pandas, numpy, statsmodels
- `db`: PostgreSQL 16
- `infra`: Docker Compose

## Архитектура системы

Система состоит из четырёх прикладных частей:
- `frontend` - пользовательский интерфейс на Next.js;
- `backend` - основной API и orchestration layer;
- `analytics` - отдельный FastAPI-сервис прогнозирования;
- `db` - PostgreSQL для операционных и прогнозных данных.

Роли компонентов:
- `frontend` работает только с backend API;
- `backend` отвечает за аутентификацию, RBAC, бизнес-операции, расчёт финансовых агрегатов, orchestration forecast flow и экспорт PDF;
- `analytics` принимает формализованный request на прогноз и возвращает сценарии `p10 / yhat / p90`;
- `db` хранит проектные, финансовые, forecast и user-related данные.

Контур взаимодействия:
1. пользователь работает через `frontend`;
2. `frontend` обращается только к `backend`;
3. `backend` при необходимости вызывает `analytics` как отдельный forecasting service;
4. результаты операций и прогнозов сохраняются в `PostgreSQL`.

## Сервисы и порты

- Frontend: `http://localhost:3000`
- Backend API: `http://localhost:8080/api`
- Backend API docs: `http://localhost:8080/swagger`
- Backend OpenAPI: `http://localhost:8080/openapi.yaml`
- Analytics API: `http://localhost:8000`
- Analytics docs: `http://localhost:8000/docs`
- Postgres: `localhost:5432`

## Запуск через Docker Compose

```bash
docker compose up -d --build
```

Проверка статуса:

```bash
docker compose ps
```

Остановка:

```bash
docker compose down
```

## Конфигурация backend

Backend читает настройки из переменных окружения.

Ключевые моменты по безопасности и конфигурации:
- `APP_ENV=production` отключает небезопасный development fallback для `JWT_SECRET`; в production секрет должен быть задан явно.
- `JWT_TTL` управляет временем жизни access token. По умолчанию используется `24h`.
- `FRONTEND_ORIGINS` можно передавать как список origins через запятую. Если он не задан, используется `FRONTEND_ORIGIN`. В development по умолчанию разрешён `http://localhost:3000`.
- `ANALYTICS_URL` и `FX_API_BASE_URL` валидируются как HTTP(S) URL при старте приложения.
- Для чувствительных auth/invite операций доступны env-параметры мягкого rate limiting:
  - `AUTH_RATE_LIMIT_WINDOW`, `AUTH_RATE_LIMIT_MAX`
  - `ACCOUNT_MUTATION_RATE_LIMIT_WINDOW`, `ACCOUNT_MUTATION_RATE_LIMIT_MAX`
  - `INVITE_ACTION_RATE_LIMIT_WINDOW`, `INVITE_ACTION_RATE_LIMIT_MAX`

Пример:

```env
APP_ENV=development
JWT_SECRET=dev-secret-change-me
JWT_TTL=24h
FRONTEND_ORIGINS=http://localhost:3000,http://127.0.0.1:3000
ANALYTICS_URL=http://localhost:8000
AUTH_RATE_LIMIT_WINDOW=1m
AUTH_RATE_LIMIT_MAX=10
```

Дополнительные ключевые env-переменные:
- `DATABASE_URL` - строка подключения к PostgreSQL;
- `PORT` - порт backend API;
- `MIGRATIONS_PATH` - путь к SQL-миграциям;
- `PAYOUT_SNAPSHOTS_DIR` - каталог PDF-снимков выплат;
- `AVATARS_DIR` - каталог аватаров;
- `FX_API_BASE_URL` - base URL FX provider;
- `FX_UPDATE_INTERVAL` - период обновления курсов валют.

## Security Notes

Реализованные прикладные меры защиты:
- JWT access token подписывается `HS256` и при разборе проверяется ожидаемый signing algorithm.
- В production запрещён insecure fallback для `JWT_SECRET`.
- Для чувствительных маршрутов включён мягкий in-memory rate limiting:
  - `POST /api/auth/register`
  - `POST /api/auth/login`
  - `PUT /api/auth/email`
  - `PUT /api/auth/password`
  - `POST /api/project-invites/{token}/accept`
  - `POST /api/project-invites/{token}/decline`
- Upload аватаров ограничен по размеру и проверяется по расширению и фактическому содержимому файла.
- CORS разрешает только явно сконфигурированные origins без wildcard.
- Внешние HTTP-интеграции (`analytics`, `FX API`) используют таймауты и не раскрывают технические детали клиенту API.
- Error responses backend стандартизированы и не должны возвращать stack traces, внутренние URL или чувствительные поля.

Текущие ограничения:
- refresh token flow пока не реализован: используется только access token с ограниченным `JWT_TTL`;
- rate limiting хранится в памяти процесса и подходит для учебного/одиночного backend deployment, но не для горизонтально масштабируемого production без общего storage.
- access token хранится во frontend-клиенте и не переведён на cookie-based session model;
- file upload hardening ограничен аватарами и не является универсальным файловым шлюзом для произвольных вложений.

## Основная бизнес-логика

1. В БД доход хранится по дням как daily delta series (`actual_income_daily`), поддерживается upsert по дате.
2. Основной экран доходов в UI принимает накопленный доход по дням кампании, а backend пересчитывает его в дневные дельты.
3. Forecast и analytics работают на дневной истории; накопительные графики строятся как производное представление поверх daily series.
4. Расходы:
- плановые (`planned_expenses`) для обязательств и приоритетов;
- фактические (`actual_expenses`) для реальных выплат.
5. Налоги применяются последовательно по уровням:
- уровень 1 считается от доступной базы,
- уровень 2 считается от остатка после уровня 1,
- и так далее.
6. После налогов доступная сумма распределяется по правилам `profit_share_rules`.
7. Если часть прибыли не распределена правилами, остаток в preview относится администратору проекта (PM).

## Статусы проекта

- `preparation`
- `active`
- `completed`

Переход `preparation -> active` выполняется вручную.
Переход `active -> completed` выполняется автоматически по длительности проекта.

## Роли и права (RBAC)

- `PM`: полный доступ (проект, участники, финансы, прогноз)
- `ANALYST`: финансы + прогноз, без управления участниками
- `TEAM`: просмотр проекта/финансов/дашборда/прогноза
- `CUSTOMER`: просмотр + прогноз

Ограничение: в проекте может быть только один `PM` (создатель проекта).

## Ключевые API эндпоинты

Все приватные эндпоинты идут под `/api` и требуют `Authorization: Bearer <token>`.

## Формат ошибок API

Базовая совместимость со старым frontend сохранена: поле `error` остаётся основным.

Текущий формат ошибки:

```json
{
  "error": "nickname failed on min",
  "code": "validation_error",
  "details": {
    "nickname": "min"
  }
}
```

Пояснения:
- `error` - человекочитаемое сообщение, на которое уже ориентируется frontend;
- `code` - стабильный машинный код категории ошибки;
- `details` - дополнительные сведения по полям для ошибок валидации.

Для некорректного JSON backend также возвращает `code: "invalid_json"`.

Дополнительные типовые ветки ошибок:
- `rate_limited` - превышен лимит запросов на чувствительном маршруте;
- `forecast_service_timeout` - analytics service не ответил вовремя;
- `forecast_service_unavailable` - analytics service недоступен или вернул bad upstream status;
- `forecast_service_invalid_response` - analytics service вернул неконсистентный ответ.

### Auth

- `POST /api/auth/register`
- `POST /api/auth/login`
- `GET /api/auth/me`
- `PUT /api/auth/preferences`
- `PUT /api/auth/profile`
- `PUT /api/auth/email`
- `PUT /api/auth/password`
- `POST /api/auth/avatar`
- `GET /api/auth/avatar/{fileName}`

### Projects

- `GET /api/projects`
- `POST /api/projects`
- `GET /api/projects/{projectId}`
- `PUT /api/projects/{projectId}`
- `POST /api/projects/{projectId}/activate`
- `DELETE /api/projects/{projectId}`
- `POST /api/projects/{projectId}/leave`

### Members

- `GET /api/projects/{projectId}/members`
- `POST /api/projects/{projectId}/members`
- `DELETE /api/projects/{projectId}/members/{memberId}`

### Invites / Notifications

- `GET /api/projects/{projectId}/invites`
- `GET /api/project-invites/incoming`
- `POST /api/project-invites/{token}/accept`
- `POST /api/project-invites/{token}/decline`

### Finance

- Planned expenses:
  - `GET /api/projects/{projectId}/planned-expenses`
  - `POST /api/projects/{projectId}/planned-expenses`
  - `PUT /api/projects/{projectId}/planned-expenses/{id}`
  - `DELETE /api/projects/{projectId}/planned-expenses/{id}`
- Income daily:
  - `GET /api/projects/{projectId}/income-daily`
  - `POST /api/projects/{projectId}/income-daily`
  - `PUT /api/projects/{projectId}/income-daily/cumulative`
  - `DELETE /api/projects/{projectId}/income-daily/{date}`
- Actual expenses:
  - `GET /api/projects/{projectId}/actual-expenses`
  - `POST /api/projects/{projectId}/actual-expenses`
  - `PUT /api/projects/{projectId}/actual-expenses/{id}`
  - `DELETE /api/projects/{projectId}/actual-expenses/{id}`
- Taxes:
  - `GET /api/projects/{projectId}/tax-rules`
  - `POST /api/projects/{projectId}/tax-rules`
  - `PUT /api/projects/{projectId}/tax-rules/reorder`
  - `PUT /api/projects/{projectId}/tax-rules/{id}`
  - `DELETE /api/projects/{projectId}/tax-rules/{id}`
- Profit sharing:
  - `GET /api/projects/{projectId}/profit-share-rules`
  - `POST /api/projects/{projectId}/profit-share-rules`
  - `PUT /api/projects/{projectId}/profit-share-rules/{id}`
  - `DELETE /api/projects/{projectId}/profit-share-rules/{id}`
- Payouts:
  - `GET /api/projects/{projectId}/payout-preview`
  - `GET /api/projects/{projectId}/my-income`
  - `POST /api/projects/{projectId}/payouts/calculate`
  - `GET /api/projects/{projectId}/payout-snapshots/{fileName}`
- Dashboard:
  - `GET /api/projects/{projectId}/dashboard`

### Forecast

- `POST /api/projects/{projectId}/forecast`
- `GET /api/projects/{projectId}/forecast/runs`
- `GET /api/projects/{projectId}/forecast/latest`
- `GET /api/projects/{projectId}/forecast/{runId}`
- `GET /api/projects/{projectId}/forecast/{runId}/report.pdf`
- `DELETE /api/projects/{projectId}/forecast/{runId}`

### Historical campaigns (глобально для пользователя)

- `GET /api/historical-campaigns`
- `POST /api/historical-campaigns`
- `PUT /api/historical-campaigns/{campaignId}`
- `DELETE /api/historical-campaigns/{campaignId}`

## Аналитика (FastAPI)

`POST /forecast` принимает:
- текущую историю доходов по дням,
- горизонт прогноза,
- исторические кампании (ручные + завершенные проекты той же категории),
- текущие суммы дохода/рекламного бюджета.

Backend в этом процессе выполняет orchestration role:
1. собирает фактическую историю доходов проекта;
2. вычисляет текущий ad spend;
3. подмешивает релевантные historical campaigns;
4. вызывает analytics service;
5. нормализует ответ и сохраняет forecast run + forecast points;
6. по запросу формирует PDF-отчёт на стороне backend.

Основная модель analytics:
- state-space (`UnobservedComponents`) + гибридная корректировка формой краудфандинга,
- сценарии `p10 / yhat / p90`,
- детерминированность через seed.

Семантика сценариев:
- `p10` - pessimistic scenario,
- `yhat` - expected scenario,
- `p90` - optimistic scenario.

Offline evaluation:
- baseline comparison и experiment runner описаны в `analytics/README.md`;
- production contract `/forecast` при этом не меняется.
- offline evaluation сравнивает:
  - `naive_last_value`
  - `moving_average`
  - `linear_trend_ols`
  - `crowdfunding_hybrid_v2`

Для академического описания это важно:
- production forecast остаётся отдельным сервисом;
- основная гибридная модель не подменяется новой архитектурой;
- выбор модели можно обосновывать через baseline comparison и метрики качества.

## Структура репозитория

```text
.
├── analytics/
│   └── app/
├── backend/
│   ├── cmd/api
│   └── internal/
│       ├── auth/
│       ├── dashboard/
│       ├── db/
│       │   ├── migrations/
│       │   ├── sql/
│       │   └── sqlc/
│       ├── finance/
│       ├── forecast/
│       ├── fx/
│       └── projects/
├── frontend/
│   └── src/
└── docker-compose.yml
```

## Локальные проверки

Backend:

```bash
cd backend
go test ./...
```

Если в среде несколько Go runtime и важно избежать попытки автозагрузки toolchain, достаточно локального Go `>= 1.23`.
Точный patch-level toolchain проект больше не требует.

Frontend:

```bash
cd frontend
npm ci
npm run typecheck
npm run build
```

Frontend type checking:

```bash
cd frontend
npm run typecheck
```

Важно: frontend `node_modules` не нужно переиспользовать между Windows и WSL.
У Next.js здесь platform-specific `@next/swc` optional binaries, а npm создаёт разные bin-shim под разные ОС.
Если переключились между Windows Node и WSL Node, удалите `node_modules` и `.next`, затем заново выполните `npm ci` в том же runtime, где будете запускать `typecheck` и `build`.

Analytics:

```bash
cd analytics
python -m pip install -r requirements.txt
python -m compileall app scripts tests
python -m unittest discover -s tests -t . -p 'test_*.py'
```

## CI checks

Базовый CI pipeline находится в `.github/workflows/ci.yml` и выполняет:
- `backend`:
  - `gofmt -l .`
  - `go test ./...`
- `frontend`:
  - `npm ci`
  - `npm run typecheck`
  - `npm run build`
- `analytics`:
  - `python -m pip install -r requirements.txt`
  - `python -m compileall app scripts tests`
  - `python -m unittest discover -s tests -t . -p 'test_*.py'`

Что именно подтверждает CI:
- backend unit/contract/regression tests запускаются в CI, фактический результат указан в Actions;
- frontend собирается в production-режиме и проходит строгую TypeScript-проверку;
- analytics service сохраняет рабочий import/runtime-контур и проходит модульные/contract tests.

## Frontend Notes

Frontend сознательно развивается итеративно:
- основные пользовательские сценарии уже закрыты рабочими экранами;
- самые тяжёлые page-компоненты частично декомпозированы на page-level sections;
- глобальный radical rewrite frontend architecture не выполнялся, чтобы не ломать стабильный UI и текущую бизнес-логику.

Это компромиссное, но инженерно оправданное состояние:
- код стал заметно чище в крупных экранах;
- при этом проект не уходил в рискованный рефакторинг перед завершением работы.

## OpenAPI и API-документация

- `backend/openapi.yaml` - актуальная OpenAPI-спецификация backend API;
- `http://localhost:8080/openapi.yaml` - raw schema;
- `http://localhost:8080/swagger` - локальная offline API docs-страница, которую backend обслуживает без внешнего CDN.

Спецификация описывает:
- auth DTO и access-token flow;
- unified error format;
- rate-limited branches для чувствительных auth/invite маршрутов;
- forecast orchestration endpoints и PDF report download;
- ключевые finance и historical campaign endpoints.

Для демонстрации в закрытой сети важно, что `/swagger` и `openapi.yaml` доступны полностью локально.

## Ограничения проекта

Текущие сознательные ограничения:
- frontend architecture улучшена точечно, но не переведена на полностью модульную feature-first схему;
- авторизация остаётся в модели `access-token only`, без refresh tokens и без cookie-session storage;
- offline evaluation для baseline-моделей использует point-forecast comparison, а интервальная production-логика в полном виде относится к основной hybrid model;
- rate limiting реализован как in-memory middleware и не предназначен для распределённого production deployment;
- проект ориентирован на учебный, но инженерно серьёзный уровень, а не на полностью индустриальный enterprise-grade контур.

## Примечания

Для пересборки после смены ветки или крупных изменений используйте:

```bash
docker compose down
docker compose up -d --build
```
