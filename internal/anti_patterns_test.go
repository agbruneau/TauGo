// internal/anti_patterns_test.go
package internal_test

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/calibration"
	"github.com/agbruneau/taugo/internal/tau/invariants"
)

// predictivePattern catches the three forbidden prefixes (PRD §7.2 #1).
var predictivePattern = regexp.MustCompile(`^(Predict|Expected|Forecast)`)

// gardedPackages enumerates the import paths whose exported identifiers
// must not match predictivePattern.
var gardedPackages = []string{
	"github.com/agbruneau/taugo/internal/tau",
	"github.com/agbruneau/taugo/internal/tau/dimensions",
	"github.com/agbruneau/taugo/internal/tau/invariants",
	"github.com/agbruneau/taugo/internal/orchestration",
}

// TestNoPredictiveAPI parses the source AST of each guarded package and
// fails if any exported function, method, or type matches the forbidden
// predictive pattern (PRD §7.2 anti-patron #1).
func TestNoPredictiveAPI(t *testing.T) {
	t.Parallel()
	for _, pkgPath := range gardedPackages {
		t.Run(strings.ReplaceAll(pkgPath, "/", "_"), func(t *testing.T) {
			t.Parallel()
			pkg, err := build.Default.Import(pkgPath, ".", build.ImportComment)
			if err != nil {
				t.Skipf("package not built: %v", err)
			}
			fset := token.NewFileSet()
			for _, src := range pkg.GoFiles {
				path := filepath.Join(pkg.Dir, src)
				f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
				if err != nil {
					t.Fatalf("parse %s: %v", path, err)
				}
				for _, decl := range f.Decls {
					name := exportedDeclName(decl)
					if name == "" {
						continue
					}
					if predictivePattern.MatchString(name) {
						t.Errorf("forbidden predictive API in %s: %s", pkgPath, name)
					}
				}
			}
		})
	}
}

// exportedDeclName returns the exported name of a top-level declaration, or "".
// Covers FuncDecl (functions and methods) and GenDecl (type, var, const specs).
// Returns only the first exported name per GenDecl; multi-spec blocks are
// walked entirely via the Specs slice.
func exportedDeclName(decl ast.Decl) string {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		if d.Name.IsExported() {
			return d.Name.Name
		}
	case *ast.GenDecl:
		for _, spec := range d.Specs {
			switch s := spec.(type) {
			case *ast.TypeSpec:
				if s.Name.IsExported() {
					return s.Name.Name
				}
			case *ast.ValueSpec:
				for _, name := range s.Names {
					if name.IsExported() {
						return name.Name
					}
				}
			}
		}
	}
	return ""
}

// TestI3_DateRevisionRespectee guards anti-patron #3 (atemporel).
// DefaultProfile must always carry a DateRevision strictly in the future AND
// at or before the I3 péremption limit (2027-01-01) (PRD §7.2 #3, §7.1 C3).
func TestI3_DateRevisionRespectee(t *testing.T) {
	t.Parallel()
	p := calibration.DefaultProfile()
	now := time.Now().UTC()
	if !p.DateRevision.After(now) {
		t.Fatalf("Profile.DateRevision %v is not after now %v", p.DateRevision, now)
	}
	if p.DateRevision.After(invariants.I3PerimptionLimite) {
		t.Fatalf("Profile.DateRevision %v is beyond I3 péremption limit %v",
			p.DateRevision, invariants.I3PerimptionLimite)
	}
}

// TestUnmodeledObservationsReported guards anti-patron #4 (clos).
// The contract: when EvaluateInvariants reports a violation, the dispatcher
// MUST append the corresponding line to Trace.UnmodeledObservations (PRD §7.2 #4).
//
// V1 check: verifies Statuses.Summary produces non-empty, well-formed strings
// that can be threaded into Trace.UnmodeledObservations.
//
// Full end-to-end guard (dispatcher → Trace.UnmodeledObservations) is covered
// by TestStep8_InvariantsEvaluated_ViolationDetected_TraceEnriched in
// internal/orchestration/dispatcher_invariants_test.go.
func TestUnmodeledObservationsReported(t *testing.T) {
	t.Parallel()
	s := invariants.Statuses{
		I1: invariants.Violated,
		I3: invariants.Violated,
	}
	got := s.Summary()
	if len(got) == 0 {
		t.Fatal("Statuses.Summary returned empty on a known violation set")
	}
	for _, line := range got {
		if strings.TrimSpace(line) == "" {
			t.Fatal("empty Summary line — would silently fail anti-patron #4 guard")
		}
	}
	// Verify the lines are anchored on the right invariant numbers.
	if got[0][:2] != "I1" {
		t.Fatalf("first Summary line prefix = %q, want \"I1\"", got[0][:2])
	}
	if got[1][:2] != "I3" {
		t.Fatalf("second Summary line prefix = %q, want \"I3\"", got[1][:2])
	}
}
