package rabbitmq

import (
	"path/filepath"
	"testing"
)

// Custo por evento da saída durável (gravar + fsync + confirmar): é o que cada mensagem recebida paga no engine.
func BenchmarkOutbox_AddConfirm(b *testing.B) {
	o, _ := openOutbox(filepath.Join(b.TempDir(), "events.outbox"))
	defer func() { _ = o.Close() }()
	body := []byte(`{"sessionId":"1","message":{"id":"3EB0AAAA","body":"olá"}}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s, _ := o.Add("wbot.t.1.message.received", body)
		_ = o.Confirm(s)
	}
}
