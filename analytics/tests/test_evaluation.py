from __future__ import annotations

import unittest
from datetime import date

import numpy as np

from app.evaluation import evaluate_predictions, linear_trend_ols, moving_average, naive_last_value, prepare_series_for_evaluation
from app.schemas import HistoryPoint


class EvaluationTests(unittest.TestCase):
    def test_baselines_return_expected_shapes(self) -> None:
        series = prepare_series_for_evaluation(
            [
                HistoryPoint(date=date(2026, 3, 1), amount=100),
                HistoryPoint(date=date(2026, 3, 2), amount=150),
                HistoryPoint(date=date(2026, 3, 3), amount=180),
            ]
        )
        self.assertEqual(len(naive_last_value(series, 4)), 4)
        self.assertEqual(len(moving_average(series, 4)), 4)
        self.assertEqual(len(linear_trend_ols(series, 4)), 4)

    def test_metrics_are_non_negative(self) -> None:
        metrics = evaluate_predictions(
            actual=np.asarray([100.0, 120.0, 140.0]),
            yhat=np.asarray([110.0, 118.0, 135.0]),
            p10=np.asarray([90.0, 100.0, 120.0]),
            p90=np.asarray([130.0, 140.0, 160.0]),
        )
        self.assertGreaterEqual(metrics.mae, 0.0)
        self.assertGreaterEqual(metrics.rmse, 0.0)
        self.assertGreaterEqual(metrics.smape, 0.0)
        self.assertGreaterEqual(metrics.pinball_p10, 0.0)
        self.assertGreaterEqual(metrics.pinball_p90, 0.0)


if __name__ == "__main__":
    unittest.main()
