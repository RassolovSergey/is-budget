from __future__ import annotations

import unittest
from datetime import date

from app.offline_evaluation import OfflineEvaluationCase, result_to_json_dict, run_offline_evaluation
from app.schemas import HistoryPoint


class OfflineEvaluationRunnerTests(unittest.TestCase):
    def test_runner_returns_expected_output_shape(self) -> None:
        result = run_offline_evaluation(
            [
                OfflineEvaluationCase(
                    name="case_a",
                    history=[
                        HistoryPoint(date=date(2026, 3, 1), amount=100),
                        HistoryPoint(date=date(2026, 3, 2), amount=120),
                    ],
                    actual_future=[90.0, 85.0],
                    historical_campaigns=[],
                    current_ad_spend_to_date=0.0,
                    seed=42,
                )
            ]
        )
        payload = result_to_json_dict(result)
        self.assertEqual(payload["cases_evaluated"], 1)
        self.assertGreaterEqual(len(payload["summary"]), 1)
        self.assertGreaterEqual(len(payload["case_results"]), 1)

    def test_runner_uses_registered_candidates(self) -> None:
        result = run_offline_evaluation(
            [
                OfflineEvaluationCase(
                    name="case_a",
                    history=[HistoryPoint(date=date(2026, 3, 1), amount=100)],
                    actual_future=[80.0],
                    historical_campaigns=[],
                    current_ad_spend_to_date=0.0,
                    seed=42,
                )
            ]
        )
        model_names = {item.model_name for item in result.summary}
        self.assertIn("naive_last_value", model_names)
        self.assertIn("moving_average", model_names)
        self.assertIn("linear_trend_ols", model_names)
        self.assertIn("crowdfunding_hybrid_v2", model_names)


if __name__ == "__main__":
    unittest.main()
