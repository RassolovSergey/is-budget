from __future__ import annotations

import numpy as np

from .features import ObservationFeatures
from .schemas import HistoricalCampaign


def is_ad_category(category: str) -> bool:
    cat = category.strip().lower()
    return cat in {
        "ads",
        "ad",
        "advertising",
        "marketing",
        "paid marketing",
        "реклама",
        "маркетинг",
        "рекламный бюджет",
    }


def default_crowdfunding_shape(total_days: int) -> np.ndarray:
    total_days = max(int(total_days), 1)
    if total_days == 1:
        return np.array([1.0], dtype="float64")

    t = np.linspace(0.0, 1.0, total_days)
    launch_peak = 2.35 * np.exp(-((t - 0.03) / 0.085) ** 2)
    second_day_peak = 0.85 * np.exp(-((t - 0.13) / 0.10) ** 2)
    final_peak = 1.55 * np.exp(-((t - 0.97) / 0.095) ** 2)
    middle_base = 0.42 + 0.04 * np.cos(np.pi * t)

    shape = np.maximum(middle_base + launch_peak + second_day_peak + final_peak, 1e-6)
    return shape / float(np.sum(shape))


def campaign_day_income(campaign: HistoricalCampaign) -> np.ndarray:
    if campaign.duration_days <= 0:
        return np.zeros(1, dtype="float64")

    duration = int(campaign.duration_days)
    arr = np.zeros(duration, dtype="float64")
    for day in campaign.days:
        day_idx = int(day.day_number) - 1
        if 0 <= day_idx < duration:
            arr[day_idx] = max(float(day.income_amount), 0.0)

    if len(arr) >= 2:
        diffs = np.diff(arr)
        non_decreasing_ratio = float(np.mean(diffs >= -1e-9))
        appears_cumulative = (
            non_decreasing_ratio >= 0.88
            and float(arr[-1]) > 0
            and float(np.sum(arr)) > float(arr[-1]) * 1.35
        )
        if appears_cumulative:
            arr = np.diff(np.concatenate(([0.0], arr)))
            arr = np.maximum(arr, 0.0)

    return arr


def campaign_ad_spend(campaign: HistoricalCampaign) -> float:
    total = 0.0
    for day in campaign.days:
        for expense in day.expenses:
            if is_ad_category(expense.category):
                total += max(float(expense.amount), 0.0)
    return total


def resample(values: np.ndarray, target_len: int) -> np.ndarray:
    values = np.asarray(values, dtype="float64")
    target_len = max(int(target_len), 1)

    if len(values) == 0:
        return np.ones(target_len, dtype="float64") / target_len
    if len(values) == target_len:
        return values.copy()
    if len(values) == 1:
        return np.repeat(float(values[0]), target_len)

    src_x = np.linspace(0.0, 1.0, len(values))
    dst_x = np.linspace(0.0, 1.0, target_len)
    return np.interp(dst_x, src_x, values)


def historical_shape(campaigns: list[HistoricalCampaign], total_days: int) -> tuple[np.ndarray, int]:
    default_shape = default_crowdfunding_shape(total_days)
    profiles: list[np.ndarray] = []

    for campaign in campaigns:
        day_income = campaign_day_income(campaign)
        total_income = float(np.sum(day_income))
        if total_income <= 0:
            continue

        shares = day_income / total_income
        resampled = resample(shares, total_days)
        resampled = np.maximum(resampled, 1e-6)
        resampled = resampled / float(np.sum(resampled))
        profiles.append(resampled)

    used = len(profiles)
    if used == 0:
        return default_shape, 0

    hist_shape = np.median(np.stack(profiles), axis=0)
    hist_shape = np.maximum(hist_shape, 1e-6)
    hist_shape = hist_shape / float(np.sum(hist_shape))

    blend = float(np.clip(0.20 + 0.07 * used, 0.25, 0.75))
    shape = (1.0 - blend) * default_shape + blend * hist_shape
    shape = np.maximum(shape, 1e-6)
    shape = shape / float(np.sum(shape))
    return shape, used


