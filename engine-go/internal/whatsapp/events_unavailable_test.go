package whatsapp

import (
	"testing"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func unavailableEvt(utype events.UnavailableType, unavailable bool) *events.UndecryptableMessage {
	return &events.UndecryptableMessage{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:   types.NewJID("165923277262973", types.HiddenUserServer),
				Sender: types.NewJID("165923277262973", types.HiddenUserServer),
			},
			ID: "AC14283B1D77A001FDADD43526B99DCE", Timestamp: time.Unix(1791339493, 0), PushName: "Diomedes Neto",
		},
		IsUnavailable: unavailable, UnavailableType: utype,
	}
}

// O WhatsApp NÃO entrega a visualização única a aparelhos vinculados: chega só <unavailable type="view_once"/>,
// sem nenhum conteúdo. Antes o engine ignorava o evento e o atendente nem sabia que o cliente mandou algo. Agora
// vira uma mensagem no chat, marcada como visualização única e sem mídia.
func TestUnavailableViewOnce_BecomesAChatNotice(t *testing.T) {
	svc, calls := newTestService()
	svc.handleUndecryptable(nil, 1, "tenant-a", unavailableEvt(events.UnavailableTypeViewOnce, true))

	if len(*calls) != 1 {
		t.Fatalf("esperava 1 evento publicado, veio %d", len(*calls))
	}
	c := (*calls)[0]
	if c.eventType != "message.received" || c.tenantID != "tenant-a" || c.sessionID != 1 {
		t.Fatalf("evento %+v", c)
	}
	m := c.payload["message"].(map[string]interface{})
	if m["id"] != "AC14283B1D77A001FDADD43526B99DCE" {
		t.Fatalf("o id tem de ser o do WhatsApp (a re-entrega não duplica): %v", m["id"])
	}
	if m["type"] != "view_once" {
		t.Fatalf("type = %v, esperado view_once", m["type"])
	}
	if m["mediaProto"] != "" || m["thumbnail"] != "" {
		t.Fatalf("não há conteúdo para baixar: %v %v", m["mediaProto"], m["thumbnail"])
	}
	if m["from"] != "165923277262973@lid" || m["pushName"] != "Diomedes Neto" || m["fromMe"] != false {
		t.Fatalf("remetente: %v", m)
	}
	if m["timestamp"] != int64(1791339493) {
		t.Fatalf("timestamp = %v", m["timestamp"])
	}
}

// Outras mensagens indisponíveis (não visualização única) e falhas comuns de decifragem NÃO viram aviso: o
// whatsmeow já pede um reenvio e a mensagem chega depois normalmente; avisar duplicaria o chat.
func TestUndecryptableOtherThanViewOnce_IsNotPublished(t *testing.T) {
	svc, calls := newTestService()
	svc.handleUndecryptable(nil, 1, "t", unavailableEvt(events.UnavailableTypeUnknown, true))
	svc.handleUndecryptable(nil, 1, "t", unavailableEvt(events.UnavailableTypeUnknown, false))
	if len(*calls) != 0 {
		t.Fatalf("publicou %d eventos para mensagens que o whatsmeow vai reentregar", len(*calls))
	}
}

// Status/Stories não são conversas (mesma regra do handler de mensagens).
func TestUnavailableViewOnce_StatusBroadcastIsIgnored(t *testing.T) {
	svc, calls := newTestService()
	e := unavailableEvt(events.UnavailableTypeViewOnce, true)
	e.Info.Chat = types.NewJID("status", "broadcast")
	svc.handleUndecryptable(nil, 1, "t", e)
	if len(*calls) != 0 {
		t.Fatal("status@broadcast não gera ticket")
	}
}

// Visualização única que NÓS mandamos pelo celular (fromMe) também chega indisponível: vira aviso do nosso lado.
func TestUnavailableViewOnce_FromMeKeepsDirection(t *testing.T) {
	svc, calls := newTestService()
	e := unavailableEvt(events.UnavailableTypeViewOnce, true)
	e.Info.IsFromMe = true
	svc.handleUndecryptable(nil, 1, "t", e)
	m := (*calls)[0].payload["message"].(map[string]interface{})
	if m["fromMe"] != true {
		t.Fatalf("fromMe = %v", m["fromMe"])
	}
}

// O handler só serve se o despachante o chamar: o evento do whatsmeow chega por handleEvent.
func TestHandleEvent_RoutesUnavailableViewOnce(t *testing.T) {
	svc, calls := newTestService()
	svc.clients[1] = whatsmeow.NewClient(&store.Device{}, nil)
	svc.handleEvent(1, "tenant-v", unavailableEvt(events.UnavailableTypeViewOnce, true))
	if len(*calls) != 1 || (*calls)[0].eventType != "message.received" {
		t.Fatalf("o despachante não roteou a visualização única: %+v", *calls)
	}
}
