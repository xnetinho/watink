package safego

import (
	"errors"
	"testing"
	"time"
)

func TestRun_ConverteOPanicEmErro(t *testing.T) {
	before := Panics()
	err := Run("teste", func() error { panic("falhou") })
	var pe *PanicError
	if !errors.As(err, &pe) || pe.Name != "teste" {
		t.Fatalf("esperava PanicError, veio %v", err)
	}
	if Panics() != before+1 {
		t.Fatal("o pânico deveria ser contado")
	}
}

func TestRun_NaoMexeNoErroNormal(t *testing.T) {
	want := errors.New("erro comum")
	if got := Run("teste", func() error { return want }); !errors.Is(got, want) {
		t.Fatalf("erro alterado: %v", got)
	}
	if Run("teste", func() error { return nil }) != nil {
		t.Fatal("esperava nil")
	}
}

func TestGo_PanicNaoDerrubaOProcesso(t *testing.T) {
	before := Panics()
	done := make(chan struct{})
	Go("teste", func() {
		defer close(done)
		panic("solto")
	})
	<-done
	deadline := time.Now().Add(2 * time.Second)
	for Panics() == before && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if Panics() == before {
		t.Fatal("o pânico deveria ter sido contido")
	}
}