def launch_multipliers(campaigns: list[HistoricalCampaign]) -> tuple[float | None, float | None]:
    total_to_day1: list[float] = []
    total_to_day12: list[float] = []

    for campaign in campaigns:
        day_income = campaign_day_income(campaign)
        if len(day_income) == 0:
            continue

        total = float(np.sum(day_income))
        day1 = float(day_income[0])
        day12 = float(day_income[0] + (day_income[1] if len(day_income) > 1 else 0.0))

        if total > 0 and day1 > 0:
            total_to_day1.append(total / day1)
        if total > 0 and day12 > 0:
            total_to_day12.append(total / day12)

    m1 = float(np.median(np.asarray(total_to_day1, dtype="float64"))) if total_to_day1 else None
    m12 = float(np.median(np.asarray(total_to_day12, dtype="float64"))) if total_to_day12 else None
    return m1, m12


def ad_lift(campaigns: list[HistoricalCampaign], current_ad_spend: float) -> float:
    current_ad_spend = max(float(current_ad_spend), 0.0)
    if current_ad_spend <= 0:
        return 1.0

    xs: list[float] = []
    ys: list[float] = []
    for campaign in campaigns:
        total_income = float(np.sum(campaign_day_income(campaign)))
        ad_spend = campaign_ad_spend(campaign)
        if total_income > 0 and ad_spend > 0:
            xs.append(float(np.log1p(ad_spend)))
            ys.append(float(np.log1p(total_income)))

    if len(xs) >= 3:
        x = np.asarray(xs, dtype="float64")
        y = np.asarray(ys, dtype="float64")
        design = np.column_stack([np.ones_like(x), x])
        coeff, *_ = np.linalg.lstsq(design, y, rcond=None)
        slope = float(coeff[1])
        centered = float(np.log1p(current_ad_spend) - np.median(x))
        lift = float(np.exp(slope * centered))
    else:
        lift = float(1.0 + 0.09 * np.log1p(current_ad_spend / 50.0))

    if current_ad_spend >= 50.0:
        lift *= 1.05

    return float(np.clip(lift, 0.75, 1.70))


def historical_total_estimates_by_progress(
    campaigns: list[HistoricalCampaign],
    total_days: int,
    features: ObservationFeatures,
) -> tuple[float, float, float, int]:
    if features.observed_days <= 0 or features.observed_total <= 0:
        return features.observed_total, features.observed_total, features.observed_total, 0

    shares_at_observed: list[float] = []
    for campaign in campaigns:
        day_income = campaign_day_income(campaign)
        total_income = float(np.sum(day_income))
        if total_income <= 0:
            continue

        shares = day_income / total_income
        shares = resample(shares, total_days)
        shares = np.maximum(shares, 1e-9)
        shares = shares / float(np.sum(shares))

        progress_share = float(np.sum(shares[: features.observed_days]))
        if progress_share > 1e-6:
            shares_at_observed.append(progress_share)

    if not shares_at_observed:
        default_shape = default_crowdfunding_shape(total_days)
        progress_share = float(np.sum(default_shape[: features.observed_days]))
        progress_share = float(np.clip(progress_share, 0.16, 0.98))
        total_est = features.observed_total / progress_share
        return total_est, total_est * 1.25, max(features.observed_total, total_est * 0.82), 0

    share_arr = np.asarray(shares_at_observed, dtype="float64")
    share_p10 = float(np.percentile(share_arr, 10))
    share_p50 = float(np.percentile(share_arr, 50))
    share_p90 = float(np.percentile(share_arr, 90))

    share_p10 = float(np.clip(share_p10, 0.10, 0.99))
    share_p50 = float(np.clip(share_p50, 0.12, 0.99))
    share_p90 = float(np.clip(share_p90, 0.14, 0.995))

    total_by_median = features.observed_total / share_p50
    upper_from_share = features.observed_total / share_p10
    lower_from_share = max(features.observed_total, features.observed_total / share_p90)
    return total_by_median, upper_from_share, lower_from_share, len(shares_at_observed)
