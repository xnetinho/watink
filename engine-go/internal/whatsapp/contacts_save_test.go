package whatsapp

import (
	"testing"

	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/types"
)

func TestBuildContactSavePatch_FormatoDoWhatsAppWeb(t *testing.T) {
	jid := types.NewJID("5583999990000", types.DefaultUserServer)
	p := buildContactSavePatch(jid, "Maria da Silva")

	if p.Type != appstate.WAPatchCriticalUnblockLow {
		t.Fatalf("a agenda vive no patch critical_unblock_low, veio %q", p.Type)
	}
	if len(p.Mutations) != 1 {
		t.Fatalf("uma mutação por contato, veio %d", len(p.Mutations))
	}
	m := p.Mutations[0]
	if len(m.Index) != 2 || m.Index[0] != "contact" || m.Index[1] != jid.String() {
		t.Fatalf("índice = %v", m.Index)
	}
	a := m.Value.GetContactAction()
	if a.GetFullName() != "Maria da Silva" || a.GetFirstName() != "Maria" || !a.GetSaveOnPrimaryAddressbook() {
		t.Fatalf("ContactAction = %v", a)
	}
}

func TestBuildContactSavePatch_NomeUnicoEhOPrimeiroNome(t *testing.T) {
	a := buildContactSavePatch(types.NewJID("5583999990000", types.DefaultUserServer), "  Cher ").Mutations[0].Value.GetContactAction()
	if a.GetFullName() != "Cher" || a.GetFirstName() != "Cher" {
		t.Fatalf("%v", a)
	}
}
