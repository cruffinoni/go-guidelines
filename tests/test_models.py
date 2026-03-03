from go_guidelines_lint.models import Finding, RuleMeta


def test_severity_is_derived_from_confidence_for_rulemeta_and_finding() -> None:
    assert RuleMeta("GBP001", 1, "Imports", severity="warning", confidence="high").severity == "error"
    assert RuleMeta("GBP002", 2, "Docs", severity="error", confidence="medium").severity == "warning"
    assert RuleMeta("GBP005", 5, "Concurrency", severity="error", confidence="low").severity == "info"

    finding = Finding(
        rule_id="GBP005",
        guideline_number=5,
        title="Concurrency",
        severity="error",
        confidence="low",
        message="test",
        file="x.go",
    )
    assert finding.severity == "info"
