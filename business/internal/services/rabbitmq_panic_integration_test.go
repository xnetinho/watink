//go:build integration

package services

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/pkg/safego"
	"github.com/streadway/amqp"
)

// Um pânico no tratador de uma entrega não pode derrubar o processo nem matar o consumidor: a entrega segue para o
// retry/DLQ e as mensagens seguintes continuam sendo tratadas.
func TestRabbitMQConsumer_PanicNoTratadorNaoDerrubaOConsumo(t *testing.T) {
	svc := NewRabbitMQProvider(rabbitMQURL(t))
	if err := svc.Connect(); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer svc.Close()

	queue := fmt.Sprintf("test.panic.%d", time.Now().UnixNano())
	key := queue + ".key"
	var handled atomic.Int32
	before := safego.Panics()

	err := svc.ConsumeEvents(queue, []string{key}, func(d amqp.Delivery) error {
		if string(d.Body) == "boom" {
			panic("tratador quebrou")
		}
		handled.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("ConsumeEvents: %v", err)
	}
	defer func() { _ = svc.channel.Close() }()

	publish := func(body string) {
		t.Helper()
		if err := svc.channel.Publish("wbot.events", key, false, false, amqp.Publishing{ContentType: "text/plain", Body: []byte(body)}); err != nil {
			t.Fatalf("publish %q: %v", body, err)
		}
	}
	publish("boom")
	publish("ok-1")
	publish("ok-2")

	deadline := time.Now().Add(10 * time.Second)
	for handled.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if got := handled.Load(); got != 2 {
		t.Fatalf("mensagens boas tratadas = %d, esperado 2 (o consumo parou após o pânico)", got)
	}
	if safego.Panics() <= before {
		t.Fatal("o pânico deveria ter sido contido e contado por safego")
	}
}
