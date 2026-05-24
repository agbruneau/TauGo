// internal/arch_test.go
package internal_test

import (
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
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
		"github.com/agbruneau/taugo/internal/orchestration",
		"github.com/agbruneau/taugo/internal/app",
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

// TestBridgeNoTauImport walks every non-test .go file under internal/bridge/*
// and fails if any file imports internal/tau or a sub-package thereof.
// Uses go/parser (AST) so it catches violations even in packages that do not
// compile yet, providing an early-warning gate for future bridge sub-packages.
func TestBridgeNoTauImport(t *testing.T) {
	t.Parallel()

	const tauPrefix = "github.com/agbruneau/taugo/internal/tau"

	// Locate the bridge directory relative to this test's source file.
	// GOFILE is not available at test time; use build.Default to find the
	// module root via the internal package itself.
	intPkg, err := build.Default.Import("github.com/agbruneau/taugo/internal/calibration", ".", build.FindOnly)
	if err != nil {
		t.Skipf("module root not resolvable: %v", err)
	}
	bridgeRoot := filepath.Join(filepath.Dir(intPkg.Dir), "bridge")

	if _, statErr := os.Stat(bridgeRoot); os.IsNotExist(statErr) {
		t.Skipf("internal/bridge does not exist yet")
	}

	fset := token.NewFileSet()
	err = filepath.WalkDir(bridgeRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if parseErr != nil {
			t.Errorf("parse error %s: %v", path, parseErr)
			return nil
		}
		for _, imp := range f.Imports {
			// Strip surrounding quotes from the import path literal.
			raw := strings.Trim(imp.Path.Value, `"`)
			if raw == tauPrefix || strings.HasPrefix(raw, tauPrefix+"/") {
				t.Errorf("bridge isolation violation: %s imports %s", path, raw)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir failed: %v", err)
	}
}
