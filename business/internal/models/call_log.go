package models

import (
	"time"

	"github.com/google/uuid"
)

// CallLog é o registro de uma chamada de voz do WhatsApp (1:1). A identidade é
// (tenantId, callId): o callId vem do WhatsApp e só é único dentro de uma
// empresa para fins do Watink, então a unicidade NUNCA é global.
type CallLog struct {
	ID         int       `gorm:"primaryKey" json:"id"`
	TenantID   uuid.UUID `gorm:"column:tenantId;type:uuid;not null;uniqueIndex:idx_calllogs_tenant_call,priority:1;index:idx_calllogs_tenant_started,priority:1" json:"-"`
	CallID     string    `gorm:"column:callId;not null;uniqueIndex:idx_calllogs_tenant_call,priority:2" json:"callId"`
	WhatsappID int       `gorm:"column:whatsappId;not null;index" json:"whatsappId"`
	ContactID  *int      `gorm:"column:contactId" json:"contactId"`
	TicketID   *int      `gorm:"column:ticketId" json:"ticketId"`

	// Direction: incoming | outgoing.
	Direction string `gorm:"not null" json:"direction"`
	// Media: audio | video. Ausente (registros anteriores ao vídeo) vale audio.
	Media string `gorm:"not null;default:'audio'" json:"media"`
	// Status: ringing | active | ended | missed | rejected | failed | interrupted.
	Status string `gorm:"not null;default:'ringing';index" json:"status"`
	// PeerJid/CallerPn identificam o chamador mesmo quando ainda não há contato.
	PeerJid  string `gorm:"column:peerJid" json:"peerJid"`
	CallerPn string `gorm:"column:callerPn" json:"callerPn"`

	StartedAt   time.Time  `gorm:"column:startedAt;not null;index:idx_calllogs_tenant_started,priority:2" json:"startedAt"`
	AnsweredAt  *time.Time `gorm:"column:answeredAt" json:"answeredAt"`
	EndedAt     *time.Time `gorm:"column:endedAt" json:"endedAt"`
	DurationSec int        `gorm:"column:durationSec;not null;default:0" json:"durationSec"`

	HandledByUserID *int `gorm:"column:handledByUserId" json:"handledByUserId"`
	// EndReason: user_ended | declined | timeout | busy | cancelled | failed |
	// no_operator | proxy_blocked | unsupported_type | accepted_elsewhere | interrupted.
	EndReason string `gorm:"column:endReason" json:"endReason"`

	// Resumo agregado de qualidade (não guarda a série temporal). O índice é uma
	// ESTIMATIVA (modelo E simplificado) e a perda só cobre contato→operador.
	RttAvg         *float64 `gorm:"column:rttAvg" json:"rttAvg"`
	RttMax         *float64 `gorm:"column:rttMax" json:"rttMax"`
	LossAvg        *float64 `gorm:"column:lossAvg" json:"lossAvg"`
	LossMax        *float64 `gorm:"column:lossMax" json:"lossMax"`
	JitterAvg      *float64 `gorm:"column:jitterAvg" json:"jitterAvg"`
	JitterMax      *float64 `gorm:"column:jitterMax" json:"jitterMax"`
	MosEstimated   *float64 `gorm:"column:mosEstimated" json:"mosEstimated"`
	QualitySamples int      `gorm:"column:qualitySamples;not null;default:0" json:"-"`

	// RecordingKey é a CHAVE do objeto no S3 (nunca URL assinada, que expira).
	RecordingKey         string `gorm:"column:recordingKey" json:"-"`
	RecordingStatus      string `gorm:"column:recordingStatus" json:"recordingStatus"`
	RecordingDurationSec int    `gorm:"column:recordingDurationSec;not null;default:0" json:"recordingDurationSec"`

	CreatedAt time.Time `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updatedAt" json:"updatedAt"`
}

func (CallLog) TableName() string { return "CallLogs" }

// CallRecordingAccess audita quem ouviu ou excluiu uma gravação.
type CallRecordingAccess struct {
	ID       int       `gorm:"primaryKey" json:"id"`
	TenantID uuid.UUID `gorm:"column:tenantId;type:uuid;not null;index:idx_callrecaccess_tenant_call,priority:1" json:"-"`
	CallID   string    `gorm:"column:callId;not null;index:idx_callrecaccess_tenant_call,priority:2" json:"callId"`
	UserID   int       `gorm:"column:userId;not null" json:"userId"`
	// Action: listen | delete.
	Action string    `gorm:"not null" json:"action"`
	At     time.Time `gorm:"not null" json:"at"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (CallRecordingAccess) TableName() string { return "CallRecordingAccesses" }
