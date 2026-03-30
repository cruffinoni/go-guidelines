from pathlib import Path

from go_guidelines_lint.config import AppConfig
from go_guidelines_lint.runner import has_blocking_findings, run_scan


def test_run_scan_detects_findings(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    bad_go = Path("tests/fixtures/basic/bad.go").read_text(encoding="utf-8")

    (tmp_path / "bad.go").write_text(bad_go, encoding="utf-8")
    (tmp_path / "go.mod").write_text("module example.com/demo\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1)

    result = run_scan(config)

    rule_ids = {finding.rule_id for finding in result.findings}
    assert "GBP001" in rule_ids
    assert "GBP011" in rule_ids
    assert has_blocking_findings(result, "error") is True


def test_run_scan_can_explicitly_enable_default_disabled_gbp012(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    bad_go = Path("tests/fixtures/basic/bad.go").read_text(encoding="utf-8")

    (tmp_path / "bad.go").write_text(bad_go, encoding="utf-8")
    (tmp_path / "go.mod").write_text("module example.com/demo\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1)
    config.rules.enable = ["GBP012"]

    result = run_scan(config)

    rule_ids = {finding.rule_id for finding in result.findings}
    assert "GBP012" in rule_ids


def test_rule_007_flags_only_interfaces_with_10_or_more_methods(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    (tmp_path / "interfaces.go").write_text(
        """
package sample

type Small interface {
    M1()
    M2()
    M3()
    M4()
    M5()
}

type Large interface {
    M1()
    M2()
    M3()
    M4()
    M5()
    M6()
    M7()
    M8()
    M9()
    M10()
}
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1)
    config.rules.enable = ["GBP007"]

    result = run_scan(config)
    messages = [finding.message for finding in result.findings if finding.rule_id == "GBP007"]

    assert any("Interface `Large` has 10 methods" in message for message in messages)
    assert all("Interface `Small` has 5 methods" not in message for message in messages)


def test_rule_018_is_warning_from_medium_confidence(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    (tmp_path / "method_org.go").write_text(
        """
package sample

type Handler struct{}

func BuildAPI(handler *Handler, name string) {}
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1)
    config.rules.enable = ["GBP018"]

    result = run_scan(config)
    gbp018_findings = [finding for finding in result.findings if finding.rule_id == "GBP018"]

    assert gbp018_findings
    assert all(finding.severity == "warning" for finding in gbp018_findings)
    assert has_blocking_findings(result, "error") is False
    assert has_blocking_findings(result, "warning") is True


def test_gbp010_gbp015_and_gbp020_are_disabled_by_default(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    (tmp_path / "payload.go").write_text(
        """
package sample

type Payload struct {
    Items []string `json:"items"`
}

func Build(
    a string,
    b int,
) error {
    return nil
}
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "payload_test.go").write_text(
        """
package sample

import "testing"

func TestPayload(t *testing.T) {
    tests := map[string]int{"a": 1, "b": 2}
    for name := range tests {
        _ = name
    }
}
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1)
    result = run_scan(config)
    rule_ids = {finding.rule_id for finding in result.findings}
    assert "GBP010" not in rule_ids
    assert "GBP015" not in rule_ids
    assert "GBP020" not in rule_ids

    config.rules.enable = ["GBP010", "GBP015", "GBP020"]
    enabled_result = run_scan(config)
    enabled_rule_ids = {finding.rule_id for finding in enabled_result.findings}
    assert "GBP010" in enabled_rule_ids
    assert "GBP015" in enabled_rule_ids
    assert "GBP020" in enabled_rule_ids


def test_rule_020_flags_multiline_signature_that_fits_single_line(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    (tmp_path / "signature_short_multiline.go").write_text(
        """
package sample

func Build(
    a string,
    b int,
) error {
    return nil
}
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1)
    config.rules.enable = ["GBP020"]

    result = run_scan(config)
    gbp020_findings = [finding for finding in result.findings if finding.rule_id == "GBP020"]

    assert gbp020_findings
    assert any("could fit on one line" in finding.message for finding in gbp020_findings)


def test_rule_020_flags_long_single_line_signature_and_suggests_multiline(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    (tmp_path / "signature_long_single_line.go").write_text(
        """
package sample

func Build(a string, b int, c bool, d float64, e []byte, f map[string]int, g chan string) error {
    return nil
}
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1, max_line_length=80)
    config.rules.enable = ["GBP020"]

    result = run_scan(config)
    gbp020_findings = [finding for finding in result.findings if finding.rule_id == "GBP020"]

    assert gbp020_findings
    long_signature = next((f for f in gbp020_findings if "exceeds 80 characters" in f.message), None)
    assert long_signature is not None
    assert long_signature.suggestion is not None
    assert "Use multiline signature:" in long_signature.suggestion
    assert "\n\t" in long_signature.suggestion


def test_only_gbp015_applies_to_test_files(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    (tmp_path / "sample_test.go").write_text(
        """
package sample

import "log"

func TestSample() {
    log.Printf("hello")
    tests := map[string]int{"a": 1, "b": 2}
    for name := range tests {
        _ = name
    }
}
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1)
    config.rules.enable = ["GBP011", "GBP015"]

    result = run_scan(config)
    rule_ids = {finding.rule_id for finding in result.findings}

    assert "GBP015" in rule_ids
    assert "GBP011" not in rule_ids


def test_rule_021_flags_forward_local_function_call(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    (tmp_path / "order_forward.go").write_text(
        """
package sample

func Build() {
    resolve()
}

func resolve() {}
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1)
    config.rules.enable = ["GBP021"]

    result = run_scan(config)
    gbp021_findings = [finding for finding in result.findings if finding.rule_id == "GBP021"]

    assert gbp021_findings
    assert any("calls local function `resolve` before its declaration" in finding.message for finding in gbp021_findings)


def test_rule_021_method_caller_flags_forward_local_function_call(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    (tmp_path / "order_method_forward.go").write_text(
        """
package sample

type service struct{}

func (s *service) Build() {
    resolve()
}

func resolve() {}
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1)
    config.rules.enable = ["GBP021"]

    result = run_scan(config)
    gbp021_findings = [finding for finding in result.findings if finding.rule_id == "GBP021"]

    assert gbp021_findings
    assert any("Function `Build` calls local function `resolve` before its declaration" in finding.message for finding in gbp021_findings)


def test_rule_021_allows_backward_calls_and_contiguous_mutual_recursion(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    (tmp_path / "order_valid.go").write_text(
        """
package sample

func resolve() {}

func Build() {
    resolve()
}

func RecA() {
    RecB()
}

func RecB() {
    RecA()
}
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1)
    config.rules.enable = ["GBP021"]

    result = run_scan(config)
    gbp021_findings = [finding for finding in result.findings if finding.rule_id == "GBP021"]
    assert gbp021_findings == []


def test_rule_021_flags_noncontiguous_mutual_recursion(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    (tmp_path / "order_recursive_split.go").write_text(
        """
package sample

func RecA() {
    RecB()
}

func helper() {}

func RecB() {
    RecA()
}

type Service struct{}

func (s *Service) Use() {
    s.Later()
}

func (s *Service) Later() {}
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1)
    config.rules.enable = ["GBP021"]

    result = run_scan(config)
    gbp021_findings = [finding for finding in result.findings if finding.rule_id == "GBP021"]

    assert gbp021_findings
    assert any("Mutually recursive functions must be declared contiguously" in finding.message for finding in gbp021_findings)
    assert all("Use` calls local function `Later`" not in finding.message for finding in gbp021_findings)


def test_max_line_length_applies_to_signatures_and_comments(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    comments = Path("tests/fixtures/basic/COMMENTS.md").resolve()
    (tmp_path / "long_lines.go").write_text(
        """
package sample

// This comment should exceed the custom maximum line length threshold only.
func Build(a string, b int, c bool, d float64, e []byte) error {
    return nil
}
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(
        guidelines_path=str(guideline),
        comments_guidelines_path=str(comments),
        target="./...",
        max_workers=1,
        max_line_length=50,
    )
    config.rules.enable = ["GBP020", "GCM007"]

    result = run_scan(config)

    gbp020 = [finding for finding in result.findings if finding.rule_id == "GBP020"]
    gcm007 = [finding for finding in result.findings if finding.rule_id == "GCM007"]
    assert gbp020
    assert gcm007
    assert any("50" in finding.message for finding in gbp020)
    assert any("50" in finding.message for finding in gcm007)


def test_comments_rules_run_by_default_when_comments_file_exists(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    comments = Path("tests/fixtures/basic/COMMENTS.md").resolve()
    (tmp_path / "GO_BEST_PRACTICES.md").write_text(guideline.read_text(encoding="utf-8"), encoding="utf-8")
    (tmp_path / "COMMENTS.md").write_text(comments.read_text(encoding="utf-8"), encoding="utf-8")
    (tmp_path / "doc.go").write_text(
        """
package sample

type Widget struct{}
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str((tmp_path / "GO_BEST_PRACTICES.md").resolve()), target="./...", max_workers=1)

    result = run_scan(config)
    rule_ids = {finding.rule_id for finding in result.findings}
    assert "GCM001" in rule_ids


def test_comments_rules_can_be_disabled_explicitly(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    comments = Path("tests/fixtures/basic/COMMENTS.md").resolve()
    (tmp_path / "GO_BEST_PRACTICES.md").write_text(guideline.read_text(encoding="utf-8"), encoding="utf-8")
    (tmp_path / "COMMENTS.md").write_text(comments.read_text(encoding="utf-8"), encoding="utf-8")
    (tmp_path / "doc.go").write_text(
        """
package sample

type Widget struct{}
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str((tmp_path / "GO_BEST_PRACTICES.md").resolve()), target="./...", max_workers=1)
    config.enable_comments_guidelines = False

    result = run_scan(config)
    rule_ids = {finding.rule_id for finding in result.findings}
    assert "GCM001" not in rule_ids


def test_missing_comments_file_is_non_fatal(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    (tmp_path / "GO_BEST_PRACTICES.md").write_text(guideline.read_text(encoding="utf-8"), encoding="utf-8")
    (tmp_path / "main.go").write_text("package main\n", encoding="utf-8")
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str((tmp_path / "GO_BEST_PRACTICES.md").resolve()), target="./...", max_workers=1)

    result = run_scan(config)
    assert result.errors == []


def test_comments_rules_cover_markdown_long_lines_examples_and_tautology(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    comments = Path("tests/fixtures/basic/COMMENTS.md").resolve()
    (tmp_path / "GO_BEST_PRACTICES.md").write_text(guideline.read_text(encoding="utf-8"), encoding="utf-8")
    (tmp_path / "COMMENTS.md").write_text(comments.read_text(encoding="utf-8"), encoding="utf-8")
    (tmp_path / "comment_rules.go").write_text(
        """
package sample

// **Important**: markdown should not be used.
// This comment line is intentionally made very very very very very very very very very very very long.
// GetName returns the name.
func GetName() string { return "" }
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "comment_rules_test.go").write_text(
        """
package sample

func ExampleGetName() {
    _ = GetName()
}
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(
        guidelines_path=str((tmp_path / "GO_BEST_PRACTICES.md").resolve()),
        target="./...",
        max_workers=1,
        max_line_length=80,
    )
    config.rules.enable = ["GCM006", "GCM007", "GCM008", "GCM009"]

    result = run_scan(config)
    rule_ids = {finding.rule_id for finding in result.findings}
    assert "GCM006" in rule_ids
    assert "GCM007" in rule_ids
    assert "GCM008" not in rule_ids
    assert "GCM009" in rule_ids


def test_rule_020_single_line_exactly_at_max_len_does_not_fire(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    sig = "func ExactlyMaxLen(a string, b int, c bool, d float64, e []byte, f map[string]int, gxxxxxxxxxxxxxxxxxxxxxx string) error"
    assert len(sig) == 120
    (tmp_path / "exact.go").write_text(
        f"package sample\n\n{sig} {{\n\treturn nil\n}}\n",
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1)
    config.rules.enable = ["GBP020"]

    result = run_scan(config)
    gbp020_findings = [f for f in result.findings if f.rule_id == "GBP020"]

    assert gbp020_findings == []


def test_rule_020_single_line_one_over_max_len_fires(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    sig = "func ExactlyMaxLen(a string, b int, c bool, d float64, e []byte, f map[string]int, gxxxxxxxxxxxxxxxxxxxxxxx string) error"
    assert len(sig) == 121
    (tmp_path / "one_over.go").write_text(
        f"package sample\n\n{sig} {{\n\treturn nil\n}}\n",
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1)
    config.rules.enable = ["GBP020"]

    result = run_scan(config)
    gbp020_findings = [f for f in result.findings if f.rule_id == "GBP020"]

    assert len(gbp020_findings) == 1
    assert "exceeds 120 characters" in gbp020_findings[0].message


def test_rule_020_multiline_signature_still_too_long_is_flagged(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    (tmp_path / "multiline_too_long.go").write_text(
        """
package sample

func ProcessWithManyLongParamNames(
\talpha string,
\tbeta int,
\tgamma bool,
\tdelta float64,
\tepsilon []byte,
\tzeta map[string]int,
) (string, error) {
\treturn "", nil
}
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1)
    config.rules.enable = ["GBP020"]

    result = run_scan(config)
    gbp020_findings = [f for f in result.findings if f.rule_id == "GBP020"]

    assert len(gbp020_findings) == 1
    assert "still exceeds 120 characters" in gbp020_findings[0].message


def test_rule_020_method_multiline_fits_single_line(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    (tmp_path / "method_multiline.go").write_text(
        """
package sample

type Svc struct{}

func (s *Svc) Do(
\tname string,
\tval int,
) error { return nil }
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1)
    config.rules.enable = ["GBP020"]

    result = run_scan(config)
    gbp020_findings = [f for f in result.findings if f.rule_id == "GBP020"]

    assert gbp020_findings
    assert any("could fit on one line" in f.message for f in gbp020_findings)
    assert any("(s *Svc)" in (f.suggestion or "") for f in gbp020_findings)


def test_rule_020_method_single_line_too_long(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    sig = "func (s *Service) ProcessLongName(alpha string, beta int, gamma bool, delta float64) error"
    assert len(sig) == 90
    (tmp_path / "method_long.go").write_text(
        f"package sample\n\ntype Service struct{{}}\n\n{sig} {{\n\treturn nil\n}}\n",
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1, max_line_length=80)
    config.rules.enable = ["GBP020"]

    result = run_scan(config)
    gbp020_findings = [f for f in result.findings if f.rule_id == "GBP020"]

    assert gbp020_findings
    assert any("exceeds 80 characters" in f.message for f in gbp020_findings)
    assert any("(s *Service)" in (f.suggestion or "") for f in gbp020_findings)


def test_rule_020_multiple_return_values_multiline_fits(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    (tmp_path / "multi_return.go").write_text(
        """
package sample

func Split(
\tinput string,
\tsep string,
) (string, error) { return "", nil }
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1)
    config.rules.enable = ["GBP020"]

    result = run_scan(config)
    gbp020_findings = [f for f in result.findings if f.rule_id == "GBP020"]

    assert gbp020_findings
    assert any("could fit on one line" in f.message for f in gbp020_findings)
    assert any("(string, error)" in (f.suggestion or "") for f in gbp020_findings)


def test_rule_020_variadic_param_multiline_fits(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    (tmp_path / "variadic.go").write_text(
        """
package sample

func Log(
\tmsg string,
\targs ...any,
) {}
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1)
    config.rules.enable = ["GBP020"]

    result = run_scan(config)
    gbp020_findings = [f for f in result.findings if f.rule_id == "GBP020"]

    assert gbp020_findings
    assert any("could fit on one line" in f.message for f in gbp020_findings)
    assert any("...any" in (f.suggestion or "") for f in gbp020_findings)


def test_rule_020_chan_params_single_line_too_long(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    sig = "func Produce(out chan<- string, done <-chan bool, count int, prefix string, tag string) error"
    assert len(sig) == 93
    (tmp_path / "chan_params.go").write_text(
        f"package sample\n\n{sig} {{\n\treturn nil\n}}\n",
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1, max_line_length=80)
    config.rules.enable = ["GBP020"]

    result = run_scan(config)
    gbp020_findings = [f for f in result.findings if f.rule_id == "GBP020"]

    assert gbp020_findings
    assert any("exceeds 80 characters" in f.message for f in gbp020_findings)
    finding = next(f for f in gbp020_findings if "exceeds 80 characters" in f.message)
    assert finding.evidence is not None
    assert "chan<- string" in finding.evidence
    assert "<-chan bool" in finding.evidence


def test_rule_020_map_param_multiline_fits(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    (tmp_path / "map_param.go").write_text(
        """
package sample

func ProcessItems(
\tdata map[string]int,
\tname string,
\tvalue bool,
) error { return nil }
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1)
    config.rules.enable = ["GBP020"]

    result = run_scan(config)
    gbp020_findings = [f for f in result.findings if f.rule_id == "GBP020"]

    assert gbp020_findings
    assert any("could fit on one line" in f.message for f in gbp020_findings)
    assert any("map[string]int" in (f.suggestion or "") for f in gbp020_findings)


def test_rule_020_function_type_param_single_line_too_long(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    sig = "func Transform(data []byte, fn func([]byte) ([]byte, error), timeout int) ([]byte, error)"
    assert len(sig) == 89
    (tmp_path / "fn_param.go").write_text(
        f"package sample\n\n{sig} {{\n\treturn nil, nil\n}}\n",
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1, max_line_length=80)
    config.rules.enable = ["GBP020"]

    result = run_scan(config)
    gbp020_findings = [f for f in result.findings if f.rule_id == "GBP020"]

    assert gbp020_findings
    assert any("exceeds 80 characters" in f.message for f in gbp020_findings)
    assert any(f.suggestion is not None for f in gbp020_findings)


def test_rule_020_no_param_long_function_name_fires(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    sig = "func InitializeDefaultApplicationConfigurationForAllRegisteredComponentsX() error"
    assert len(sig) == 81
    (tmp_path / "long_name.go").write_text(
        f"package sample\n\n{sig} {{\n\treturn nil\n}}\n",
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1, max_line_length=80)
    config.rules.enable = ["GBP020"]

    result = run_scan(config)
    gbp020_findings = [f for f in result.findings if f.rule_id == "GBP020"]

    assert gbp020_findings
    assert any("exceeds 80 characters" in f.message for f in gbp020_findings)
    assert any(f.suggestion is not None for f in gbp020_findings)


def test_rule_020_generic_function_does_not_crash(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    (tmp_path / "generic.go").write_text(
        """
package sample

func Map[T, U any](slice []T, fn func(T) U) []U { return nil }
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1, max_line_length=40)
    config.rules.enable = ["GBP020"]

    result = run_scan(config)
    gbp020_findings = [f for f in result.findings if f.rule_id == "GBP020"]

    assert len(gbp020_findings) == 1
    assert gbp020_findings[0].suggestion == "Break signature into multiple lines with one parameter per line."


def test_rule_020_clean_functions_produce_no_findings(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    (tmp_path / "clean.go").write_text(
        """
package sample

func Add(a, b int) int { return a + b }

func Greet(name string) string { return name }
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1)
    config.rules.enable = ["GBP020"]

    result = run_scan(config)
    gbp020_findings = [f for f in result.findings if f.rule_id == "GBP020"]

    assert gbp020_findings == []


def test_rule_020_mixed_file_flags_only_bad_function(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    (tmp_path / "mixed.go").write_text(
        """
package sample

func Add(a, b int) int { return a + b }

func Build(
\ta string,
\tb int,
) error {
\treturn nil
}
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1)
    config.rules.enable = ["GBP020"]

    result = run_scan(config)
    gbp020_findings = [f for f in result.findings if f.rule_id == "GBP020"]

    assert len(gbp020_findings) == 1
    assert "could fit on one line" in gbp020_findings[0].message


def test_rule_001_dot_import_in_block_produces_single_finding(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    (tmp_path / "dot.go").write_text(
        """
package sample

import (
    . "fmt"
    "os"
)
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1)
    config.rules.enable = ["GBP001"]

    result = run_scan(config)
    gbp001 = [f for f in result.findings if f.rule_id == "GBP001"]

    assert len(gbp001) == 1
    assert "Dot-import found in import block" in gbp001[0].message


def test_rule_001_standalone_dot_import_produces_single_finding(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    (tmp_path / "dot.go").write_text(
        """
package sample

import . "fmt"
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1)
    config.rules.enable = ["GBP001"]

    result = run_scan(config)
    gbp001 = [f for f in result.findings if f.rule_id == "GBP001"]

    assert len(gbp001) == 1
    assert "pollutes namespace" in gbp001[0].message


def test_rule_001_mixed_standalone_and_block_produces_two_findings(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    (tmp_path / "dot.go").write_text(
        """
package sample

import . "log"

import (
    . "fmt"
    "os"
)
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1)
    config.rules.enable = ["GBP001"]

    result = run_scan(config)
    gbp001 = [f for f in result.findings if f.rule_id == "GBP001"]

    assert len(gbp001) == 2
    assert any("pollutes namespace" in f.message for f in gbp001)
    assert any("Dot-import found in import block" in f.message for f in gbp001)


def test_rule_019_does_not_flag_err_nil_check_in_constructor(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    (tmp_path / "ctor.go").write_text(
        """
package sample

type Foo struct{}

func NewFoo() (*Foo, error) {
    f, err := build()
    if err == nil {
        return f, nil
    }
    return nil, err
}

func build() (*Foo, error) { return &Foo{}, nil }
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1)
    config.rules.enable = ["GBP019"]

    result = run_scan(config)
    gbp019 = [f for f in result.findings if f.rule_id == "GBP019"]

    assert gbp019 == [], f"Expected no findings, got: {gbp019}"


def test_rule_008_wrong_defer_detected_with_blank_line_between(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    (tmp_path / "defer.go").write_text(
        """
package sample

import "os"

func Open(name string) {
    f, err := os.Open(name)

    defer f.Close()

    if err != nil {
        return
    }
    _ = f
}
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1)
    config.rules.enable = ["GBP008"]

    result = run_scan(config)
    gbp008 = [f for f in result.findings if f.rule_id == "GBP008"]

    assert any("defer" in f.message.lower() or "close" in f.message.lower() for f in gbp008)


def test_rule_006_body_close_in_other_function_does_not_suppress_finding(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    (tmp_path / "http.go").write_text(
        """
package sample

import "net/http"

func FetchWithClose() {
    resp, _ := http.Get("https://example.com")
    defer resp.Body.Close()
    _ = resp
}

func FetchWithoutClose() {
    resp, _ := http.Get("https://example.com")
    _ = resp
}
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1)
    config.rules.enable = ["GBP006"]

    result = run_scan(config)
    body_close_findings = [
        f for f in result.findings
        if f.rule_id == "GBP006" and "body" in f.message.lower()
    ]

    assert body_close_findings, "Expected a body-close finding for FetchWithoutClose"


def test_panic_err_reported_only_by_gbp003_not_gbp008(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    (tmp_path / "panic.go").write_text(
        """
package sample

func Run(err error) {
    panic(err)
}
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(guidelines_path=str(guideline), target="./...", max_workers=1)
    config.rules.enable = ["GBP003", "GBP008"]

    result = run_scan(config)
    rule_ids = [f.rule_id for f in result.findings]

    assert "GBP003" in rule_ids
    assert "GBP008" not in rule_ids


def test_comment_rules_do_not_apply_to_test_files_by_default(tmp_path: Path, monkeypatch) -> None:
    guideline = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    comments = Path("tests/fixtures/basic/COMMENTS.md").resolve()
    (tmp_path / "GO_BEST_PRACTICES.md").write_text(guideline.read_text(encoding="utf-8"), encoding="utf-8")
    (tmp_path / "COMMENTS.md").write_text(comments.read_text(encoding="utf-8"), encoding="utf-8")
    (tmp_path / "comment_rules_test.go").write_text(
        """
package sample

// **Important**: markdown should not be used.
// This comment line is intentionally made very very very very very very very very very very very long.
func ExampleGetName() {
}
""".strip(),
        encoding="utf-8",
    )
    (tmp_path / "go.mod").write_text("module example.com/demo\ngo 1.22\n", encoding="utf-8")

    monkeypatch.chdir(tmp_path)
    config = AppConfig(
        guidelines_path=str((tmp_path / "GO_BEST_PRACTICES.md").resolve()),
        target="./...",
        max_workers=1,
        max_line_length=80,
    )
    config.rules.enable = ["GCM006", "GCM007", "GCM008"]

    result = run_scan(config)
    rule_ids = {finding.rule_id for finding in result.findings}
    assert "GCM006" not in rule_ids
    assert "GCM007" not in rule_ids
    assert "GCM008" not in rule_ids
