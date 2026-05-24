from go_guidelines_lint.models import Finding, RuleMeta


def test_severity_is_explicit_and_independent_from_confidence() -> None:
    assert RuleMeta("GBP001", 1, "Imports", severity="warning", confidence="high").severity == "warning"
    assert RuleMeta("GBP002", 2, "Docs", severity="error", confidence="medium").severity == "error"
    assert RuleMeta("GBP005", 5, "Concurrency", severity="error", confidence="low").severity == "error"

    finding = Finding(
        rule_id="GBP005",
        guideline_number=5,
        title="Concurrency",
        severity="error",
        confidence="low",
        message="test",
        file="x.go",
    )
    assert finding.severity == "error"
