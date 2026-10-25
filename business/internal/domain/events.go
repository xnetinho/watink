package domain

import (
	"encoding/json"

	"github.com/google/uuid"
)

// Domain Events
type DomainEvent interface {
	EventName() string
	TenantID() uuid.UUID
}

type TicketAssignedEvent struct {
	TicketID int
	UserID   int
	tenantID uuid.UUID
}

type ticketAssignedJSON struct {
	TicketID int       `json:"ticketId"`
	UserID   int       `json:"userId"`
	TenantID uuid.UUID `json:"tenantId"`
}

func (e TicketAssignedEvent) MarshalJSON() ([]byte, error) {
	return json.Marshal(ticketAssignedJSON{TicketID: e.TicketID, UserID: e.UserID, TenantID: e.tenantID})
}

func (e *TicketAssignedEvent) UnmarshalJSON(b []byte) error {
	var j ticketAssignedJSON
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	e.TicketID = j.TicketID
	e.UserID = j.UserID
	e.tenantID = j.TenantID
	return nil
}

func (e TicketAssignedEvent) EventName() string   { return "TicketAssigned" }
func (e TicketAssignedEvent) TenantID() uuid.UUID { return e.tenantID }

func NewTicketAssignedEvent(ticketID, userID int, tenantID uuid.UUID) TicketAssignedEvent {
	return TicketAssignedEvent{TicketID: ticketID, UserID: userID, tenantID: tenantID}
}

type MessageReceivedEvent struct {
	MessageID string
	TicketID  int
	tenantID  uuid.UUID
}

type messageReceivedJSON struct {
	MessageID string    `json:"messageId"`
	TicketID  int       `json:"ticketId"`
	TenantID  uuid.UUID `json:"tenantId"`
}

func (e MessageReceivedEvent) MarshalJSON() ([]byte, error) {
	return json.Marshal(messageReceivedJSON{MessageID: e.MessageID, TicketID: e.TicketID, TenantID: e.tenantID})
}

func (e *MessageReceivedEvent) UnmarshalJSON(b []byte) error {
	var j messageReceivedJSON
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	e.MessageID = j.MessageID
	e.TicketID = j.TicketID
	e.tenantID = j.TenantID
	return nil
}

func (e MessageReceivedEvent) EventName() string   { return "MessageReceived" }
func (e MessageReceivedEvent) TenantID() uuid.UUID { return e.tenantID }

func NewMessageReceivedEvent(messageID string, ticketID int, tenantID uuid.UUID) MessageReceivedEvent {
	return MessageReceivedEvent{MessageID: messageID, TicketID: ticketID, tenantID: tenantID}
}

type TicketStatusChangedEvent struct {
	TicketID  int
	OldStatus string
	NewStatus string
	tenantID  uuid.UUID
}

type ticketStatusChangedJSON struct {
	TicketID  int       `json:"ticketId"`
	OldStatus string    `json:"oldStatus"`
	NewStatus string    `json:"newStatus"`
	TenantID  uuid.UUID `json:"tenantId"`
}

func (e TicketStatusChangedEvent) MarshalJSON() ([]byte, error) {
	return json.Marshal(ticketStatusChangedJSON{TicketID: e.TicketID, OldStatus: e.OldStatus, NewStatus: e.NewStatus, TenantID: e.tenantID})
}

func (e *TicketStatusChangedEvent) UnmarshalJSON(b []byte) error {
	var j ticketStatusChangedJSON
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	e.TicketID = j.TicketID
	e.OldStatus = j.OldStatus
	e.NewStatus = j.NewStatus
	e.tenantID = j.TenantID
	return nil
}

func (e TicketStatusChangedEvent) EventName() string   { return "TicketStatusChanged" }
func (e TicketStatusChangedEvent) TenantID() uuid.UUID { return e.tenantID }

func NewTicketStatusChangedEvent(ticketID int, oldStatus, newStatus string, tenantID uuid.UUID) TicketStatusChangedEvent {
	return TicketStatusChangedEvent{TicketID: ticketID, OldStatus: oldStatus, NewStatus: newStatus, tenantID: tenantID}
}
