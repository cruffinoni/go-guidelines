package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunConfigRuleSettingIsOverriddenByExplicitFlagWithoutMutation(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "gg-lint.toml")
	config := []byte("[rules]\nGBP001 = false\n")
	if err := os.WriteFile(configPath, config, 0o600); err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(root, "sample.go")
	if err := os.WriteFile(filename, []byte("package sample\nimport . \"fmt\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--config", configPath, "--enable-rule", "GBP001", "--format=json", filename}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run() exit code = %d, want 1; stderr=%q", code, stderr.String())
	}
	var result struct {
		Findings []json.RawMessage `json:"findings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("stdout is not JSON: %v; stdout=%q", err, stdout.String())
	}
	if len(result.Findings) != 1 {
		t.Fatalf("findings = %d, want 1 after CLI override", len(result.Findings))
	}
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, config) {
		t.Fatalf("config changed after override: before %q, after %q", config, after)
	}
}

func TestRunConfigSetsDefaultsAndCLIFlagWins(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "gg-lint.toml")
	if err := os.WriteFile(configPath, []byte("format = \"json\"\nfail_on = \"none\"\n[rules]\nGBP001 = false\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(root, "sample.go")
	if err := os.WriteFile(filename, []byte("package sample\nimport . \"fmt\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--config", configPath, "--enable-rule", "GBP001", "--fail-on", "error", filename}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run() exit code = %d, want 1 because explicit --fail-on wins; stderr=%q", code, stderr.String())
	}
	var result struct {
		Findings []json.RawMessage `json:"findings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("config format=json was not applied: %v; stdout=%q", err, stdout.String())
	}
	if len(result.Findings) != 1 {
		t.Fatalf("findings = %d, want 1", len(result.Findings))
	}
}

func TestRunRejectsUnknownConfigRule(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "gg-lint.toml")
	if err := os.WriteFile(configPath, []byte("[rules]\nGBP999 = true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--config", configPath}, &stdout, &stderr)
	if code != 2 || stderr.Len() == 0 {
		t.Fatalf("Run() = %d, stderr=%q; want config error and exit 2", code, stderr.String())
	}
}

func TestRunRejectsConflictingRuleFlags(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "sample.go")
	if err := os.WriteFile(filename, []byte("package sample\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--enable-rule", "GBP001", "--disable-rule", "GBP001", filename}, &stdout, &stderr)
	if code != 2 || stderr.Len() == 0 {
		t.Fatalf("Run() = %d, stderr=%q; want conflict error and exit 2", code, stderr.String())
	}
}

func TestRunRejectsInvalidConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "gg-lint.toml")
	if err := os.WriteFile(configPath, []byte("fail_on = \"catastrophe\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--config", configPath}, &stdout, &stderr)
	if code != 2 || stderr.Len() == 0 {
		t.Fatalf("Run() = %d, stderr=%q; want config error and exit 2", code, stderr.String())
	}
}

func TestRunDiscoversConfigFromRepositoryRootWhenNested(t *testing.T) {
	root := newConfigTestProject(t)
	writeGoFile(t, root, "pkg/sample.go", "package sample\nimport . \"fmt\"\n")
	if err := os.WriteFile(filepath.Join(root, "gg-lint.toml"), []byte("format = \"json\"\n[rules]\nGBP001 = false\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	chdirConfigTest(t, filepath.Join(root, "pkg"))

	var stdout, stderr bytes.Buffer
	code := Run(nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() = %d, want 0 with discovered disabled rule; stderr=%q", code, stderr.String())
	}
	var result struct {
		Findings []json.RawMessage `json:"findings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("discovered config did not select JSON output: %v; stdout=%q", err, stdout.String())
	}
	if len(result.Findings) != 0 {
		t.Fatalf("findings = %d, want 0 from repository config", len(result.Findings))
	}
}

func TestRunUsesDefaultsWhenRepositoryConfigIsMissing(t *testing.T) {
	root := newConfigTestProject(t)
	writeGoFile(t, root, "pkg/sample.go", "package sample\n")
	chdirConfigTest(t, root)
	var stdout, stderr bytes.Buffer
	code := Run([]string{"."}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), "No findings") {
		t.Fatalf("Run() = %d, stdout=%q stderr=%q; want default text output", code, stdout.String(), stderr.String())
	}
}

func TestRunExplicitConfigOverridesDiscoveredRepositoryConfig(t *testing.T) {
	root := newConfigTestProject(t)
	writeGoFile(t, root, "pkg/sample.go", "package sample\n")
	if err := os.WriteFile(filepath.Join(root, "gg-lint.toml"), []byte("format = \"json\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	explicitPath := filepath.Join(root, "custom.toml")
	if err := os.WriteFile(explicitPath, []byte("format = \"text\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	chdirConfigTest(t, root)
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--config", explicitPath, "."}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), "No findings") {
		t.Fatalf("explicit config did not override discovered config: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestRunReportsMalformedDiscoveredConfig(t *testing.T) {
	root := newConfigTestProject(t)
	if err := os.WriteFile(filepath.Join(root, "gg-lint.toml"), []byte("format = [broken\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	chdirConfigTest(t, root)
	var stdout, stderr bytes.Buffer
	code := Run([]string{"."}, &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "gg-lint.toml:1") {
		t.Fatalf("Run() = %d, stderr=%q; want a user-facing config path and line", code, stderr.String())
	}
}

func TestRunDoesNotMutateDiscoveredConfigWhenFlagsOverrideIt(t *testing.T) {
	root := newConfigTestProject(t)
	writeGoFile(t, root, "sample.go", "package sample\nimport . \"fmt\"\n")
	original := []byte("format = \"json\"\nfail_on = \"none\"\n[rules]\nGBP001 = false\n")
	configPath := filepath.Join(root, "gg-lint.toml")
	if err := os.WriteFile(configPath, original, 0o600); err != nil {
		t.Fatal(err)
	}
	chdirConfigTest(t, root)
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--enable-rule", "GBP001", "--fail-on", "error", "."}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run() = %d, want 1 after flag overrides; stderr=%q", code, stderr.String())
	}
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, original) {
		t.Fatalf("discovered config changed: before %q, after %q", original, after)
	}
}

func newConfigTestProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	command := exec.Command("git", "init", "-q", root)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	return root
}

func chdirConfigTest(t *testing.T, directory string) {
	t.Helper()
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(directory); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
}
