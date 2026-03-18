from __future__ import annotations

import pandas as pd

from .schemas import HistoryPoint


def prepare_history(history: list[HistoryPoint]) -> pd.Series:
    if not history:
        today = pd.Timestamp.utcnow().normalize()
        return pd.Series([0.0], index=pd.DatetimeIndex([today]), dtype="float64")

    rows = sorted(history, key=lambda x: x.date)
    index = pd.DatetimeIndex([pd.Timestamp(point.date) for point in rows])
    values = pd.Series([max(float(point.amount), 0.0) for point in rows], index=index, dtype="float64")

    full_idx = pd.date_range(start=index.min(), end=index.max(), freq="D")
    return values.reindex(full_idx, fill_value=0.0)
