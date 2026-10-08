//go:build integration

package services

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/pkg/mediastore"
	"github.com/streadway/amqp"
)

// Prova o elo que o teste unitário não cobre: o que SAI do broker (o que o engine lê) leva os bytes.
// Regressão: a imagem enviada pelo chat ficava no relógio porque o engine recebia só "/public/media/x.png".
func TestRabbitMQService_PublishCommand_SendMediaCarriesTheBytes(t *testing.T) {
	t.Chdir(t.TempDir())
	url, err := mediastore.SaveMediaReader(strings.NewReader("IMG-BYTES"), "image/png")
	if err != nil {
		t.Fatal(err)
	}

	svc := NewRabbitMQProvider(rabbitMQURL(t))
	if err := svc.Connect(); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer svc.Close()

	conn, err := amqp.Dial(rabbitMQURL(t))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		t.Fatal(err)
	}
	q, err := ch.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	const key = "wbot.tenant-x.1.message.send.media"
	if err := ch.QueueBind(q.Name, key, "wbot.commands", false, nil); err != nil {
		t.Fatal(err)
	}
	msgs, err := ch.Consume(q.Name, "", true, true, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}

	cmd := map[string]interface{}{"type": "message.send.media", "payload": map[string]interface{}{"sessionId": 1, "mediaUrl": url, "mediaType": "image"}}
	if err := svc.PublishCommand(key, cmd); err != nil {
		t.Fatalf("PublishCommand: %v", err)
	}

	select {
	case d := <-msgs:
		var got struct {
			Payload struct {
				MediaURL  string `json:"mediaUrl"`
				MediaData string `json:"mediaData"`
			} `json:"payload"`
		}
		if err := json.Unmarshal(d.Body, &got); err != nil {
			t.Fatal(err)
		}
		if got.Payload.MediaData != base64.StdEncoding.EncodeToString([]byte("IMG-BYTES")) {
			t.Fatalf("o engine receberia mediaData = %q", got.Payload.MediaData)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("o comando não chegou na fila")
	}
}
