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
	// AUDIT V-A2: calibration must not depend on tau/*, orchestration, or bridge/*.
	// These packages are higher-level; pulling them into calibration would invert
	// the dependency direction and violate Clean Architecture layer ordering.
	{from: "github.com/agbruneau/taugo/internal/calibration", deny: []string{
		"github.com/agbruneau/taugo/internal/tau",
		"github.com/agbruneau/taugo/internal/orchestration",
		"github.com/agbruneau/taugo/internal/bridge",
	}},
	// ADR-0006: internal/thresholds is a leaf value-type package with no taugo deps.
	// It must not import any other taugo internal package to preserve the transverse layer.
	{from: "github.com/agbruneau/taugo/internal/thresholds", deny: []string{
		"github.com/agbruneau/taugo/internal/tau",
		"github.com/agbruneau/taugo/internal/orchestration",
		"github.com/agbruneau/taugo/internal/calibration",
		"github.com/agbruneau/taugo/internal/bridge",
		"github.com/agbruneau/taugo/internal/app",
		"github.com/agbruneau/taugo/internal/errors",
		"github.com/agbruneau/taugo/internal/testutil",
	}},
	// ADR-0009: internal/errors is a transverse leaf — typed error families with
	// no taugo dependencies. It must not import any application-layer package.
	{from: "github.com/agbruneau/taugo/internal/errors", deny: []string{
		"github.com/agbruneau/taugo/internal/tau",
		"github.com/agbruneau/taugo/internal/orchestration",
		"github.com/agbruneau/taugo/internal/bridge",
		"github.com/agbruneau/taugo/internal/calibration",
		"github.com/agbruneau/taugo/internal/app",
		"github.com/agbruneau/taugo/internal/thresholds",
	}},
	// internal/testutil is a transverse test leaf: it builds tau value types only.
	// It must not reach into the orchestration/bridge/calibration/app layers.
	{from: "github.com/agbruneau/taugo/internal/testutil", deny: []string{
		"github.com/agbruneau/taugo/internal/orchestration",
		"github.com/agbruneau/taugo/internal/bridge",
		"github.com/agbruneau/taugo/internal/calibration",
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

// TestArchNoConcreteLLMInDomain walks every non-test .go file under
// internal/tau/ (including sub-packages) and internal/orchestration/,
// and fails if any file imports a concrete LLM provider SDK.
// Anti-pattern PRD §7.2 #6: concrete LLM SDK forbidden in the domain layer.
func TestArchNoConcreteLLMInDomain(t *testing.T) {
	t.Parallel()

	// Substrings whose presence in an import path signals a concrete LLM dependency.
	forbiddenSubstrings := []string{
		"anthropic",
		"openai",
		"mistralai",
		"mistral-go",
		"cohere",
		"google.golang.org/genai",
		"huggingface",
		"ollama",
		"replicate",
		"together",
		"anyscale",
		"groq",
	}

	intPkg, err := build.Default.Import("github.com/agbruneau/taugo/internal/calibration", ".", build.FindOnly)
	if err != nil {
		t.Skipf("module root not resolvable: %v", err)
	}
	internalRoot := filepath.Dir(intPkg.Dir)

	domainDirs := []string{
		filepath.Join(internalRoot, "tau"),
		filepath.Join(internalRoot, "orchestration"),
	}

	fset := token.NewFileSet()
	for _, domainDir := range domainDirs {
		if _, statErr := os.Stat(domainDir); os.IsNotExist(statErr) {
			continue
		}
		walkErr := filepath.WalkDir(domainDir, func(path string, d os.DirEntry, wErr error) error {
			if wErr != nil {
				return wErr
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			f, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
			if parseErr != nil {
				t.Errorf("parse error %s: %v", path, parseErr)
				return nil
			}
			rel, _ := filepath.Rel(internalRoot, path)
			for _, imp := range f.Imports {
				raw := strings.ToLower(strings.Trim(imp.Path.Value, `"`))
				for _, forbidden := range forbiddenSubstrings {
					if strings.Contains(raw, forbidden) {
						t.Errorf("anti-pattern PRD §7.2 #6 — concrete LLM SDK forbidden in domain layer: %s imports %s", rel, raw)
					}
				}
			}
			return nil
		})
		if walkErr != nil {
			t.Fatalf("WalkDir(%s) failed: %v", domainDir, walkErr)
		}
	}
}
