// Package wbot defines the versioned AMQP contract shared by business and engine-go.
package wbot

import (
	"encoding/json"
	"fmt"
	"strconv"
)

const Version = "v1"

const (
	ExchangeCommands = "wbot.commands"
	ExchangeEvents   = "wbot.events"
	ExchangeDLQ      = "wbot.dlq"
)

const (
	CommandSessionStart       = "session.start"
	CommandSessionStop        = "session.stop"
	CommandSessionDelete      = "session.delete"
	CommandMessageText        = "message.send.text"
	CommandMessageMedia       = "message.send.media"
	CommandMessageButtons     = "message.send.buttons"
	CommandMessageList        = "message.send.list"
	CommandMessagePoll        = "message.send.poll"
	CommandMessagePollVote    = "message.vote.poll"
	CommandMessageInteractive = "message.send.interactive"
	CommandMessageTemplate    = "message.send.template"
	CommandMessageCarousel    = "message.send.carousel"
	CommandMessageReact       = "message.react"
	CommandMessageMarkRead    = "message.markAsRead"
	CommandMediaDownload      = "media.download"
	CommandContactSync        = "contact.sync"
	CommandContactImport      = "contact.import"
	CommandContactSave        = "contact.save" // grava o contato na agenda do celular (como o WhatsApp Web)
	CommandHistoryRecover     = "history.recover"
	CommandChatPresence       = "chat.presence"
	CommandPresenceSubscribe  = "presence.subscribe"
	CommandPresencePolicy     = "presence.policy"
	CommandCallReady          = "call.ready"
	CommandCallAccept         = "call.accept"
	CommandCallReject         = "call.reject"
	CommandCallEnd            = "call.end"
	CommandCallStart          = "call.start"
)

const (
	EventEngineReady          = "engine.ready"
	EventSessionQRCode        = "session.qrcode"
	EventSessionPairingCode   = "session.pairing_code"
	EventSessionStatus        = "session.status"
	EventSessionHistorySync   = "session.history_sync"
	EventSessionRisk          = "session.risk"
	EventSessionJIDRegistered = "session.jid_registered"
	EventMessageReceived      = "message.received"
	EventMessageAck           = "message.ack"
	EventMessageRevoke        = "message.revoke"
	EventMessageEdited        = "message.edited" // o contato editou uma mensagem (texto novo + id da original)
	EventMessageReaction      = "message.reaction"
	EventMessageMedia         = "message.media"
	EventMessagePollVote      = "message.poll_vote"
	EventContactUpdate        = "contact.update"
	EventContactImport        = "contact.import"
	EventContactSaved         = "contact.saved" // resultado da gravação do contato na agenda do celular
	EventContactPresence      = "contact.presence"
)

// Command returns a routing key for a command sent to engine-go.
func Command(tenantID, connectionID, command string) string {
	return fmt.Sprintf("wbot.%s.%s.%s", tenantID, connectionID, command)
}

// Event returns a routing key for an event sent to business.
func Event(tenantID, connectionID, event string) string {
	return fmt.Sprintf("wbot.%s.%s.%s", tenantID, connectionID, event)
}

// CommandPattern returns the topic binding for all tenants and connections.
func CommandPattern(command string) string { return "wbot.*.*." + command }

// EventPattern returns the topic binding for all tenants and connections.
func EventPattern(event string) string { return "wbot.*.*." + event }

// Envelope is the versioned wire wrapper used by engine events.
type Envelope struct {
	Version   string          `json:"version"`
	Type      string          `json:"type"`
	TenantID  string          `json:"tenantId"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp int64           `json:"timestamp,omitempty"`
}

type SessionStart struct {
	SessionID string `json:"sessionId"`
	Name      string `json:"name"`
	JID       string `json:"jid,omitempty"`
}

type SessionStatus struct {
	SessionID       string `json:"sessionId"`
	Status          string `json:"status"`
	Number          string `json:"number,omitempty"`
	ProfilePicURL   string `json:"profilePicUrl,omitempty"`
	FirstConnection bool   `json:"firstConnection,omitempty"`
}

type MessageReceived struct {
	SessionID string          `json:"sessionId"`
	Message   json.RawMessage `json:"message"`
}

// ParseSessionID validates the numeric connection/session identifier in routing keys.
func ParseSessionID(value string) (int, error) { return strconv.Atoi(value) }
