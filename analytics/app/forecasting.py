from __future__ import annotations

from datetime import timedelta
from typing import Any, cast

import numpy as np
import pandas as pd
from statsmodels.tsa.statespace.structural import UnobservedComponents

from .features import extract_observation_features
from .historical_shape import ad_lift, historical_shape, historical_total_estimates_by_progress, launch_multipliers
from .intervals import scenario_bounds
from .schemas import ForecastPoint, HistoricalCampaign, HistoryPoint
from .preprocessing import prepare_history

Z_10_90 = 1.2815515655446004
MAX_FORECAST_AMOUNT = 9999999999.99


def state_space_forecast(series: pd.Series, horizon_days: int) -> tuple[np.ndarray, np.ndarray, np.ndarray]:
    endog = np.maximum(series.to_numpy(dtype="float64"), 0.0)

    if len(endog) <= 1:
        base = float(endog[-1] if len(endog) else 0.0)
        yhat = np.full(horizon_days, base, dtype="float64")
        p10 = np.maximum(yhat * 0.5, 0.0)
        p90 = yhat * 1.6
        return yhat, p10, p90

    transformed = np.log1p(endog)
    levels: tuple[str, ...] = ("local linear trend", "local level")
    for level_name in levels:
        try:
            model_ctor = cast(Any, UnobservedComponents)
            model = model_ctor(transformed, **{"level": level_name})
            fitted = cast(Any, model.fit(disp=False))
            forecast_res = cast(Any, fitted.get_forecast(steps=horizon_days))

            mu = np.asarray(forecast_res.predicted_mean, dtype="float64")
            se = np.asarray(forecast_res.se_mean, dtype="float64")
            se = np.where(np.isfinite(se), se, 0.0)
            se = np.maximum(se, 1e-4)

            yhat = np.expm1(mu)
            p10 = np.expm1(mu - Z_10_90 * se)
            p90 = np.expm1(mu + Z_10_90 * se)

            yhat = np.maximum(yhat, 0.0)
            p10 = np.maximum(p10, 0.0)
            p90 = np.maximum(p90, yhat)
            return yhat, p10, p90
        except Exception:
            continue

    base = float(endog[-1])
    yhat = np.full(horizon_days, base, dtype="float64")
    trend = float(np.mean(np.diff(endog[-min(len(endog), 5) :]))) if len(endog) >= 3 else 0.0
    for i in range(horizon_days):
        yhat[i] = max(base + trend * (i + 1), 0.0)
    p10 = np.maximum(yhat * 0.55, 0.0)
    p90 = yhat * 1.55
    return yhat, p10, p90


def normalize_forecast_amount(v: float) -> float:
    if np.isnan(v) or np.isinf(v):
        return 0.0
    if v < 0:
        return 0.0
    if v > MAX_FORECAST_AMOUNT:
        return MAX_FORECAST_AMOUNT
    return round2(v)


def hybrid_base_forecast(
    series: pd.Series,
    horizon_days: int,
    campaigns: list[HistoricalCampaign],
    current_ad_spend_to_date: float,
    ss_yhat: np.ndarray,
) -> tuple[np.ndarray, np.ndarray]:
    features = extract_observation_features(series)
    total_days = max(features.observed_days + horizon_days, horizon_days)
    shape, campaigns_used = historical_shape(campaigns, total_days)

    if features.observed_days < total_days:
        future_shape = shape[features.observed_days : features.observed_days + horizon_days]
    else:
        future_shape = np.ones(horizon_days, dtype="float64")
    future_shape = np.maximum(future_shape, 1e-9)
    future_shape = future_shape / float(np.sum(future_shape))

    share_total_est, share_upper_est, share_lower_est, share_used = historical_total_estimates_by_progress(
        campaigns=campaigns,
        total_days=total_days,
        features=features,
    )

    mult_day1, mult_day12 = launch_multipliers(campaigns)
    launch_total_est = None
    if mult_day12 is not None and features.first_two_days_total > 0:
        launch_total_est = features.first_two_days_total * mult_day12
    elif mult_day1 is not None and features.first_day_total > 0:
        launch_total_est = features.first_day_total * mult_day1

    ss_total_est = features.observed_total + float(np.sum(ss_yhat))

    estimates: list[tuple[float, float]] = [(ss_total_est, 0.28)]
    estimates.append((share_total_est, 0.50 + 0.025 * min(share_used, 10)))
    if launch_total_est is not None:
        estimates.append((launch_total_est, 0.22))

    weights = np.asarray([w for _, w in estimates], dtype="float64")
    values = np.asarray([v for v, _ in estimates], dtype="float64")
    weight_sum = float(np.sum(weights))
    total_target = ss_total_est if weight_sum <= 0 else float(np.dot(values, weights) / weight_sum)
    total_target *= ad_lift(campaigns, current_ad_spend_to_date)

    lower_bound = max(features.observed_total, share_lower_est * 0.98)
    upper_candidates: list[float] = [share_upper_est * 1.03]
    if launch_total_est is not None:
        upper_candidates.append(launch_total_est * 1.22)
    if campaigns_used == 0 and share_used == 0:
        upper_candidates.append(max(features.observed_total + float(np.sum(ss_yhat)) * 1.45, features.observed_total * 1.35))
    upper_bound = max(lower_bound, min(upper_candidates))
    total_target = float(np.clip(total_target, lower_bound, upper_bound))

    remaining_target = max(total_target - features.observed_total, 0.0)
    shape_daily = remaining_target * future_shape

    shape_weight = 0.62 - 0.05 * min(features.observed_days, 8) + 0.035 * min(campaigns_used, 10)
    if campaigns_used == 0:
        shape_weight = min(shape_weight, 0.42)
    shape_weight = float(np.clip(shape_weight, 0.28, 0.82))

    base = (1.0 - shape_weight) * ss_yhat + shape_weight * shape_daily
    base = np.maximum(base, 0.0)
    return base, future_shape


def forecast_local_trend(
    history: list[HistoryPoint],
    horizon_days: int,
    seed: int,
    historical_campaigns: list[HistoricalCampaign] | None = None,
    current_income_to_date: float = 0.0,
    current_ad_spend_to_date: float = 0.0,
) -> list[ForecastPoint]:
    np.random.seed(int(seed) % (2**32 - 1))
    _ = current_income_to_date

    series = prepare_history(history)
    last_date = pd.Timestamp(series.index[-1])

    ss_yhat, ss_p10, ss_p90 = state_space_forecast(series, horizon_days)
    campaigns = historical_campaigns or []
    base, future_shape = hybrid_base_forecast(
        series=series,
        horizon_days=horizon_days,
        campaigns=campaigns,
        current_ad_spend_to_date=float(current_ad_spend_to_date),
        ss_yhat=ss_yhat,
    )
    p10, p90 = scenario_bounds(
        base=base,
        future_shape=future_shape,
        observed_series=series,
        ss_p10=np.maximum(ss_p10, 0.0),
        ss_p90=np.maximum(ss_p90, 0.0),
    )

    points: list[ForecastPoint] = []
    for idx in range(horizon_days):
        day = (last_date + timedelta(days=idx + 1)).date()
        points.append(
            ForecastPoint(
                date=day,
                yhat=float(normalize_forecast_amount(base[idx])),
                p10=float(normalize_forecast_amount(p10[idx])),
                p90=float(normalize_forecast_amount(p90[idx])),
            )
        )
    return points


def round2(v: float) -> float:
    return float(int(v * 100 + 0.5)) / 100
