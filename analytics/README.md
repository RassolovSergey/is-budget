# Analytics Service and Offline Evaluation

`analytics` — отдельный сервис прогнозирования, который backend использует как внешний forecasting component.

Важно:
- production endpoint `/forecast` сохраняет стабильный request/response contract;
- основная модель не заменяется, а оформляется как интерпретируемая `crowdfunding_hybrid_v2`;
- offline evaluation живёт отдельно от production endpoint логики.

## Production contract

Production API:
- `GET /health`
- `POST /forecast`

`POST /forecast` принимает:
- `project_id`
- `horizon_days`
- `history`
- `historical_campaigns`
- `current_income_to_date`
- `current_ad_spend_to_date`
- `model_version`
- `seed`

И возвращает:
- `model_version`
- `horizon_days`
- `forecast[]`, где каждый элемент содержит:
  - `date`
  - `p10`
  - `yhat`
  - `p90`

Семантика output:
- `p10` — pessimistic scenario;
- `yhat` — expected scenario;
- `p90` — optimistic scenario.

Входной контракт валидируется через `pydantic`, лишние поля запрещены, а ordering `p10 <= yhat <= p90` проверяется на уровне response model.

Ошибка сервиса оформляется стабильно:

```json
{
  "error": "validation failed",
  "code": "validation_error",
  "details": {}
}
```

## Архитектурная структура

Код разложен по ответственностям:
- `preprocessing.py`
- `features.py`
- `historical_shape.py`
- `forecasting.py`
- `intervals.py`
- `evaluation.py`
- `offline_evaluation.py`

Это позволяет отдельно описывать:
- preprocessing,
- shape/business adjustments,
- основную hybrid model,
- сценарную интерпретацию интервалов,
- baseline comparison и evaluation metrics.

## Что сравнивается

- `naive_last_value`
- `moving_average`
- `linear_trend_ols`
- `crowdfunding_hybrid_v2`

## Метрики

- `MAE`
- `RMSE`
- `sMAPE`
- `pinball_p10`
- `pinball_p90`

Важно:
- production interval logic (`p10 / yhat / p90`) в полном виде реализована для `crowdfunding_hybrid_v2`;
- baseline-модели в offline evaluation остаются point-forecast baselines, поэтому для pinball-метрик у них используется вырожденный интервал `p10 == yhat == p90`;
- это не меняет production `/forecast`, а только честно фиксирует ограничение сравнительного эксперимента.

## Формат входного файла

Скрипт ожидает JSON с массивом `cases`.

Пример:

```json
{
  "cases": [
    {
      "name": "campaign_alpha",
      "history": [
        { "date": "2026-03-01", "amount": 1000 },
        { "date": "2026-03-02", "amount": 300 }
      ],
      "actual_future": [180, 140, 120],
      "historical_campaigns": [],
      "current_ad_spend_to_date": 50,
      "seed": 42
    }
  ]
}
```

Где:
- `history` — наблюдавшийся дневной ряд на момент запуска прогноза;
- `actual_future` — фактически реализовавшиеся будущие дневные значения для сравнения;
- `historical_campaigns` — optional historical campaigns в том же формате, что и production contract;
- `current_ad_spend_to_date` — накопленный ad spend на момент прогноза;
- `seed` — фиксированный seed для воспроизводимости.

## Запуск

Из каталога `analytics`:

```bash
python -m scripts.run_offline_evaluation --input examples/offline_evaluation_sample.json --output-dir artifacts/offline-evaluation
```

Либо эквивалентно:

```bash
python scripts/run_offline_evaluation.py --input examples/offline_evaluation_sample.json --output-dir artifacts/offline-evaluation
```

## Что получится на выходе

В `output-dir` сохраняются:

- `results.json`
- `summary.csv`
- `case_results.csv`
- `summary.md`

Это позволяет:
- вставлять результаты в курсовую работу;
- сравнивать baseline и hybrid model в повторяемом виде;
- хранить экспериментальные результаты отдельно от production API.

## Академический смысл

Offline evaluation нужен для того, чтобы:
- сравнивать основную hybrid model с baseline-кандидатами;
- показывать метрики качества воспроизводимо;
- обосновывать выбор модели в курсовой работе не эвристически, а через controlled comparison.

При этом production API backend↔analytics не меняется.
