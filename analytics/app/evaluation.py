from __future__ import annotations

from dataclasses import dataclass
from typing import Callable

import numpy as np
import pandas as pd

from .forecasting import forecast_local_trend, round2
from .preprocessing import prepare_history
from .schemas import HistoricalCampaign, HistoryPoint


@dataclass(frozen=True)
class BaselineResult:
    name: str
    daily_forecast: np.ndarray


@dataclass(frozen=True)
class EvaluationMetrics:
    mae: float
    rmse: float
    smape: float
    pinball_p10: float
    pinball_p90: float


def naive_last_value(series: pd.Series, horizon_days: int) -> np.ndarray:
    values = np.maximum(series.to_numpy(dtype="float64"), 0.0)
    base = float(values[-1] if len(values) else 0.0)
    return np.full(horizon_days, base, dtype="float64")


def moving_average(series: pd.Series, horizon_days: int, window: int = 3) -> np.ndarray:
    values = np.maximum(series.to_numpy(dtype="float64"), 0.0)
    if len(values) == 0:
        return np.zeros(horizon_days, dtype="float64")
    used_window = max(1, min(window, len(values)))
    mean_value = float(np.mean(values[-used_window:]))
    return np.full(horizon_days, mean_value, dtype="float64")


def linear_trend_ols(series: pd.Series, horizon_days: int) -> np.ndarray:
    values = np.maximum(series.to_numpy(dtype="float64"), 0.0)
    if len(values) <= 1:
        return naive_last_value(series, horizon_days)
    x = np.arange(len(values), dtype="float64")
    design = np.column_stack([np.ones_like(x), x])
    coeff, *_ = np.linalg.lstsq(design, values, rcond=None)
    future_x = np.arange(len(values), len(values) + horizon_days, dtype="float64")
    preds = coeff[0] + coeff[1] * future_x
    return np.maximum(preds, 0.0)


def hybrid_current_model(
    history: list[HistoryPoint],
    horizon_days: int,
    historical_campaigns: list[HistoricalCampaign] | None = None,
    current_ad_spend_to_date: float = 0.0,
    seed: int = 42,
) -> np.ndarray:
    points = forecast_local_trend(
        history=history,
        horizon_days=horizon_days,
        seed=seed,
        historical_campaigns=historical_campaigns,
        current_ad_spend_to_date=current_ad_spend_to_date,
    )
    return np.asarray([point.yhat for point in points], dtype="float64")


def mae(actual: np.ndarray, predicted: np.ndarray) -> float:
    return float(np.mean(np.abs(actual - predicted)))


def rmse(actual: np.ndarray, predicted: np.ndarray) -> float:
    return float(np.sqrt(np.mean(np.square(actual - predicted))))


def smape(actual: np.ndarray, predicted: np.ndarray) -> float:
    denominator = np.abs(actual) + np.abs(predicted)
    denominator = np.where(denominator == 0, 1.0, denominator)
    return float(np.mean(2.0 * np.abs(predicted - actual) / denominator) * 100.0)


def pinball_loss(actual: np.ndarray, predicted: np.ndarray, alpha: float) -> float:
    diff = actual - predicted
    return float(np.mean(np.maximum(alpha * diff, (alpha - 1.0) * diff)))


def evaluate_predictions(actual: np.ndarray, yhat: np.ndarray, p10: np.ndarray | None = None, p90: np.ndarray | None = None) -> EvaluationMetrics:
    actual = np.asarray(actual, dtype="float64")
    yhat = np.asarray(yhat, dtype="float64")
    if p10 is None:
        p10 = yhat
    if p90 is None:
        p90 = yhat
    p10 = np.asarray(p10, dtype="float64")
    p90 = np.asarray(p90, dtype="float64")

    return EvaluationMetrics(
        mae=round2(mae(actual, yhat)),
        rmse=round2(rmse(actual, yhat)),
        smape=round2(smape(actual, yhat)),
        pinball_p10=round2(pinball_loss(actual, p10, 0.10)),
        pinball_p90=round2(pinball_loss(actual, p90, 0.90)),
    )


def baseline_registry() -> dict[str, Callable[[pd.Series, int], np.ndarray]]:
    return {
        "naive_last_value": naive_last_value,
        "moving_average": moving_average,
        "linear_trend_ols": linear_trend_ols,
    }


def candidate_model_names() -> list[str]:
    return [
        "naive_last_value",
        "moving_average",
        "linear_trend_ols",
        "crowdfunding_hybrid_v2",
    ]


def prepare_series_for_evaluation(history: list[HistoryPoint]) -> pd.Series:
    return prepare_history(history)
