from pathlib import Path

from go_guidelines_lint.discovery import resolve_go_files


def test_resolve_go_files_directory_and_recursive(tmp_path: Path) -> None:
    src = tmp_path / "src"
    vendor = tmp_path / "vendor"
    src.mkdir()
    vendor.mkdir()
    (src / "a.go").write_text("package a\n", encoding="utf-8")
    (vendor / "b.go").write_text("package b\n", encoding="utf-8")

    files = resolve_go_files("./...", ["**/*.go"], ["**/vendor/**"], cwd=tmp_path)

    assert [f.name for f in files] == ["a.go"]


def test_resolve_go_files_single_file(tmp_path: Path) -> None:
    file_path = tmp_path / "main.go"
    file_path.write_text("package main\n", encoding="utf-8")

    files = resolve_go_files(str(file_path), ["**/*.go"], [], cwd=tmp_path)

    assert files == [file_path.resolve()]
