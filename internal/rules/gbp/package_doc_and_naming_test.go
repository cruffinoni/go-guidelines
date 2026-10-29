package gbp

import (
	"testing"

	"github.com/cruffinoni/go-guidelines/internal/lint/linttest"
)

func TestPackageDocAndNaming(t *testing.T) {
	linttest.Run(t, PackageDocAndNaming{}, "testdata/package_doc_and_naming")
}
