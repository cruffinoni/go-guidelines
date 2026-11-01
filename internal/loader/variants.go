package loader

import "strings"

// dedupeTestVariants keeps "p [p.test]" instead of the plain "p" package it
// was compiled from, keeps the external "p_test [p.test]" package, and drops
// the synthesized "p.test" test-binary main package. It must be applied per
// load group: package IDs such as "command-line-arguments" repeat across
// groups loaded from different module roots.
func dedupeTestVariants(pkgs []*Package) []*Package {
	byID := make(map[string]*Package, len(pkgs))
	for _, p := range pkgs {
		byID[p.ID] = p
	}

	drop := make(map[string]bool)
	for id := range byID {
		if strings.HasSuffix(id, ".test") && !strings.Contains(id, "[") {
			drop[id] = true
		}
	}
	for id := range byID {
		if drop[id] || strings.Contains(id, "[") {
			continue
		}
		if _, ok := byID[id+" ["+id+".test]"]; ok {
			drop[id] = true
		}
	}

	out := make([]*Package, 0, len(pkgs))
	for _, p := range pkgs {
		if !drop[p.ID] {
			out = append(out, p)
		}
	}
	return out
}
