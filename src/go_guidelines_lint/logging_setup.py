"""Centralized logging configuration for the CLI."""

from __future__ import annotations

from datetime import datetime, timezone
import json
import logging


class JsonFormatter(logging.Formatter):
    """Serialize log records to JSON lines."""

    def format(self, record: logging.LogRecord) -> str:
        payload = {
            "ts": datetime.now(timezone.utc).isoformat(),
            "level": record.levelname,
            "logger": record.name,
            "message": record.getMessage(),
        }
        if record.exc_info:
            payload["exc_info"] = self.formatException(record.exc_info)
        return json.dumps(payload, ensure_ascii=True)


def configure_logging(level: str = "INFO", fmt: str = "text") -> None:
    """Configure process-wide logging once for this CLI execution."""

    root = logging.getLogger()
    root.handlers.clear()

    handler = logging.StreamHandler()
    if fmt == "json":
        handler.setFormatter(JsonFormatter())
    else:
        handler.setFormatter(logging.Formatter("%(asctime)s %(levelname)s [%(name)s] %(message)s"))

    root.addHandler(handler)
    root.setLevel(level.upper())
