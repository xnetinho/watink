package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/alltomatos/watinkdev/engine-go/internal/callsapi"
	"github.com/alltomatos/watinkdev/engine-go/internal/command"
	"github.com/alltomatos/watinkdev/engine-go/internal/groupsapi"
	"github.com/alltomatos/watinkdev/engine-go/internal/health"
	"github.com/alltomatos/watinkdev/engine-go/internal/rabbitmq"
	"github.com/alltomatos/watinkdev/engine-go/internal/safego"
	"github.com/alltomatos/watinkdev/engine-go/internal/whatsapp"
	"github.com/joho/godotenv"
	amqp "github.com/streadway/amqp"
	"go.mau.fi/whatsmeow"
)

type commandEnvelope struct {
	Type      string          `json:"type"`
	Timestamp int64           `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

func main() {
	_ = godotenv.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Health server — responde /health antes mesmo do RabbitMQ conectar
	// O serviço só existe depois do RabbitMQ; até lá a carga é zero.
	var callsLoad atomic.Pointer[func() (int, int)]
	go health.Start(ctx, func() (int, int) {
		if f := callsLoad.Load(); f != nil {
			return (*f)()
		}
		return 0, 0
	})

	rabbit := rabbitmq.NewRabbitMQService()
	if err := rabbit.Connect(); err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	log.Println("Connected to RabbitMQ")

	sessionLoader := whatsapp.NewPostgresSessionLoader(whatsapp.BuildPostgresDSN())
	waService := whatsapp.NewWhatsAppService(rabbit, sessionLoader)

	// API interna de grupos/comunidades (T1.1, docker-internal only) — no-op
	// se GROUPS_API_TOKEN não estiver configurado (fail-closed).
	safego.Go("groupsapi", func() { groupsapi.Start(ctx, waService) })

	// Endpoint interno de áudio das chamadas (docker-internal only, sem autenticação:
	// a defesa é a rede — a porta nunca pode ser publicada em `ports:`).
	load := waService.CallsLoad
	callsLoad.Store(&load)
	safego.Go("callsapi", func() { callsapi.Start(ctx, waService) })

	safego.Go("autoRestartSessions", func() {
		time.Sleep(5 * time.Second)
		log.Println("Auto-restarting existing sessions...")
		waService.AutoRestartSessions()
	})

	routingKeys := []string{
		"wbot.*.*.session.start",
		"wbot.*.*.session.stop",
		"wbot.*.*.session.delete",
		"wbot.*.*.message.send.text",
		"wbot.*.*.message.send.media",
		"wbot.*.*.message.send.buttons",
		"wbot.*.*.message.send.list",
		"wbot.*.*.message.send.poll",
		"wbot.*.*.message.send.interactive",
		"wbot.*.*.message.send.template",
		"wbot.*.*.message.send.carousel",
		"wbot.*.*.message.react",
		"wbot.*.*.message.markAsRead",
		"wbot.*.*.media.download",
		"wbot.*.*.contact.sync",
		"wbot.*.*.contact.import",
		"wbot.*.*.history.sync",
		"wbot.*.*.history.recover",
		"wbot.*.*.chat.presence",
	}

	// O consumidor devolve o erro: nil = Ack; recuperável = reentrega com limite; irrecuperável ou esgotado = fila de
	// falhas (engine.go.commands.dlq). Antes o Ack era dado sempre, até em falha, e o comando se perdia.
	err := rabbit.ConsumeCommands("engine.go.commands", routingKeys, func(d amqp.Delivery) error {
		err := handleCommand(d, waService)
		if err != nil {
			log.Printf("Command failed %s: %v", d.RoutingKey, err)
		}
		return classifyCommandError(err)
	})
	if err != nil {
		log.Fatalf("Failed to start command consumer: %v", err)
	}

	// Comandos de chamada têm fila, canal e consumidor próprios, e cada um roda em
	// goroutine: "Atender"/"Encerrar" nunca esperam o envio de mensagens (que usa
	// o laço serial acima).
	if err := rabbit.ConsumeCommandsConcurrent(callsQueue, callRoutingKeys, func(d amqp.Delivery) error {
		err := handleCommand(d, waService)
		if err != nil {
			log.Printf("Call command failed %s: %v", d.RoutingKey, err)
		}
		return classifyCommandError(err)
	}); err != nil {
		log.Fatalf("Failed to start call command consumer: %v", err)
	}

	log.Println("Watink Engine Go (whatsmeow) started — PostgreSQL store, RabbitMQ connected")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	log.Println("Shutting down gracefully...")
	cancel() // para health server
	_ = rabbit.Close()
	log.Println("Engine stopped")
}

// classifyCommandError separa o que vale a pena repetir do que não vale. A sessão desconectada é recuperável (ela pode
// estar reconectando); JSON inválido, JID inválido e sessão inexistente nunca vão passar sozinhos.
func classifyCommandError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, rabbitmq.ErrUnrecoverable):
		return err
	case errors.Is(err, whatsapp.ErrSessionNotConnected), errors.Is(err, whatsapp.ErrCallsUnavailable):
		return err // recuperável: tenta de novo com backoff
	case errors.Is(err, whatsapp.ErrInvalidInput), errors.Is(err, whatsapp.ErrInvalidGroupJID):
		return rabbitmq.Unrecoverable(err)
	}
	var syn *json.SyntaxError
	var typ *json.UnmarshalTypeError
	if errors.As(err, &syn) || errors.As(err, &typ) {
		return rabbitmq.Unrecoverable(err)
	}
	var iq *whatsmeow.IQError
	if errors.As(err, &iq) {
		// 4xx do WhatsApp é decisão do servidor (sem permissão, limite, número inexistente): repetir só piora o risco.
		if iq.Code >= 400 && iq.Code < 500 {
			return rabbitmq.Unrecoverable(err)
		}
		return err
	}
	if strings.Contains(err.Error(), "invalid JID") || strings.Contains(err.Error(), "unmarshal envelope") {
		return rabbitmq.Unrecoverable(err)
	}
	return err
}

const callsQueue = "engine.go.calls"

var callRoutingKeys = []string{"wbot.*.*.call.*"}

func handleCommand(d amqp.Delivery, svc *whatsapp.WhatsAppService) error {
	log.Printf("Received command: %s", d.RoutingKey)

	tenantID, sessionIDStr, cmd, err := command.ParseRoutingKey(d.RoutingKey)
	if err != nil {
		log.Printf("%v", err)
		return nil
	}

	sessionID, err := strconv.Atoi(sessionIDStr)
	if err != nil {
		log.Printf("Invalid session ID in routing key %s: %v", d.RoutingKey, err)
		return nil
	}

	var env commandEnvelope
	if err := json.Unmarshal(d.Body, &env); err != nil {
		return fmt.Errorf("unmarshal envelope: %w", err)
	}

	switch cmd {
	case "call.ready", "call.accept", "call.reject", "call.end", "call.start":
		var p whatsapp.CallPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return err
		}
		return svc.HandleCallCommand(sessionID, cmd, p)

	case "session.start":
		var p struct {
			ProxyURL       string `json:"proxyUrl"`
			UsePairingCode bool   `json:"usePairingCode"`
			PhoneNumber    string `json:"phoneNumber"`
			Name           string `json:"name"`
			Wid            string `json:"wid"`
		}
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return err
		}
		ts := env.Timestamp
		if ts == 0 {
			ts = time.Now().UnixMilli()
		}
		return svc.StartClient(sessionID, tenantID, p.Name, ts, p.ProxyURL, p.UsePairingCode, p.PhoneNumber, p.Wid)

	case "session.stop":
		return svc.StopClient(sessionID)

	case "session.delete":
		if err := svc.StopClient(sessionID); err != nil {
			log.Printf("Stop client warning for %d: %v", sessionID, err)
		}
		return svc.ForceLogout(sessionID)

	case "message.send.text":
		var p whatsapp.TextCommandPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return err
		}
		return svc.SendText(sessionID, tenantID, p)

	case "message.send.media":
		var p whatsapp.MediaCommandPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return err
		}
		return svc.SendMedia(sessionID, tenantID, p)

	case "message.send.buttons":
		var p whatsapp.ButtonsCommandPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return err
		}
		return svc.SendButtons(sessionID, tenantID, p)

	case "message.send.list":
		var p whatsapp.ListCommandPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return err
		}
		return svc.SendList(sessionID, tenantID, p)

	case "message.send.poll":
		var p whatsapp.PollCommandPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return err
		}
		return svc.SendPoll(sessionID, tenantID, p)

	case "message.send.interactive":
		var p whatsapp.InteractiveCommandPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return err
		}
		return svc.SendInteractive(sessionID, tenantID, p)

	case "message.send.template":
		var p whatsapp.TemplateCommandPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return err
		}
		return svc.SendTemplate(sessionID, tenantID, p)

	case "message.send.carousel":
		var p whatsapp.CarouselCommandPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return err
		}
		return svc.SendCarousel(sessionID, tenantID, p)

	case "chat.presence":
		var p whatsapp.PresenceCommandPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return err
		}
		return svc.SendPresence(sessionID, tenantID, p)

	case "message.react":
		var p whatsapp.ReactionCommandPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return err
		}
		return svc.SendReaction(sessionID, tenantID, p)

	case "message.markAsRead":
		var p whatsapp.MarkReadCommandPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return err
		}
		return svc.MarkRead(sessionID, tenantID, p)

	case "media.download":
		var p whatsapp.DownloadMediaCommandPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return err
		}
		// O consumidor de comandos é um único loop sequencial (ConsumeCommands
		// processa uma Delivery por vez para TODAS as sessões/tenants) — um
		// download de mídia real (rede até o CDN do WhatsApp, minutos em casos
		// ruins) travaria toda fila de comandos atrás dele. media.download já é
		// assíncrono do ponto de vista do chamador (o resultado só chega pelo
		// evento message.media, nunca pelo retorno síncrono do comando), então
		// rodar em goroutine própria não muda o contrato — só libera a fila.
		safego.Go("media.download", func() {
			if err := svc.DownloadMedia(sessionID, tenantID, p); err != nil {
				log.Printf("Command failed %s: %v", d.RoutingKey, err)
			}
		})
		return nil

	case "contact.sync":
		var p whatsapp.SyncContactPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return err
		}
		return svc.SyncContact(sessionID, tenantID, p)

	case "contact.import":
		return svc.ImportContacts(sessionID, tenantID)

	case "history.recover":
		var p whatsapp.HistoryRecoverPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return err
		}
		return svc.RecoverHistory(sessionID, tenantID, p)

	case "history.sync":
		log.Printf("Command %s received but not implemented in engine-go", cmd)
		return nil

	default:
		log.Printf("Unknown command type: %s", cmd)
		return nil
	}
}
