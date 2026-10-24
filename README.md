# IS Budget

**Приложение для управления финансами проектной команды: доходы, расходы, распределение средств и прогнозирование.**

![CI](https://github.com/RassolovSergey/IS-Budget-public/actions/workflows/ci.yml/badge.svg)

Проект объединяет финансовую модель, роли участников и планирование работы. Подходит для демонстрации backend-разработки, интеграции сервисов и работы с предметной областью.

## Возможности

| Область | Что делает приложение |
|:--|:--|
| Финансы | Учитывает доходы по дням, плановые и фактические расходы |
| Распределение | Применяет уровни налогов, комиссий и правила распределения средств |
| Команда | Разграничивает доступ к проектам и операциям через RBAC |
| Прогноз | Запрашивает отдельный analytics-сервис и сохраняет результаты |
| Рабочий процесс | Содержит граф задач, зависимости и операции с участниками |
| Отчётность | Формирует PDF-отчёты и снимки выплат |

## Стек и устройство

| Компонент | Технологии |
|:--|:--|
| Backend | Go, chi, pgx, sqlc, JWT |
| Frontend | Next.js, React, TypeScript |
| Analytics | FastAPI, pandas, numpy, statsmodels |
| Данные | PostgreSQL, SQL-миграции |
| Запуск | Docker Compose |

Frontend обращается к backend API. Backend выполняет бизнес-операции, работает с PostgreSQL и вызывает analytics-сервис для прогнозирования.

## Быстрый старт

Требуются Docker и Docker Compose.

```sh
docker compose up -d --build
docker compose ps
```

| Сервис | Адрес |
|:--|:--|
| Интерфейс | <http://localhost:3000> |
| Backend API | <http://localhost:8080/api> |
| Swagger | <http://localhost:8080/swagger> |
| Analytics API | <http://localhost:8000/docs> |

Compose использует локальные development-настройки и демонстрационные секреты. Перед внешним развёртыванием конфигурацию необходимо заменить.

## Проверки

```sh
cd backend
go test ./...
```

```sh
cd frontend
npm ci
npm run typecheck
npm run build
```

```sh
cd analytics
python -m pip install -r requirements.txt
python -m unittest discover -s tests -t . -p 'test_*.py'
```

CI проверяет backend, frontend и analytics отдельно. [Состояние проверок](https://github.com/RassolovSergey/IS-Budget-public/actions).

## Статус и ограничения

Это развивающийся учебный прототип. Наличие функции в коде не является подтверждением её готовности для промышленной эксплуатации.

- Авторизация использует access token без refresh flow.
- Rate limiting хранит состояние в памяти процесса.
- Качество прогноза необходимо оценивать на данных и сравнивать с baseline-моделями.
- Схема рабочего процесса и финансовые операции развиваются независимо.

## Подробнее

[Архитектура, конфигурация, API и ограничения](docs/technical-reference.md)  
[Устройство сервиса прогнозирования](analytics/README.md)
