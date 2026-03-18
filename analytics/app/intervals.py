from __future__ import annotations

import numpy as np
import pandas as pd


def scenario_bounds(
    base: np.ndarray,
    future_shape: np.ndarray,
    observed_series: pd.Series,
    ss_p10: np.ndarray,
    ss_p90: np.ndarray,
) -> tuple[np.ndarray, np.ndarray]:
    obs = np.maximum(observed_series.to_numpy(dtype="float64"), 0.0)
    if len(obs) >= 2:
        window = obs[-min(len(obs), 7) :]
        mean_w = float(np.mean(window))
        std_w = float(np.std(window))
        volatility = std_w / max(mean_w, 1.0)
    else:
        volatility = 0.25
    volatility = float(np.clip(volatility, 0.10, 0.85))

    peak = future_shape / max(float(np.mean(future_shape)), 1e-6)
    down_spread = np.clip(0.16 + 0.40 * volatility * peak, 0.10, 0.60)
    up_spread = np.clip(0.20 + 0.50 * volatility * peak, 0.14, 1.05)

    p10 = base * (1.0 - down_spread)
    p90 = base * (1.0 + up_spread)

    p10 = 0.55 * p10 + 0.45 * ss_p10
    p90 = 0.55 * p90 + 0.45 * ss_p90

    p10 = np.maximum(p10, base * 0.18)
    p10 = np.maximum(p10, 0.0)
    p90 = np.maximum(p90, np.maximum(base * 1.06, p10 + 1e-6))
    return ensure_scenario_ordering(base, p10, p90)


def ensure_scenario_ordering(
    yhat: np.ndarray, p10: np.ndarray, p90: np.ndarray
) -> tuple[np.ndarray, np.ndarray]:
    p10 = np.minimum(np.maximum(p10, 0.0), np.maximum(yhat, 0.0))
    p90 = np.maximum(np.maximum(p90, 0.0), np.maximum(yhat, p10))
    return p10, p90
