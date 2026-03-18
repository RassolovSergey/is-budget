from __future__ import annotations

from fastapi import FastAPI

from .errors import install_exception_handlers
from .model import forecast_local_trend
from .schemas import ForecastRequest, ForecastResponse

app = FastAPI(
    title="IS Budget Analytics API",
    version="1.1.0",
    description="Hybrid crowdfunding-aware forecast service returning daily scenarios (p10/yhat/p90)",
)
install_exception_handlers(app)


@app.get("/health")
def health() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/forecast", response_model=ForecastResponse)
def run_forecast(req: ForecastRequest) -> ForecastResponse:
    points = forecast_local_trend(
        history=req.history,
        horizon_days=req.horizon_days,
        seed=req.seed,
        historical_campaigns=req.historical_campaigns,
        current_income_to_date=req.current_income_to_date,
        current_ad_spend_to_date=req.current_ad_spend_to_date,
    )
    return ForecastResponse(
        model_version=req.model_version,
        horizon_days=req.horizon_days,
        forecast=points,
    )
