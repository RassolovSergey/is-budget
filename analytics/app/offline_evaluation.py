from __future__ import annotations

from dataclasses import asdict, dataclass
from statistics import mean

import numpy as np

from .evaluation import (
    EvaluationMetrics,
    baseline_registry,
    candidate_model_names,
    evaluate_predictions,
    hybrid_current_model,
    prepare_series_for_evaluation,
    round2,
)
from .schemas import HistoricalCampaign, HistoryPoint


@dataclass(frozen=True)
class OfflineEvaluationCase:
    name: str
    history: list[HistoryPoint]
    actual_future: list[float]
    historical_campaigns: list[HistoricalCampaign]
    current_ad_spend_to_date: float
    seed: int

    @property
    def horizon_days(self) -> int:
        return len(self.actual_future)


@dataclass(frozen=True)
class OfflineCaseModelResult:
    case_name: str
    model_name: str
    horizon_days: int
    metrics: EvaluationMetrics


@dataclass(frozen=True)
class OfflineModelSummary:
    model_name: str
    cases_count: int
    mae: float
    rmse: float
    smape: float
    pinball_p10: float
    pinball_p90: float


@dataclass(frozen=True)
class OfflineEvaluationResult:
    cases_evaluated: int
    candidate_models: list[str]
    case_results: list[OfflineCaseModelResult]
    summary: list[OfflineModelSummary]


def run_offline_evaluation(cases: list[OfflineEvaluationCase]) -> OfflineEvaluationResult:
    case_results: list[OfflineCaseModelResult] = []

    for case in cases:
        if case.horizon_days <= 0:
            continue
        actual = np.asarray(case.actual_future, dtype="float64")

        for model_name, predicted, p10, p90 in generate_case_predictions(case):
            metrics = evaluate_predictions(actual=actual, yhat=predicted, p10=p10, p90=p90)
            case_results.append(
                OfflineCaseModelResult(
                    case_name=case.name,
                    model_name=model_name,
                    horizon_days=case.horizon_days,
                    metrics=metrics,
                )
            )

    return OfflineEvaluationResult(
        cases_evaluated=len(cases),
        candidate_models=candidate_model_names(),
        case_results=case_results,
        summary=aggregate_case_results(case_results),
    )


def generate_case_predictions(
    case: OfflineEvaluationCase,
) -> list[tuple[str, np.ndarray, np.ndarray, np.ndarray]]:
    series = np.asarray([point.amount for point in case.history], dtype="float64")
    series = np.maximum(series, 0.0)

    outputs: list[tuple[str, np.ndarray, np.ndarray, np.ndarray]] = []
    for model_name, baseline_fn in baseline_registry().items():
        baseline = baseline_fn(
            prepare_series_for_evaluation(case.history),
            case.horizon_days,
        )
        baseline = np.asarray(baseline, dtype="float64")
        # У baseline-моделей нет собственной интервальной оценки, поэтому для pinball-метрик
        # используем вырожденный интервал p10 == yhat == p90.
        outputs.append((model_name, baseline, baseline, baseline))

    hybrid_yhat = hybrid_current_model(
        history=case.history,
        horizon_days=case.horizon_days,
        historical_campaigns=case.historical_campaigns,
        current_ad_spend_to_date=case.current_ad_spend_to_date,
        seed=case.seed,
    )
    hybrid_yhat = np.asarray(hybrid_yhat, dtype="float64")
    hybrid_p10 = np.maximum(hybrid_yhat * 0.82, 0.0)
    hybrid_p90 = np.maximum(hybrid_yhat * 1.18, hybrid_yhat)
    outputs.append(("crowdfunding_hybrid_v2", hybrid_yhat, hybrid_p10, hybrid_p90))
    return outputs


def aggregate_case_results(case_results: list[OfflineCaseModelResult]) -> list[OfflineModelSummary]:
    grouped: dict[str, list[OfflineCaseModelResult]] = {}
    for result in case_results:
        grouped.setdefault(result.model_name, []).append(result)

    summaries: list[OfflineModelSummary] = []
    for model_name in candidate_model_names():
        items = grouped.get(model_name, [])
        if not items:
            continue
        summaries.append(
            OfflineModelSummary(
                model_name=model_name,
                cases_count=len(items),
                mae=round2(mean(item.metrics.mae for item in items)),
                rmse=round2(mean(item.metrics.rmse for item in items)),
                smape=round2(mean(item.metrics.smape for item in items)),
                pinball_p10=round2(mean(item.metrics.pinball_p10 for item in items)),
                pinball_p90=round2(mean(item.metrics.pinball_p90 for item in items)),
            )
        )
    return summaries


def result_to_json_dict(result: OfflineEvaluationResult) -> dict:
    return {
        "cases_evaluated": result.cases_evaluated,
        "candidate_models": result.candidate_models,
        "summary": [asdict(item) for item in result.summary],
        "case_results": [
            {
                "case_name": item.case_name,
                "model_name": item.model_name,
                "horizon_days": item.horizon_days,
                "metrics": asdict(item.metrics),
            }
            for item in result.case_results
        ],
    }


def summary_to_markdown(result: OfflineEvaluationResult) -> str:
    lines = [
        "# Offline Evaluation Summary",
        "",
        f"- Cases evaluated: {result.cases_evaluated}",
        f"- Candidate models: {', '.join(result.candidate_models)}",
        "",
        "| Model | Cases | MAE | RMSE | sMAPE | Pinball p10 | Pinball p90 |",
        "|---|---:|---:|---:|---:|---:|---:|",
    ]
    for item in result.summary:
        lines.append(
            f"| {item.model_name} | {item.cases_count} | {item.mae:.2f} | {item.rmse:.2f} | {item.smape:.2f} | {item.pinball_p10:.2f} | {item.pinball_p90:.2f} |"
        )
    return "\n".join(lines) + "\n"
