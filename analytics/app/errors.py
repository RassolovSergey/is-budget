from __future__ import annotations

from typing import Any

from fastapi import FastAPI, Request
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse


def install_exception_handlers(app: FastAPI) -> None:
    @app.exception_handler(RequestValidationError)
    async def handle_validation_error(_: Request, exc: RequestValidationError) -> JSONResponse:
        details: dict[str, str] = {}
        for err in exc.errors():
            location = ".".join(str(part) for part in err.get("loc", []) if part != "body")
            if location:
                details[location] = err.get("type", "invalid")
        return JSONResponse(
            status_code=422,
            content={
                "error": "validation failed",
                "code": "validation_error",
                "details": details,
            },
        )

    @app.exception_handler(Exception)
    async def handle_unexpected_error(_: Request, exc: Exception) -> JSONResponse:
        _ = exc
        return JSONResponse(
            status_code=500,
            content={
                "error": "internal analytics error",
                "code": "internal_error",
            },
        )
