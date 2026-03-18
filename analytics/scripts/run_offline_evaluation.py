from __future__ import annotations

import argparse
import csv
import json
from pathlib import Path
from typing import Any

from app.offline_evaluation import (
    OfflineEvaluationCase,
    result_to_json_dict,
    run_offline_evaluation,
    summary_to_markdown,
)
from app.schemas import HistoricalCampaign, HistoryPoint


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run offline evaluation for analytics baselines and hybrid model.")
    parser.add_argument("--input", required=True, help="Path to input JSON file with evaluation cases.")
    parser.add_argument("--output-dir", default="artifacts/offline-evaluation", help="Directory for JSON/CSV/MD outputs.")
    return parser.parse_args()


def load_cases(path: Path) -> list[OfflineEvaluationCase]:
    payload = json.loads(path.read_text(encoding="utf-8"))
    raw_cases = payload.get("cases", [])
    cases: list[OfflineEvaluationCase] = []
    for idx, item in enumerate(raw_cases, start=1):
        # Runner читает ровно тот же shape history/historical_campaigns, что и production contract.
        history = [HistoryPoint.model_validate(point) for point in item.get("history", [])]
        historical_campaigns = [HistoricalCampaign.model_validate(campaign) for campaign in item.get("historical_campaigns", [])]
        actual_future = [float(value) for value in item.get("actual_future", [])]
        cases.append(
            OfflineEvaluationCase(
                name=str(item.get("name") or f"case_{idx}"),
                history=history,
                actual_future=actual_future,
                historical_campaigns=historical_campaigns,
                current_ad_spend_to_date=float(item.get("current_ad_spend_to_date", 0.0)),
                seed=int(item.get("seed", 42)),
            )
        )
    return cases


def write_summary_csv(output_dir: Path, summary: list[dict[str, Any]]) -> None:
    path = output_dir / "summary.csv"
    with path.open("w", encoding="utf-8", newline="") as fh:
        writer = csv.DictWriter(
            fh,
            fieldnames=["model_name", "cases_count", "mae", "rmse", "smape", "pinball_p10", "pinball_p90"],
        )
        writer.writeheader()
        writer.writerows(summary)


def write_case_results_csv(output_dir: Path, case_results: list[dict[str, Any]]) -> None:
    path = output_dir / "case_results.csv"
    with path.open("w", encoding="utf-8", newline="") as fh:
        writer = csv.DictWriter(
            fh,
            fieldnames=["case_name", "model_name", "horizon_days", "mae", "rmse", "smape", "pinball_p10", "pinball_p90"],
        )
        writer.writeheader()
        for item in case_results:
            writer.writerow(
                {
                    "case_name": item["case_name"],
                    "model_name": item["model_name"],
                    "horizon_days": item["horizon_days"],
                    "mae": item["metrics"]["mae"],
                    "rmse": item["metrics"]["rmse"],
                    "smape": item["metrics"]["smape"],
                    "pinball_p10": item["metrics"]["pinball_p10"],
                    "pinball_p90": item["metrics"]["pinball_p90"],
                }
            )


def main() -> None:
    args = parse_args()
    input_path = Path(args.input)
    output_dir = Path(args.output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)

    cases = load_cases(input_path)
    result = run_offline_evaluation(cases)
    payload = result_to_json_dict(result)

    (output_dir / "results.json").write_text(json.dumps(payload, ensure_ascii=False, indent=2), encoding="utf-8")
    (output_dir / "summary.md").write_text(summary_to_markdown(result), encoding="utf-8")
    write_summary_csv(output_dir, payload["summary"])
    write_case_results_csv(output_dir, payload["case_results"])

    print(f"Offline evaluation completed. Results saved to: {output_dir}")


if __name__ == "__main__":
    main()
