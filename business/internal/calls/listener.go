package calls

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
	amqp "github.com/streadway/amqp"
)

// callsQueue é a fila DEDICADA aos eventos de chamada do engine. Fica separada de
// api.events.process.go de propósito: um tenant com muito tráfego de mensagens não
// pode atrasar um toque de chamada atrás de centenas de eventos de outras
// conversas. Dentro da fila a ordem é preservada (incoming antes de ended).
const callsQueue = "api.events.calls.go"

// CallEventRoutingKeys são os eventos de chamada publicados pelo engine.
var CallEventRoutingKeys = []string{"wbot.*.*.call.*"}

type envelope struct {
	TenantID string          `json:"tenantId"`
	Type     string          `json:"type"`
	Payload  json.RawMessage `json:"payload"`
}

// EventConsumer é o que o listener precisa do RabbitMQ.
type EventConsumer interface {
	ConsumeEvents(queueName string, routingKeys []string, handler func(amqp.Delivery) error) error
}

// Start liga o consumidor dedicado de eventos de chamada.
func (s *Service) Start(c EventConsumer) error {
	return c.ConsumeEvents(callsQueue, CallEventRoutingKeys, func(d amqp.Delivery) error {
		return s.Dispatch(context.Background(), d.Body)
	})
}

// Dispatch interpreta um envelope de evento e o entrega ao tratador certo.
// Um tipo desconhecido é ignorado (não é erro: evita DLQ por evento novo).
func (s *Service) Dispatch(ctx context.Context, body []byte) error {
	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("envelope de chamada inválido: %w", err)
	}
	tenantID, err := uuid.Parse(env.TenantID)
	if err != nil {
		return fmt.Errorf("tenantId inválido %q: %w", env.TenantID, err)
	}
	switch env.Type {
	case "call.incoming":
		return s.HandleIncoming(ctx, tenantID, env.Payload)
	case "call.missed":
		return s.HandleMissed(ctx, tenantID, env.Payload)
	case "call.state":
		return s.HandleState(ctx, tenantID, env.Payload)
	case "call.ended":
		return s.HandleEnded(ctx, tenantID, env.Payload)
	case "call.quality":
		return s.HandleQuality(ctx, tenantID, env.Payload)
	case "call.reset":
		return s.HandleReset(ctx, tenantID, env.Payload)
	}
	log.Printf("[calls] evento ignorado: %s", env.Type)
	return nil
}
