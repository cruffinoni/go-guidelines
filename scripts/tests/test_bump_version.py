"""Regression tests for the project-owned Go release tag tool."""

from __future__ import annotations

import subprocess
import tempfile
import unittest
from pathlib import Path


SCRIPT = Path(__file__).parents[1] / "bump-release-version.sh"


class ReleaseTagToolTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory(prefix="gg-lint-version-test-")
        self.repo = Path(self.temp.name)
        subprocess.run(["git", "init", "-q"], cwd=self.repo, check=True)
        subprocess.run(["git", "config", "user.email", "test@example.invalid"], cwd=self.repo, check=True)
        subprocess.run(["git", "config", "user.name", "Version Test"], cwd=self.repo, check=True)
        (self.repo / "go.mod").write_text("module example.test/gg-lint\n\ngo 1.27.3\n")
        (self.repo / "pyproject.toml").write_text(
            '[project]\nname = "legacy-package"\nversion = "0.1.2"\n'
        )
        (self.repo / "src/legacy_package").mkdir(parents=True)
        (self.repo / "src/legacy_package/__init__.py").write_text('__version__ = "0.1.0"\n')
        subprocess.run(["git", "add", "."], cwd=self.repo, check=True)
        subprocess.run(["git", "commit", "-qm", "initial"], cwd=self.repo, check=True)

    def tearDown(self) -> None:
        self.temp.cleanup()

    def add_tag(self, tag: str) -> None:
        subprocess.run(["git", "tag", "-a", tag, "-m", "fixture"], cwd=self.repo, check=True)

    def run_bump(self, *args: str) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            ["bash", str(SCRIPT), *args],
            cwd=self.repo,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            check=False,
        )

    def test_repeated_prerelease_increments_counter_without_bumping_core(self) -> None:
        self.add_tag("v2.4.0-alpha.1")

        result = self.run_bump("patch", "--pre", "alpha")

        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertIn("Proposed tag: v2.4.0-alpha.2", result.stdout)
        self.assertNotIn("v2.4.1-alpha.2", result.stdout)

    def test_dry_run_does_not_edit_python_metadata_or_changelog(self) -> None:
        before_pyproject = (self.repo / "pyproject.toml").read_text()
        before_module = (self.repo / "src/legacy_package/__init__.py").read_text()

        result = self.run_bump("patch", "--pre", "alpha")

        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertEqual((self.repo / "pyproject.toml").read_text(), before_pyproject)
        self.assertEqual((self.repo / "src/legacy_package/__init__.py").read_text(), before_module)
        self.assertFalse((self.repo / "CHANGELOG.md").exists())

    def test_new_prerelease_starts_from_latest_stable_core(self) -> None:
        self.add_tag("v2.4.0")

        result = self.run_bump("patch", "--pre", "alpha")

        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertIn("Proposed tag: v2.4.1-alpha.1", result.stdout)

    def test_changing_prerelease_label_keeps_core_and_resets_counter(self) -> None:
        self.add_tag("v2.4.0-alpha.2")

        result = self.run_bump("patch", "--pre", "beta")

        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertIn("Proposed tag: v2.4.0-beta.1", result.stdout)

    def test_stable_command_promotes_latest_prerelease_without_incrementing_core(self) -> None:
        self.add_tag("v2.4.0-rc.1")

        result = self.run_bump("stable")

        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertIn("Proposed tag: v2.4.0", result.stdout)

    def test_commit_creates_only_the_release_tag(self) -> None:
        before_head = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=self.repo, text=True).strip()
        before_pyproject = (self.repo / "pyproject.toml").read_text()
        before_module = (self.repo / "src/legacy_package/__init__.py").read_text()

        result = self.run_bump("patch", "--pre", "alpha", "--commit")

        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertEqual(subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=self.repo, text=True).strip(), before_head)
        self.assertEqual(subprocess.check_output(["git", "tag", "--list", "v0.0.1-alpha.1"], cwd=self.repo, text=True).strip(), "v0.0.1-alpha.1")
        self.assertEqual((self.repo / "pyproject.toml").read_text(), before_pyproject)
        self.assertEqual((self.repo / "src/legacy_package/__init__.py").read_text(), before_module)
        self.assertEqual(subprocess.check_output(["git", "status", "--porcelain"], cwd=self.repo, text=True), "")

    def test_push_uses_remote_tags_as_the_version_baseline(self) -> None:
        remote = self.repo.parent / f"{self.repo.name}-origin.git"
        subprocess.run(["git", "init", "--bare", "-q", str(remote)], check=True)
        subprocess.run(["git", "remote", "add", "origin", str(remote)], cwd=self.repo, check=True)
        self.add_tag("v0.0.1-alpha.1")
        subprocess.run(["git", "push", "origin", "v0.0.1-alpha.1"], cwd=self.repo, check=True, capture_output=True)
        subprocess.run(["git", "tag", "-d", "v0.0.1-alpha.1"], cwd=self.repo, check=True, capture_output=True)

        result = self.run_bump("patch", "--pre", "alpha", "--commit", "--push")

        self.assertEqual(result.returncode, 0, result.stdout)
        self.assertIn("Proposed tag: v0.0.1-alpha.2", result.stdout)
        self.assertEqual(
            subprocess.check_output(["git", "ls-remote", "--tags", "origin", "refs/tags/v0.0.1-alpha.2"], cwd=self.repo, text=True).strip().split()[-1],
            "refs/tags/v0.0.1-alpha.2",
        )

    def test_commit_refuses_a_dirty_or_untracked_tree(self) -> None:
        (self.repo / "untracked.txt").write_text("not part of this release\n")

        result = self.run_bump("patch", "--commit")

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("working tree has changes", result.stdout)
        self.assertEqual(subprocess.check_output(["git", "tag", "--list"], cwd=self.repo, text=True), "")


if __name__ == "__main__":
    unittest.main()
