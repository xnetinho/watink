package controllers

import "testing"

func TestValidatePollVote(t *testing.T) {
	poll := []string{"A", "B", "C"}
	cases := []struct {
		name       string
		chosen     []string
		selectable int
		wantErr    bool
	}{
		{"única válida", []string{"B"}, 1, false},
		{"remover voto", nil, 1, false},
		{"múltipla dentro do limite", []string{"A", "C"}, 2, false},
		{"acima do limite", []string{"A", "B", "C"}, 2, true},
		{"única com duas", []string{"A", "B"}, 1, true},
		{"opção que não existe", []string{"Z"}, 1, true},
		{"opção repetida", []string{"A", "A"}, 3, true},
		{"sem limite (0)", []string{"A", "B", "C"}, 0, false},
	}
	for _, c := range cases {
		if got := validatePollVote(poll, c.chosen, c.selectable) != ""; got != c.wantErr {
			t.Errorf("%s: erro=%v, esperado %v", c.name, got, c.wantErr)
		}
	}
}

func TestPollOptions(t *testing.T) {
	o, n := pollOptions(`{"rich":{"options":["A","B"],"selectable":2}}`)
	if len(o) != 2 || n != 2 {
		t.Fatalf("%v %d", o, n)
	}
	if o, _ := pollOptions(`{}`); len(o) != 0 {
		t.Fatal("enquete antiga sem opções estruturadas")
	}
}
