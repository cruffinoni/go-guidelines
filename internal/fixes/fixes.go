// Package fixes contains deterministic, in-process source transformations.
// It deliberately does not execute commands or infer fixes from arbitrary text.
package fixes

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Edit is a source change applicable only to the exact byte range in Before.
type Edit struct {
	RuleID string
	Start  int
	End    int
	Before []byte
	After  []byte
}

var simpleVarDeclaration = regexp.MustCompile(`^var [A-Za-z_][A-Za-z0-9_]*(?:\s+(?:[A-Za-z_][A-Za-z0-9_]*|=.+))?$`)

// Plan returns non-overlapping GBP017 edits for simple adjacent top-level vars.
// Declarations with comments, grouped syntax, or complex type expressions are
// intentionally left untouched rather than guessed at.
func Plan(source []byte) ([]Edit, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "source.go", source, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return nil, err
	}
	eligible := make(map[int]string)
	for _, decl := range file.Decls {
		group, ok := decl.(*ast.GenDecl)
		if !ok || group.Tok != token.VAR || group.Lparen.IsValid() || group.Doc != nil || len(group.Specs) != 1 {
			continue
		}
		value, ok := group.Specs[0].(*ast.ValueSpec)
		if !ok || value.Doc != nil || value.Comment != nil || len(value.Names) != 1 {
			continue
		}
		start := fset.Position(group.Pos()).Offset
		end := fset.Position(group.End()).Offset
		lineStart := bytes.LastIndexByte(source[:start], '\n') + 1
		lineEnd := bytes.IndexByte(source[end:], '\n')
		if lineEnd < 0 {
			lineEnd = len(source)
		} else {
			lineEnd += end
		}
		line := string(source[lineStart:lineEnd])
		if strings.TrimSpace(line) != line || !simpleVarDeclaration.MatchString(line) {
			continue
		}
		eligible[lineStart] = line
	}

	lines := bytes.SplitAfter(source, []byte{'\n'})
	starts := make([]int, len(lines))
	for i := 1; i < len(lines); i++ {
		starts[i] = starts[i-1] + len(lines[i-1])
	}
	var edits []Edit
	for i := 0; i < len(lines); {
		if _, ok := eligible[starts[i]]; !ok {
			i++
			continue
		}
		j := i + 1
		for j < len(lines) {
			if _, ok := eligible[starts[j]]; !ok {
				break
			}
			j++
		}
		if j-i >= 3 {
			start := starts[i]
			end := starts[j-1] + len(lines[j-1])
			block := make([]string, 0, j-i+2)
			block = append(block, "var (")
			for k := i; k < j; k++ {
				line := strings.TrimSuffix(string(lines[k]), "\n")
				line = strings.TrimSuffix(line, "\r")
				block = append(block, "\t"+strings.TrimPrefix(line, "var "))
			}
			block = append(block, ")")
			after := strings.Join(block, "\n")
			if bytes.HasSuffix(lines[j-1], []byte{'\n'}) {
				after += "\n"
			}
			edits = append(edits, Edit{RuleID: "GBP017", Start: start, End: end, Before: append([]byte(nil), source[start:end]...), After: []byte(after)})
		}
		i = j
	}
	return validate(edits, len(source))
}

// Apply verifies edit applicability against the unchanged source and applies
// edits from right to left so offsets remain stable.
func Apply(source []byte, edits []Edit) ([]byte, error) {
	validated, err := validate(edits, len(source))
	if err != nil {
		return nil, err
	}
	for _, edit := range validated {
		if !bytes.Equal(source[edit.Start:edit.End], edit.Before) {
			return nil, fmt.Errorf("%s fix is no longer applicable at byte %d", edit.RuleID, edit.Start)
		}
	}
	result := append([]byte(nil), source...)
	for i := len(validated) - 1; i >= 0; i-- {
		edit := validated[i]
		result = append(result[:edit.Start], append(append([]byte(nil), edit.After...), result[edit.End:]...)...)
	}
	return result, nil
}

func validate(edits []Edit, sourceLen int) ([]Edit, error) {
	ordered := append([]Edit(nil), edits...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Start < ordered[j].Start })
	for i, edit := range ordered {
		if edit.Start < 0 || edit.End < edit.Start || edit.End > sourceLen {
			return nil, fmt.Errorf("invalid %s fix range [%d,%d)", edit.RuleID, edit.Start, edit.End)
		}
		if i > 0 && ordered[i-1].End > edit.Start {
			return nil, errors.New("conflicting overlapping fixes")
		}
	}
	return ordered, nil
}

// WriteAtomic replaces a regular file through a same-directory temporary file
// and rename, preserving its permission bits. It rejects content that is stale
// when checked, but does not serialize concurrent writers.
func WriteAtomic(path string, original, updated []byte) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("refusing to replace non-regular source file")
	}
	current, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if sha256.Sum256(current) != sha256.Sum256(original) {
		return errors.New("source changed since fix was planned; refusing to overwrite")
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".gg-lint-fix-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(info.Mode().Perm()); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(updated); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
