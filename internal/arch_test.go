// internal/arch_test.go
package internal_test

import (
	"go/build"
	"strings"
	"testing"
)

type rule struct {
	from string
	deny []string
}

var archRules = []rule{
	{from: "github.com/agbruneau/taugo/internal/tau", deny: []string{
		"github.com/agbruneau/taugo/internal/orchestration",
		"github.com/agbruneau/taugo/internal/bridge",
		"github.com/agbruneau/taugo/internal/app",
	}},
	{from: "github.com/agbruneau/taugo/internal/tau/dimensions", deny: []string{
		"github.com/agbruneau/taugo/internal/tau/invariants",
	}},
	{from: "github.com/agbruneau/taugo/internal/tau/invariants", deny: []string{
		"github.com/agbruneau/taugo/internal/tau/dimensions",
		"github.com/agbruneau/taugo/internal/orchestration",
		"github.com/agbruneau/taugo/internal/bridge",
	}},
	{from: "github.com/agbruneau/taugo/internal/bridge/llm", deny: []string{
		"github.com/agbruneau/taugo/internal/tau",
	}},
	{from: "github.com/agbruneau/taugo/internal/bridge/agentmeshkafka", deny: []string{
		"github.com/agbruneau/taugo/internal/tau",
	}},
}

func TestArchitectureLayering(t *testing.T) {
	t.Parallel()
	for _, r := range archRules {
		t.Run(strings.ReplaceAll(r.from, "/", "_"), func(t *testing.T) {
			t.Parallel()
			pkg, err := build.Default.Import(r.from, ".", build.ImportComment)
			if err != nil {
				// package may not exist yet in M0; skip without failing
				t.Skipf("package %s not built yet: %v", r.from, err)
			}
			imports := append([]string{}, pkg.Imports...)
			imports = append(imports, pkg.TestImports...)
			for _, imp := range imports {
				for _, denied := range r.deny {
					if imp == denied || strings.HasPrefix(imp, denied+"/") {
						t.Errorf("forbidden import: %s imports %s", r.from, imp)
					}
				}
			}
		})
	}
}
