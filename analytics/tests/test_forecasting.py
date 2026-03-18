from __future__ import annotations

import unittest
from datetime import date

from app.model import forecast_local_trend
from app.schemas import ForecastPoint, HistoryPoint


class ForecastingTests(unittest.TestCase):
    def test_forecast_preserves_scenario_ordering(self) -> None:
        points = forecast_local_trend(
            history=[
                HistoryPoint(date=date(2026, 3, 1), amount=1000),
                HistoryPoint(date=date(2026, 3, 2), amount=300),
                HistoryPoint(date=date(2026, 3, 3), amount=200),
            ],
            horizon_days=5,
            seed=42,
        )
        self.assertEqual(len(points), 5)
        for point in points:
            self.assertLessEqual(point.p10, point.yhat)
            self.assertLessEqual(point.yhat, point.p90)

    def test_forecast_handles_short_series(self) -> None:
        points = forecast_local_trend(
            history=[HistoryPoint(date=date(2026, 3, 1), amount=1200)],
            horizon_days=3,
            seed=42,
        )
        self.assertEqual(len(points), 3)
        self.assertTrue(all(isinstance(point, ForecastPoint) for point in points))


if __name__ == "__main__":
    unittest.main()
