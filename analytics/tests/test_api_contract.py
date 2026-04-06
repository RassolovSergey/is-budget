from __future__ import annotations

import unittest

from fastapi.testclient import TestClient

from app.main import app


class APIContractTests(unittest.TestCase):
    def test_validation_error_has_stable_shape(self) -> None:
        client = TestClient(app)
        response = client.post("/forecast", json={"project_id": "p1", "horizon_days": 0, "history": []})
        self.assertEqual(response.status_code, 422)
        payload = response.json()
        self.assertEqual(payload["code"], "validation_error")
        self.assertIn("error", payload)
        self.assertIn("details", payload)

    def test_forecast_response_contract_is_stable(self) -> None:
        client = TestClient(app)
        response = client.post(
            "/forecast",
            json={
                "project_id": "p1",
                "horizon_days": 2,
                "history": [
                    {"date": "2026-03-01", "amount": 1000},
                    {"date": "2026-03-02", "amount": 300},
                ],
                "historical_campaigns": [],
                "current_income_to_date": 1300,
                "current_ad_spend_to_date": 0,
                "model_version": "crowdfunding-hybrid-v2",
                "seed": 42,
            },
        )
        self.assertEqual(response.status_code, 200)
        payload = response.json()
        self.assertEqual(payload["model_version"], "crowdfunding-hybrid-v2")
        self.assertEqual(payload["horizon_days"], 2)
        self.assertEqual(len(payload["forecast"]), 2)

    def test_negative_income_inputs_are_accepted_by_contract(self) -> None:
        client = TestClient(app)
        response = client.post(
            "/forecast",
            json={
                "project_id": "p-negative",
                "horizon_days": 2,
                "history": [
                    {"date": "2026-03-01", "amount": 120},
                    {"date": "2026-03-02", "amount": -40},
                ],
                "historical_campaigns": [
                    {
                        "name": "campaign-negative",
                        "duration_days": 3,
                        "currency_code": "EUR",
                        "started_at": "2026-02-01",
                        "ended_at": "2026-02-03",
                        "days": [
                            {"day_number": 1, "income_amount": 80, "expenses": []},
                            {"day_number": 2, "income_amount": -20, "expenses": []},
                            {"day_number": 3, "income_amount": 30, "expenses": []},
                        ],
                    }
                ],
                "current_income_to_date": -15,
                "current_ad_spend_to_date": 0,
                "model_version": "crowdfunding-hybrid-v2",
                "seed": 42,
            },
        )
        self.assertEqual(response.status_code, 200)
        payload = response.json()
        self.assertEqual(payload["horizon_days"], 2)
        self.assertEqual(len(payload["forecast"]), 2)


if __name__ == "__main__":
    unittest.main()
