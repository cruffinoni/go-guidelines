"""Exercise the inline release-tag check used by GitHub Actions."""

from __future__ import annotations

import re
import subprocess
import textwrap
import unittest
from pathlib import Path


WORKFLOW = Path(__file__).parents[2] / ".github/workflows/release.yml"


class ReleaseWorkflowTagTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        text = WORKFLOW.read_text()
        match = re.search(r"(?m)^        run: \|\n((?:^          .*\n)+)", text)
        if match is None:
            raise AssertionError("could not find the inline validate-tag run block")
        cls.validation_step = textwrap.dedent(match.group(1))

    def classify(self, tag: str, created: bool = True) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            ["bash", "-c", self.validation_step],
            env={"PATH": "/usr/bin:/bin", "RELEASE_TAG": tag, "TAG_CREATED": str(created).lower(), "GITHUB_OUTPUT": "/dev/stdout"},
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            check=False,
        )

    def test_stable_and_prerelease_tags_are_accepted(self) -> None:
        expected = {
            "v2.4.0": "make_latest=true",
            "v2.4.0-alpha": "make_latest=false",
            "v2.4.0-alpha.1": "make_latest=false",
            "v2.4.0-alpha.2": "make_latest=false",
            "v2.4.0-rc.1": "make_latest=false",
            "v2.4.0-dev.1": "make_latest=false",
        }
        for tag, classification in expected.items():
            with self.subTest(tag=tag):
                result = self.classify(tag)
                self.assertEqual(result.returncode, 0, result.stdout)
                self.assertIn(classification, result.stdout)

    def test_goreleaser_receives_the_exact_event_tag(self) -> None:
        workflow = WORKFLOW.read_text()
        self.assertIn("GORELEASER_CURRENT_TAG: ${{ github.ref_name }}", workflow)

    def test_existing_tag_updates_are_rejected(self) -> None:
        result = self.classify("v2.4.0", created=False)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("newly created", result.stdout)

    def test_invalid_tags_are_rejected(self) -> None:
        for tag in ("2.4.0", "v02.4.0", "v2.4", "v2.4.0-alpha.x", "v2.4.0-1"):
            with self.subTest(tag=tag):
                self.assertNotEqual(self.classify(tag).returncode, 0)


if __name__ == "__main__":
    unittest.main()
