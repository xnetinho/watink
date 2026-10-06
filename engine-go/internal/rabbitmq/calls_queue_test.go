package rabbitmq

import (
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/streadway/amqp"
)

// Integração com um RabbitMQ real: AMQP_TEST_URL=amqp://guest:guest@localhost:56720/
// Prova que o consumidor das chamadas não espera o consumidor de mensagens.
func TestCallsConsumerRunsWhileMessageConsumerIsBlocked(t *testing.T) {
	url := os.Getenv("AMQP_TEST_URL")
	if url == "" {
		t.Skip("AMQP_TEST_URL não definido")
	}
	t.Setenv("AMQP_URL", url)
	svc := NewRabbitMQService()
	if err := svc.Connect(); err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	suffix := time.Now().Format("150405.000000")
	msgQ, callQ := "t.msgs."+suffix, "t.calls."+suffix
	release := make(chan struct{})
	var msgStarted, callDone int32
	callGot := make(chan string, 4)

	if err := svc.ConsumeCommands(msgQ, []string{"wbot.t.1.message.send.text"}, func(d amqp.Delivery) {
		atomic.AddInt32(&msgStarted, 1)
		<-release
		d.Ack(false)
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.ConsumeCommandsConcurrent(callQ, []string{"wbot.*.*.call.*"}, func(d amqp.Delivery) {
		atomic.AddInt32(&callDone, 1)
		callGot <- d.RoutingKey
		d.Ack(false)
	}); err != nil {
		t.Fatal(err)
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ch, _ := conn.Channel()
	pub := func(key string) {
		if err := ch.Publish("wbot.commands", key, false, false, amqp.Publishing{ContentType: "application/json", Body: []byte(`{}`)}); err != nil {
			t.Fatal(err)
		}
	}

	pub("wbot.t.1.message.send.text")
	for i := 0; atomic.LoadInt32(&msgStarted) == 0; i++ {
		if i > 200 {
			t.Fatal("o consumidor de mensagens não começou")
		}
		time.Sleep(10 * time.Millisecond)
	}

	pub("wbot.t.1.call.accept")
	pub("wbot.t.1.call.end")
	for _, want := range []int{1, 2} {
		select {
		case <-callGot:
		case <-time.After(2 * time.Second):
			t.Fatalf("comando de chamada %d esperou o consumidor de mensagens bloqueado", want)
		}
	}
	close(release)
}

func TestCallRoutingKeysAreNotInMessageQueueBinding(t *testing.T) {
	url := os.Getenv("AMQP_TEST_URL")
	if url == "" {
		t.Skip("AMQP_TEST_URL não definido")
	}
	t.Setenv("AMQP_URL", url)
	svc := NewRabbitMQService()
	if err := svc.Connect(); err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	suffix := time.Now().Format("150405.000000")
	var onMsgQueue int32
	if err := svc.ConsumeCommands("t.only-msgs."+suffix, []string{"wbot.*.*.message.send.text"}, func(d amqp.Delivery) {
		atomic.AddInt32(&onMsgQueue, 1)
		d.Ack(false)
	}); err != nil {
		t.Fatal(err)
	}
	conn, _ := amqp.Dial(url)
	defer conn.Close()
	ch, _ := conn.Channel()
	_ = ch.Publish("wbot.commands", "wbot.t.1.call.accept", false, false, amqp.Publishing{Body: []byte(`{}`)})
	time.Sleep(300 * time.Millisecond)
	if atomic.LoadInt32(&onMsgQueue) != 0 {
		t.Fatal("call.* não pode ser entregue na fila de mensagens")
	}
}

// Um comando de chamada lento (ex.: call.accept esperando o ack do WhatsApp) não
// pode atrasar o seguinte (ex.: call.end): cada Delivery roda em goroutine própria.
func TestCallsConsumerDispatchesEachCommandConcurrently(t *testing.T) {
	url := os.Getenv("AMQP_TEST_URL")
	if url == "" {
		t.Skip("AMQP_TEST_URL não definido")
	}
	t.Setenv("AMQP_URL", url)
	svc := NewRabbitMQService()
	if err := svc.Connect(); err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	release := make(chan struct{})
	endDone := make(chan struct{}, 1)
	q := "t.calls.conc." + time.Now().Format("150405.000000")
	if err := svc.ConsumeCommandsConcurrent(q, []string{"wbot.*.*.call.*"}, func(d amqp.Delivery) {
		if d.RoutingKey == "wbot.t.1.call.accept" {
			<-release
		} else {
			endDone <- struct{}{}
		}
		d.Ack(false)
	}); err != nil {
		t.Fatal(err)
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ch, _ := conn.Channel()
	for _, k := range []string{"wbot.t.1.call.accept", "wbot.t.1.call.end"} {
		if err := ch.Publish("wbot.commands", k, false, false, amqp.Publishing{Body: []byte(`{}`)}); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case <-endDone:
	case <-time.After(2 * time.Second):
		t.Fatal("call.end esperou o call.accept lento: o consumidor é serial")
	}
	close(release)
}
