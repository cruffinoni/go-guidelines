from pathlib import Path

from go_guidelines_lint.guidelines_parser import parse_guidelines


def test_parse_guidelines_extracts_sections() -> None:
    fixture = Path("tests/fixtures/basic/GO_BEST_PRACTICES.md").resolve()
    sections = parse_guidelines(fixture)

    assert len(sections) == 21
    assert sections[1].title == "Imports and Formatting"
    assert "Use gofmt" in sections[1].do
    assert "go directive" in sections[12].dont
    assert sections[21].title == "Function Declaration Order"


def test_parse_comments_guidelines_extracts_sections() -> None:
    fixture = Path("tests/fixtures/basic/COMMENTS.md").resolve()
    sections = parse_guidelines(fixture)

    assert len(sections) == 11
    assert sections[3].title == "Type Documentation"
