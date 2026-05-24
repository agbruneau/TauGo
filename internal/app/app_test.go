package app_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/taugo/internal/app"
	"github.com/agbruneau/taugo/internal/tau"
)

// TestApp_NewDispatcher_ChargeProfilParDefaut vérifie, de manière comportementale,
// que app.NewDispatcher() fournit un profil non nul au Dispatcher.
// Méthode : si le profil est nil, la garde d'expiration (étape 3) est désactivée
// et aucun Refus "périmé" ne peut survenir — quelle que soit l'horloge.
// En simulant une horloge postérieure à DateRevision (2026-12-01),
// un Dispatcher sans profil ne peut pas retourner Refus avec "périmé",
// alors qu'un Dispatcher avec profil le fera nécessairement.
// Ce test échouerait si app.NewDispatcher() n'injectait pas de profil.
func TestApp_NewDispatcher_ChargeProfilParDefaut(t *testing.T) {
	t.Parallel()
	if os.Getenv("TAUGO_LLM_BACKEND") == "real" {
		t.Skip("skipping: TAUGO_LLM_BACKEND=real explicitly set")
	}

	// 2027-01-02 est postérieur à DateRevision (2026-12-01) du DefaultProfile.
	futureDate := time.Date(2027, 1, 2, 0, 0, 0, 0, time.UTC)
	d := app.NewDispatcher().WithClock(func() time.Time { return futureDate })

	// L'échange doit passer les étapes 1 et 2 pour atteindre l'étape 3 (péremption).
	// Étape 1 : Inside() = true requiert les 4 conditions — DiscoveryMode!=Static,
	//           HumanInLoop=false, DelegationDepth>0.
	// Étape 2 : authScore < AuthBlock(0.85) — avec DelegationDepth=1, Organization="org-test",
	//           HumanInLoop=false, DiscoveryMode=DynamicMCP → score ≈ 0.5625.
	x := tau.Exchange{
		ID:                "test-profil-defaut",
		IntentDescription: "probe",
		DiscoveredAt:      time.Now(),
		Initiator: tau.Principal{
			ID:              "agent-test",
			HumanInLoop:     false,
			Organization:    "org-test",
			DelegationDepth: 1,
		},
		Target: tau.Capability{
			ID:            "svc-test",
			DiscoveryMode: tau.DynamicMCP,
			ContractURI:   "https://api.example.com/v1",
		},
	}

	dec, err := d.Decide(context.Background(), x)
	if err != nil {
		t.Fatalf("Decide a retourné une erreur inattendue : %v", err)
	}
	if dec.Regime != tau.Refus {
		t.Fatalf("régime attendu Refus (profil périmé), obtenu %v — profil par défaut absent ?", dec.Regime)
	}
	if !strings.Contains(dec.Diagnostic, "périmé") {
		t.Fatalf("diagnostic attendu contenant \"périmé\", obtenu %q", dec.Diagnostic)
	}
}

// TestApp_NewDispatcher_PeremptionDetecteeAvecHorlogeSimulee vérifie que
// app.NewDispatcher(), une fois l'horloge simulée au-delà de DateRevision,
// retourne Refus avec le diagnostic "profil périmé — veille requise".
// Ce test couvre la correction P0-02 (anti-patron #3, PRD §7.3 cas 4).
func TestApp_NewDispatcher_PeremptionDetecteeAvecHorlogeSimulee(t *testing.T) {
	t.Parallel()
	if os.Getenv("TAUGO_LLM_BACKEND") == "real" {
		t.Skip("skipping: TAUGO_LLM_BACKEND=real explicitly set")
	}

	// Horloge fixe strictement postérieure à DateRevision du DefaultProfile (2026-12-01).
	expiredClock := func() time.Time { return time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC) }
	d := app.NewDispatcher().WithClock(expiredClock)

	// Même échange que TestApp_NewDispatcher_ChargeProfilParDefaut : passe étapes 1 et 2,
	// bloqué à l'étape 3 par la péremption du profil.
	x := tau.Exchange{
		ID:                "test-peremption",
		IntentDescription: "péremption probe",
		DiscoveredAt:      time.Now(),
		Initiator: tau.Principal{
			ID:              "agent-test",
			HumanInLoop:     false,
			Organization:    "org-test",
			DelegationDepth: 1,
		},
		Target: tau.Capability{
			ID:            "svc-test",
			DiscoveryMode: tau.DynamicMCP,
			ContractURI:   "https://api.example.com/v1",
		},
	}

	dec, err := d.Decide(context.Background(), x)
	if err != nil {
		t.Fatalf("Decide a retourné une erreur : %v", err)
	}
	if dec.Regime != tau.Refus {
		t.Fatalf("attendu Refus pour profil périmé, obtenu régime=%v", dec.Regime)
	}
	const wantDiag = "profil périmé — veille requise"
	if dec.Diagnostic != wantDiag {
		t.Fatalf("diagnostic attendu %q, obtenu %q", wantDiag, dec.Diagnostic)
	}
}

// TestSelectLLM_RealBackend_Panics asserts that NewDispatcher panics when
// TAUGO_LLM_BACKEND=real. The real backend is not implemented until M5+;
// the panic is a deliberate sentinel that prevents silent CI regressions
// (see PRD §15.4 and selectLLM inline comment).
func TestSelectLLM_RealBackend_Panics(t *testing.T) {
	t.Setenv("TAUGO_LLM_BACKEND", "real")
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic from selectLLM with TAUGO_LLM_BACKEND=real, got none")
		}
	}()
	app.NewDispatcher() // must panic
}

// TestDefaultLLMIsStub — guards the anti-patron: in CI / default mode, no
// external LLM service may be called. The Dispatcher must always use
// llm.Stub unless TAUGO_LLM_BACKEND=real is set explicitly.
//
// Behavioral verification (M2): the stub is deterministic — two calls with
// the same exchange must produce the same TauScore. A real LLM would be
// non-deterministic and would fail this property across calls.
// The exchange is constructed to be inside the M2 frontier (DiscoveryMode!=Static,
// HumanInLoop=false, DelegationDepth>0) so Decide actually reaches the
// LLM-backed D-SENS composite rather than bailing out at the frontier.
func TestDefaultLLMIsStub(t *testing.T) {
	t.Parallel()
	if os.Getenv("TAUGO_LLM_BACKEND") == "real" {
		t.Skip("skipping: TAUGO_LLM_BACKEND=real explicitly set")
	}

	d := app.NewDispatcher()
	if d == nil {
		t.Fatal("NewDispatcher returned nil")
	}

	x := tau.Exchange{
		ID:                "witness",
		IntentDescription: "test-default-stub-witness",
		DiscoveredAt:      time.Now(),
		Initiator: tau.Principal{
			ID:              "agent-test",
			HumanInLoop:     false,
			Organization:    "org-test",
			DelegationDepth: 1,
		},
		Target: tau.Capability{
			ID:            "svc-test",
			DiscoveryMode: tau.DynamicMCP,
			ContractURI:   "https://api.example.com/v1",
		},
	}

	dec1, err := d.Decide(context.Background(), x)
	if err != nil {
		t.Fatalf("first Decide failed: %v", err)
	}
	dec2, err := d.Decide(context.Background(), x)
	if err != nil {
		t.Fatalf("second Decide failed: %v", err)
	}

	// Stub is deterministic: same exchange must produce the same TauScore.
	if dec1.Trace.TauScore != dec2.Trace.TauScore {
		t.Fatalf("TauScore not deterministic: call1=%f call2=%f (expected same stub output)",
			dec1.Trace.TauScore, dec2.Trace.TauScore)
	}
}
