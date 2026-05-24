package app

import (
	"strings"
	"testing"

	customerrors "github.com/agbruneau/taugo/internal/errors"
)

// TestSelectLLM_StubProvider vérifie que "stub" retourne un client non-nil sans erreur.
func TestSelectLLM_StubProvider(t *testing.T) {
	t.Parallel()
	client, err := selectLLM("stub")
	if err != nil {
		t.Fatalf("selectLLM(\"stub\") a retourné une erreur inattendue : %v", err)
	}
	if client == nil {
		t.Fatal("selectLLM(\"stub\") a retourné un client nil")
	}
}

// TestSelectLLM_EmptyProviderIsStub vérifie que la chaîne vide (défaut) retourne le stub.
func TestSelectLLM_EmptyProviderIsStub(t *testing.T) {
	t.Parallel()
	client, err := selectLLM("")
	if err != nil {
		t.Fatalf("selectLLM(\"\") a retourné une erreur inattendue : %v", err)
	}
	if client == nil {
		t.Fatal("selectLLM(\"\") a retourné un client nil")
	}
}

// TestSelectLLM_ProviderInconnuRetourneErreur vérifie que "foobar" retourne
// un *DispatchError dont Detail contient "inconnu".
func TestSelectLLM_ProviderInconnuRetourneErreur(t *testing.T) {
	t.Parallel()
	client, err := selectLLM("foobar")
	if err == nil {
		t.Fatal("selectLLM(\"foobar\") a retourné nil, erreur attendue")
	}
	if client != nil {
		t.Fatal("client doit être nil en cas d'erreur")
	}
	var de *customerrors.DispatchError
	if !asDispatchError(err, &de) {
		t.Fatalf("erreur n'est pas *DispatchError : %T — %v", err, err)
	}
	if !strings.Contains(de.Detail, "inconnu") {
		t.Fatalf("Detail attendu contenant \"inconnu\", obtenu %q", de.Detail)
	}
}

// TestSelectLLM_RealNotImplemented vérifie que "real" retourne une erreur typée non-nil.
func TestSelectLLM_RealNotImplemented(t *testing.T) {
	t.Parallel()
	client, err := selectLLM("real")
	if err == nil {
		t.Fatal("selectLLM(\"real\") a retourné nil, erreur attendue")
	}
	if client != nil {
		t.Fatal("client doit être nil en cas d'erreur")
	}
	var de *customerrors.DispatchError
	if !asDispatchError(err, &de) {
		t.Fatalf("erreur n'est pas *DispatchError : %T — %v", err, err)
	}
}

// asDispatchError est un helper minimal pour éviter l'import du package errors/As
// dans un fichier de test interne — il appelle errors.As directement.
func asDispatchError(err error, target **customerrors.DispatchError) bool {
	// errors.As est disponible dans le package standard errors.
	// On applique une assertion de type directe car selectLLM retourne *DispatchError directement.
	de, ok := err.(*customerrors.DispatchError)
	if ok {
		*target = de
	}
	return ok
}
