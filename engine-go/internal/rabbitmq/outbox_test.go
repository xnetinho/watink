package rabbitmq

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOutbox_PendenteSobreviveAoRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.outbox")
	o, err := openOutbox(path)
	if err != nil {
		t.Fatal(err)
	}
	s1, _ := o.Add("wbot.t.1.message.received", []byte(`{"a":1}`))
	s2, _ := o.Add("wbot.t.1.message.ack", []byte(`{"b":2}`))
	s3, _ := o.Add("wbot.t.1.message.received", []byte(`{"c":3}`))
	if err := o.Confirm(s2); err != nil {
		t.Fatal(err)
	}
	_ = o.Close() // simula a queda do processo

	o2, err := openOutbox(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = o2.Close() }()
	got := o2.Pending()
	if len(got) != 2 || got[0].Seq != s1 || got[1].Seq != s3 {
		t.Fatalf("pendentes após restart = %+v, esperado seq %d e %d em ordem", got, s1, s3)
	}
	s4, _ := o2.Add("x", []byte(`{}`))
	if s4 <= s3 {
		t.Fatalf("a sequência não pode reaproveitar números: %d depois de %d", s4, s3)
	}
}

func TestOutbox_LinhaTruncadaNaoQuebraARecarga(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.outbox")
	o, _ := openOutbox(path)
	_, _ = o.Add("k", []byte(`{"ok":true}`))
	_ = o.Close()

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(`{"op":"add","e":{"seq":9,"rk":"k","bo`) // queda no meio da escrita
	_ = f.Close()

	o2, err := openOutbox(path)
	if err != nil {
		t.Fatalf("recarga falhou: %v", err)
	}
	defer func() { _ = o2.Close() }()
	if o2.Len() != 1 {
		t.Fatalf("pendentes = %d, esperado 1 (a linha truncada é ignorada)", o2.Len())
	}
}

func TestOutbox_CompactaSemPerderOPendente(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.outbox")
	o, _ := openOutbox(path)
	keep, _ := o.Add("keep", []byte(`{"keep":true}`))
	for i := 0; i < outboxCompactAfter+50; i++ {
		s, _ := o.Add("tmp", []byte(`{}`))
		if err := o.Confirm(s); err != nil {
			t.Fatal(err)
		}
	}
	st, _ := os.Stat(path)
	if st.Size() > 100*1024 {
		t.Fatalf("arquivo não foi compactado: %d bytes", st.Size())
	}
	_ = o.Close()

	o2, _ := openOutbox(path)
	defer func() { _ = o2.Close() }()
	got := o2.Pending()
	if len(got) != 1 || got[0].Seq != keep || string(got[0].Body) != `{"keep":true}` {
		t.Fatalf("o pendente se perdeu na compactação: %+v", got)
	}
}
