package loader

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sync"
)

// sourceStore collects parsed file bytes under a mutex, since
// packages.Config.ParseFile is called concurrently.
type sourceStore struct {
	mu sync.Mutex
	m  map[string][]byte
}

func newSourceStore() *sourceStore {
	return &sourceStore{m: make(map[string][]byte)}
}

func (s *sourceStore) put(filename string, src []byte) {
	s.mu.Lock()
	s.m[filename] = src
	s.mu.Unlock()
}

// parseFile is a packages.Config.ParseFile hook that parses with go/parser
// and records each file's source bytes in s as a side effect.
func (s *sourceStore) parseFile(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
	f, err := parser.ParseFile(fset, filename, src, parser.AllErrors|parser.ParseComments|parser.SkipObjectResolution)
	s.put(filename, src)
	return f, err
}
