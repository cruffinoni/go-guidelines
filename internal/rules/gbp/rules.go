// Package gbp implements the Go best-practices (GBP) rules.
package gbp

import "github.com/cruffinoni/go-guidelines/internal/lint"

// Rules returns every GBP rule, in stable ID order.
func Rules() []lint.Rule {
	return []lint.Rule{
		NoDotImports{},
		PackageDocAndNaming{},
		WrapPropagatedErrors{},
		ContextUsage{},
		ExplicitGoroutineLifecycle{},
		ContextAwareHTTPClient{},
		NoPanicInControlFlow{},
		PreferNilSlice{},
		StructuredLogging{},
		ExplicitTimerLifecycle{},
		ChannelCancellation{},
		DeterministicTests{},
		NoInit{},
		GroupVarDeclarations{},
		MethodCandidateForPointerReceiver{},
		NoNilCheckForRequiredDependency{},
		TopDownFunctionOrder{},
	}
}
