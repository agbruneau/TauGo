// Quickstart TauGo — trois échanges, trois régimes possibles.
//
// Objectif pédagogique : montrer comment Decide(...) classe un échange
// en Deterministe / Probabiliste / Refus, et comment lire la Trace.
//
// Exécution depuis la racine du repo :
//
//	go run ./examples/quickstart
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/agbruneau/taugo/internal/app"
	"github.com/agbruneau/taugo/internal/tau"
)

func main() {
	// app.NewDispatcher() charge un profil de calibration par défaut.
	// Garde de péremption active : si la DateRevision du profil est dépassée,
	// toute décision devient Refus (anti-patron #3, P0-02).
	dispatcher := app.NewDispatcher()

	cases := []struct {
		label string
		x     tau.Exchange
	}{
		// CAS 1 : appel humain-piloté, cible statique → hors frontière τ.
		// Aucune des 4 conditions classiques n'est violée → Refus attendu
		// avec diagnostic "frontière franchie".
		{
			label: "1. Humain dans la boucle, cible statique (hors frontière)",
			x: tau.Exchange{
				ID:                "exchange-statique",
				IntentDescription: "lookup utilisateur par ID",
				Initiator: tau.Principal{
					ID:              "operateur-humain",
					HumanInLoop:     true, // humain présent
					DelegationDepth: 0,    // mandat direct
				},
				Target: tau.Capability{
					ID:            "user-service",
					DiscoveryMode: tau.Static, // contrat connu au design time
				},
			},
		},

		// CAS 2 : agent autonome, découverte MCP runtime, délégation profonde.
		// Les 4 conditions sont violées → dans la frontière τ.
		// Sans AttestationInstitutionnelle, D-AUTORITÉ devrait être élevée
		// → potentiellement Refus pour verrou ontologique.
		{
			label: "2. Agent autonome, MCP dynamique, sans attestation",
			x: tau.Exchange{
				ID:                "exchange-autonome",
				IntentDescription: "approuver une demande de prêt 75k$",
				Initiator: tau.Principal{
					ID:              "agent-back-office",
					HumanInLoop:     false,
					DelegationDepth: 3, // 3 crans de délégation
				},
				Target: tau.Capability{
					ID:            "loan-approval-api",
					DiscoveryMode: tau.DynamicMCP,
				},
				// PAS d'AttestationInstitutionnelle → I3 va déclencher
			},
		},

		// CAS 3 : même chose qu'au CAS 2, MAIS avec une attestation institutionnelle.
		// Le verrou ontologique D-AUTORITÉ se relâche → décision contextuelle
		// (Deterministe ou Probabiliste selon les scores).
		{
			label: "3. Idem CAS 2 + attestation institutionnelle",
			x: tau.Exchange{
				ID:                "exchange-attestee",
				IntentDescription: "approuver une demande de prêt 75k$",
				Initiator: tau.Principal{
					ID:              "agent-back-office",
					HumanInLoop:     false,
					DelegationDepth: 3,
				},
				Target: tau.Capability{
					ID:            "loan-approval-api",
					DiscoveryMode: tau.DynamicMCP,
				},
				AttestationInstitutionnelle: &tau.Attestation{
					Emetteur:  "Conseil de credit (cas synthetique)",
					Reference: "POL-CREDIT-2026-014",
					Marqueur:  "delegation-formelle-niveau-3",
				},
			},
		},
	}

	for _, c := range cases {
		fmt.Println("================================================================================")
		fmt.Println(c.label)
		fmt.Println("================================================================================")

		dec, err := dispatcher.Decide(context.Background(), c.x)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERREUR : %v\n", err)
			continue
		}

		// Régime + diagnostic (si Refus).
		fmt.Printf("Regime     : %s\n", dec.Regime)
		if dec.Diagnostic != "" {
			fmt.Printf("Diagnostic : %s\n", dec.Diagnostic)
		}
		fmt.Printf("Profil     : %s (revision %s)\n",
			dec.ProfileVersion, dec.DateRevision.Format("2006-01-02"))

		// Scores ventilés (peuvent être nil si early-exit Refus).
		fmt.Printf("tau_score  : %.4f\n", dec.Trace.TauScore)
		printScore("D-SENS     ", dec.Trace.DSens)
		printScore("D-AUTORITE ", dec.Trace.DAuthority)
		printScore("D-INVARIANT", dec.Trace.DInvariant)

		// Frontière : les 4 conditions classiques.
		fmt.Printf("Frontiere  : UnivOuvert=%t  CompoVar=%t  PairProba=%t  CoutNonBorne=%t  =>  Inside=%t\n",
			dec.Trace.Frontier.UniversOuvert,
			dec.Trace.Frontier.CompositionVariable,
			dec.Trace.Frontier.PairProbabiliste,
			dec.Trace.Frontier.CoutNonBorne,
			dec.Trace.Frontier.Inside())

		// Observations non modélisées (si présentes).
		if len(dec.Trace.UnmodeledObservations) > 0 {
			fmt.Printf("Non modelise : %v\n", dec.Trace.UnmodeledObservations)
		}

		// Durée totale.
		fmt.Printf("Duree      : %d ns\n", dec.Trace.DurationNs)

		// JSON brut indenté pour la dernière décision uniquement.
		if c.x.ID == "exchange-attestee" {
			fmt.Println()
			fmt.Println("--- Trace JSON complete (CAS 3) ---")
			b, _ := json.MarshalIndent(dec, "", "  ")
			fmt.Println(string(b))
		}

		fmt.Println()
	}
}

func printScore(label string, s *tau.Score) {
	if s == nil {
		fmt.Printf("%s: <non calcule>\n", label)
		return
	}
	fmt.Printf("%s: %.4f", label, s.Value)
	if len(s.Probes) > 0 {
		fmt.Printf("  (sondes: %v)", s.Probes)
	}
	fmt.Println()
}
