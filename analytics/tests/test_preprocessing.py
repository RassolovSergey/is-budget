from __future__ import annotations

import unittest
from datetime import date

from app.preprocessing import prepare_history
from app.schemas import HistoryPoint


class PrepareHistoryTests(unittest.TestCase):
    def test_prepare_history_fills_missing_days_with_zero(self) -> None:
        series = prepare_history(
            [
                HistoryPoint(date=date(2026, 3, 1), amount=100),
                HistoryPoint(date=date(2026, 3, 3), amount=40),
            ]
        )
        self.assertEqual(len(series), 3)
        self.assertEqual(float(series.iloc[0]), 100.0)
        self.assertEqual(float(series.iloc[1]), 0.0)
        self.assertEqual(float(series.iloc[2]), 40.0)

    def test_prepare_history_empty_returns_single_zero_point(self) -> None:
        series = prepare_history([])
        self.assertEqual(len(series), 1)
        self.assertEqual(float(series.iloc[0]), 0.0)


if __name__ == "__main__":
    unittest.main()
