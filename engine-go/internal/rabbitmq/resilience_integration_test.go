package rabbitmq

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/streadway/amqp"
)

// Testes contra um RabbitMQ real. AMQP_TEST_URL=amqp://guest:guest@127.0.0.1:56720/ e, para os que derrubam o broker,
// AMQP_TEST_CONTAINER=<nome do contêiner docker>. Sem as variáveis, são pulados.

func integrationService(t *testing.T) *RabbitMQService {
	t.Helper()
	url := os.Getenv("AMQP_TEST_URL")
	if url == "" {
		t.Skip("AMQP_TEST_URL não definido")
	}
	t.Setenv("AMQP_URL", url)
	t.Setenv("ENGINE_OUTBOX_DIR", t.TempDir())
	svc := NewRabbitMQService()
	if err := svc.Connect(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Close() })
	return svc
}

func rawChannel(t *testing.T) *amqp.Channel {
	t.Helper()
	conn, err := amqp.Dial(os.Getenv("AMQP_TEST_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	ch, err := conn.Channel()
	if err != nil {
		t.Fatal(err)
	}
	return ch
}

func uniq(prefix string) string { return fmt.Sprintf("%s.%d", prefix, time.Now().UnixNano()) }

func waitFor(t *testing.T, d time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("tempo esgotado: %s", what)
}

func dockerCtl(t *testing.T, action string) {
	t.Helper()
	name := os.Getenv("AMQP_TEST_CONTAINER")
	if name == "" {
		t.Skip("AMQP_TEST_CONTAINER não definido")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker indisponível para derrubar o broker")
	}
	if out, err := exec.Command("docker", action, name).CombinedOutput(); err != nil {
		t.Fatalf("docker %s %s: %v: %s", action, name, err, out)
	}
}

func publishCmd(t *testing.T, ch *amqp.Channel, key, body string) {
	t.Helper()
	if err := ch.Publish(commandsExchange, key, false, false, amqp.Publishing{ContentType: "application/json", DeliveryMode: amqp.Persistent, Body: []byte(body)}); err != nil {
		t.Fatal(err)
	}
}

func queueDepth(t *testing.T, ch *amqp.Channel, name string) int {
	t.Helper()
	q, err := ch.QueueInspect(name)
	if err != nil {
		return -1
	}
	return q.Messages
}

// 1.7 — comando com falha recuperável é reentregue e, ao dar certo, é confirmado.
func TestComando_FalhaRecuperavelEReentregue(t *testing.T) {
	svc := integrationService(t)
	ch := rawChannel(t)
	q, key := uniq("t.retry"), uniq("wbot.t.1.message.send.text")
	var attempts atomic.Int32
	done := make(chan struct{})
	if err := svc.ConsumeCommands(q, []string{key}, func(d amqp.Delivery) error {
		if attempts.Add(1) < 2 {
			return fmt.Errorf("sessão ainda reconectando") // recuperável
		}
		close(done)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	publishCmd(t, ch, key, `{"n":1}`)
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatalf("o comando não foi reprocessado (tentativas=%d)", attempts.Load())
	}
	if got := attempts.Load(); got != 2 {
		t.Fatalf("tentativas = %d, esperado 2 (1 falha + 1 sucesso)", got)
	}
	waitFor(t, 5*time.Second, "fila vazia e DLQ vazia", func() bool {
		return queueDepth(t, ch, q) == 0 && queueDepth(t, ch, q+".dlq") == 0
	})
}

// 1.7 — comando inválido (irrecuperável) vai direto para a fila de falhas, sem repetir.
func TestComando_IrrecuperavelVaiParaADLQ(t *testing.T) {
	svc := integrationService(t)
	ch := rawChannel(t)
	q, key := uniq("t.dlq"), uniq("wbot.t.1.message.send.text")
	var attempts atomic.Int32
	if err := svc.ConsumeCommands(q, []string{key}, func(d amqp.Delivery) error {
		attempts.Add(1)
		return Unrecoverable(fmt.Errorf("JID inválido"))
	}); err != nil {
		t.Fatal(err)
	}
	publishCmd(t, ch, key, `{"n":1}`)
	waitFor(t, 10*time.Second, "mensagem na DLQ", func() bool { return queueDepth(t, ch, q+".dlq") == 1 })
	time.Sleep(300 * time.Millisecond)
	if got := attempts.Load(); got != 1 {
		t.Fatalf("tentativas = %d, esperado 1: irrecuperável não se repete", got)
	}
	if d := queueDepth(t, ch, q); d != 0 {
		t.Fatalf("a fila principal deveria estar vazia, tem %d", d)
	}
}

// 1.7 — falha recuperável que nunca passa esgota as tentativas e termina na DLQ (não fica em laço eterno).
func TestComando_RecuperavelEsgotadoVaiParaADLQ(t *testing.T) {
	svc := integrationService(t)
	ch := rawChannel(t)
	q, key := uniq("t.exhaust"), uniq("wbot.t.1.message.send.text")
	var attempts atomic.Int32
	if err := svc.ConsumeCommands(q, []string{key}, func(d amqp.Delivery) error {
		attempts.Add(1)
		return fmt.Errorf("sempre falha")
	}); err != nil {
		t.Fatal(err)
	}
	publishCmd(t, ch, key, `{"n":1}`)
	waitFor(t, 40*time.Second, "mensagem na DLQ após esgotar as tentativas", func() bool { return queueDepth(t, ch, q+".dlq") == 1 })
	if got := attempts.Load(); got != maxCommandRetries+1 {
		t.Fatalf("tentativas = %d, esperado %d (1 original + %d reentregas)", got, maxCommandRetries+1, maxCommandRetries)
	}
}

// 1.7 — pânico no tratador não derruba o engine: vira falha e o consumo continua.
func TestComando_PanicNoTratadorNaoDerrubaOConsumo(t *testing.T) {
	svc := integrationService(t)
	ch := rawChannel(t)
	q, key := uniq("t.panic"), uniq("wbot.t.1.message.send.text")
	var good atomic.Int32
	if err := svc.ConsumeCommands(q, []string{key}, func(d amqp.Delivery) error {
		if string(d.Body) == "boom" {
			panic("tratador quebrou")
		}
		good.Add(1)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	publishCmd(t, ch, key, "boom")
	publishCmd(t, ch, key, "ok-1")
	publishCmd(t, ch, key, "ok-2")
	waitFor(t, 15*time.Second, "as duas mensagens boas tratadas", func() bool { return good.Load() == 2 })
}

// 1.8 — o broker cai e volta: o consumo retoma sozinho, sem reiniciar o engine.
func TestConsumidor_RetomaDepoisDeQuedaDoBroker(t *testing.T) {
	if os.Getenv("AMQP_TEST_CONTAINER") == "" {
		t.Skip("AMQP_TEST_CONTAINER não definido")
	}
	svc := integrationService(t)
	q, key := uniq("t.reconnect"), uniq("wbot.t.1.message.send.text")
	var got sync.Map
	var n atomic.Int32
	if err := svc.ConsumeCommands(q, []string{key}, func(d amqp.Delivery) error {
		got.Store(string(d.Body), true)
		n.Add(1)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	publishCmd(t, rawChannel(t), key, "antes")
	waitFor(t, 10*time.Second, "mensagem antes da queda", func() bool { _, ok := got.Load("antes"); return ok })

	dockerCtl(t, "restart")
	waitFor(t, 90*time.Second, "broker de volta e consumidor retomado", func() bool {
		conn, err := amqp.Dial(os.Getenv("AMQP_TEST_URL"))
		if err != nil {
			return false
		}
		defer func() { _ = conn.Close() }()
		ch, err := conn.Channel()
		if err != nil {
			return false
		}
		if err := ch.Publish(commandsExchange, key, false, false, amqp.Publishing{DeliveryMode: amqp.Persistent, Body: []byte("depois")}); err != nil {
			return false
		}
		_, ok := got.Load("depois")
		return ok
	})
}

// 1.9 — com o broker fora, o evento é guardado e entregue depois, uma única vez.
func TestEvento_BrokerForaEntregaDepoisSemDuplicar(t *testing.T) {
	if os.Getenv("AMQP_TEST_CONTAINER") == "" {
		t.Skip("AMQP_TEST_CONTAINER não definido")
	}
	svc := integrationService(t)
	q := uniq("t.events")
	key := "wbot.t.1.message.received"
	ch := rawChannel(t)
	// Fila durável para sobreviver à parada do broker.
	if _, err := ch.QueueDeclare(q, true, false, false, false, nil); err != nil {
		t.Fatal(err)
	}
	if err := ch.QueueBind(q, key, eventsExchange, false, nil); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if c := tryChannel(); c != nil {
			_, _ = c.QueueDelete(q, false, false, false)
		}
	})

	dockerCtl(t, "stop")
	for i := 0; i < 3; i++ {
		if err := svc.PublishEvent(key, map[string]any{"i": i}); err != nil {
			t.Fatalf("PublishEvent com o broker fora deve gravar em disco, não falhar: %v", err)
		}
	}
	if svc.Pending() != 3 {
		t.Fatalf("pendentes = %d, esperado 3 (guardados em disco)", svc.Pending())
	}

	dockerCtl(t, "start")
	waitFor(t, 90*time.Second, "pendentes entregues", func() bool { return svc.Pending() == 0 })

	c := rawChannelWait(t)
	waitFor(t, 10*time.Second, "3 eventos na fila", func() bool { return queueDepth(t, c, q) == 3 })
	time.Sleep(500 * time.Millisecond)
	if d := queueDepth(t, c, q); d != 3 {
		t.Fatalf("eventos na fila = %d, esperado exatamente 3 (sem duplicar)", d)
	}
}

func tryChannel() *amqp.Channel {
	conn, err := amqp.Dial(os.Getenv("AMQP_TEST_URL"))
	if err != nil {
		return nil
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil
	}
	return ch
}

func rawChannelWait(t *testing.T) *amqp.Channel {
	t.Helper()
	var ch *amqp.Channel
	waitFor(t, 60*time.Second, "broker aceitando conexões", func() bool { ch = tryChannel(); return ch != nil })
	return ch
}

// 1.9 — queda do PROCESSO com evento pendente: o próximo processo o encontra no disco e entrega.
func TestEvento_PendenteSobreviveAQuedaDoProcesso(t *testing.T) {
	url := os.Getenv("AMQP_TEST_URL")
	if url == "" {
		t.Skip("AMQP_TEST_URL não definido")
	}
	dir := t.TempDir()
	t.Setenv("ENGINE_OUTBOX_DIR", dir)

	// Processo 1: grava o evento sem nunca conseguir conectar (URL de porta fechada) e "morre".
	t.Setenv("AMQP_URL", "amqp://guest:guest@127.0.0.1:1/")
	s1 := NewRabbitMQService()
	if err := s1.openOutbox(); err != nil {
		t.Fatal(err)
	}
	if err := s1.PublishEvent("wbot.t.1.message.received", map[string]any{"id": "perdido-se-nao-gravar"}); err != nil {
		t.Fatal(err)
	}
	_ = s1.out.Close() // sem Connect, sem confirmação: como um kill -9

	// Processo 2: conecta ao broker real e deve entregar o que achou no disco.
	t.Setenv("AMQP_URL", url)
	ch := rawChannel(t)
	q := uniq("t.survive")
	if _, err := ch.QueueDeclare(q, false, true, false, false, nil); err != nil {
		t.Fatal(err)
	}
	if err := ch.ExchangeDeclare(eventsExchange, "topic", true, false, false, false, nil); err != nil {
		t.Fatal(err)
	}
	if err := ch.QueueBind(q, "wbot.t.1.message.received", eventsExchange, false, nil); err != nil {
		t.Fatal(err)
	}
	s2 := NewRabbitMQService()
	if err := s2.Connect(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s2.Close() }()
	waitFor(t, 15*time.Second, "evento do processo anterior entregue", func() bool { return s2.Pending() == 0 && queueDepth(t, ch, q) == 1 })
}

// Migração: uma fila criada por um engine antigo (sem fila de falhas) com comandos parados dentro. O engine novo recria
// a fila com a DLQ e republica os comandos parados, sem perder nenhum.
func TestFilaAntigaSemDLQ_EhRecriadaSemPerderComandos(t *testing.T) {
	svc := integrationService(t)
	ch := rawChannel(t)
	q, key := uniq("t.legacy"), uniq("wbot.t.1.message.send.text")

	if _, err := ch.QueueDeclare(q, true, false, false, false, nil); err != nil { // como a versão antiga
		t.Fatal(err)
	}
	if err := ch.QueueBind(q, key, commandsExchange, false, nil); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		publishCmd(t, ch, key, fmt.Sprintf("parado-%d", i))
	}
	waitFor(t, 5*time.Second, "3 comandos parados na fila antiga", func() bool { return queueDepth(t, rawChannel(t), q) == 3 })

	var mu sync.Mutex
	seen := map[string]bool{}
	if err := svc.ConsumeCommands(q, []string{key}, func(d amqp.Delivery) error {
		mu.Lock()
		seen[string(d.Body)] = true
		mu.Unlock()
		return nil
	}); err != nil {
		t.Fatalf("o engine novo deveria recriar a fila antiga, não falhar: %v", err)
	}
	waitFor(t, 10*time.Second, "os 3 comandos parados processados", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(seen) == 3
	})
	if _, err := rawChannel(t).QueueInspect(q + ".dlq"); err != nil {
		t.Fatalf("a fila de falhas deveria existir: %v", err)
	}
}
