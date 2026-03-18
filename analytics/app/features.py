from __future__ import annotations

from dataclasses import dataclass

import numpy as np
import pandas as pd


@dataclass(frozen=True)
class ObservationFeatures:
    observed_days: int
    observed_total: float
    first_day_total: float
    first_two_days_total: float


def extract_observation_features(series: pd.Series) -> ObservationFeatures:
    observed = np.maximum(series.to_numpy(dtype="float64"), 0.0)
    observed_days = int(len(observed))
    observed_total = float(np.sum(observed))
    first_day_total = float(observed[0]) if observed_days > 0 else 0.0
    first_two_days_total = float(observed[:2].sum()) if observed_days > 1 else first_day_total
    return ObservationFeatures(
        observed_days=observed_days,
        observed_total=observed_total,
        first_day_total=first_day_total,
        first_two_days_total=first_two_days_total,
    )
