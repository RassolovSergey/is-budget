from __future__ import annotations

from datetime import date

from pydantic import BaseModel, ConfigDict, Field, field_validator, model_validator


class AnalyticsBaseModel(BaseModel):
    model_config = ConfigDict(extra="forbid")


class HistoryPoint(AnalyticsBaseModel):
    date: date
    amount: float


class HistoricalCampaignExpense(AnalyticsBaseModel):
    category: str = Field(min_length=1, max_length=120)
    amount: float = Field(ge=0)


class HistoricalCampaignDay(AnalyticsBaseModel):
    day_number: int = Field(ge=1)
    income_amount: float
    expenses: list[HistoricalCampaignExpense] = Field(default_factory=list)


class HistoricalCampaign(AnalyticsBaseModel):
    name: str = Field(min_length=2, max_length=120)
    duration_days: int = Field(ge=3, le=60)
    currency_code: str = Field(default="EUR", min_length=3, max_length=3)
    started_at: date | None = None
    ended_at: date | None = None
    days: list[HistoricalCampaignDay] = Field(default_factory=list)

    @field_validator("currency_code")
    @classmethod
    def normalize_currency_code(cls, value: str) -> str:
        return value.strip().upper()

    @model_validator(mode="after")
    def validate_period_and_days(self) -> "HistoricalCampaign":
        if self.started_at and self.ended_at and self.ended_at < self.started_at:
            raise ValueError("ended_at must be on or after started_at")
        if self.days:
            seen: set[int] = set()
            for day in self.days:
                if day.day_number > self.duration_days:
                    raise ValueError("day_number exceeds duration_days")
                if day.day_number in seen:
                    raise ValueError("duplicate day_number")
                seen.add(day.day_number)
        return self


class ForecastRequest(AnalyticsBaseModel):
    project_id: str = Field(min_length=1, max_length=120)
    horizon_days: int = Field(ge=1, le=30)
    history: list[HistoryPoint]
    historical_campaigns: list[HistoricalCampaign] = Field(default_factory=list)
    current_income_to_date: float = 0
    current_ad_spend_to_date: float = Field(default=0, ge=0)
    model_version: str = "crowdfunding-hybrid-v2"
    seed: int = 42

    @field_validator("model_version")
    @classmethod
    def normalize_model_version(cls, value: str) -> str:
        return value.strip() or "crowdfunding-hybrid-v2"

    @model_validator(mode="after")
    def validate_request_consistency(self) -> "ForecastRequest":
        if self.historical_campaigns is None:
            self.historical_campaigns = []
        return self


class ForecastPoint(AnalyticsBaseModel):
    date: date
    yhat: float
    p10: float
    p90: float

    @model_validator(mode="after")
    def validate_scenario_ordering(self) -> "ForecastPoint":
        if self.p10 > self.yhat or self.yhat > self.p90:
            raise ValueError("scenario ordering must satisfy p10 <= yhat <= p90")
        return self


class ForecastResponse(AnalyticsBaseModel):
    model_version: str
    horizon_days: int
    forecast: list[ForecastPoint]

    @field_validator("model_version")
    @classmethod
    def normalize_response_model_version(cls, value: str) -> str:
        return value.strip()
