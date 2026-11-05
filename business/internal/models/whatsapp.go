package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// KindWhatsApp é o valor de ChannelConnection.Kind das conexões WhatsApp.
const KindWhatsApp = "whatsapp"

// Whatsapp é a visão achatada de uma conexão WhatsApp: a linha de ChannelConnections (kind="whatsapp") mais a de
// whatsapp_connection_details. NÃO é tabela: ler e gravar só pelas funções deste arquivo.
type Whatsapp struct {
	ID              int        `json:"id"`
	Status          string     `json:"status"`
	Name            string     `json:"name"`
	IsDefault       bool       `json:"isDefault"`
	GreetingMessage string     `json:"greetingMessage"`
	FarewellMessage string     `json:"farewellMessage"`
	TenantID        uuid.UUID  `json:"tenantId"`
	Number          string     `json:"number"`        // ChannelConnection.Identifier
	ProfilePicUrl   string     `json:"profilePicUrl"` // ChannelConnection.AvatarURL
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	FirstConnection *time.Time `json:"firstConnection"` // ChannelConnection.FirstConnectedAt
	LastConnectedAt *time.Time `json:"lastConnectedAt"`

	SyncHistory         bool   `json:"syncHistory"`
	SyncPeriod          string `json:"syncPeriod"`
	KeepAlive           bool   `json:"keepAlive"`
	PresencePolicy      string `json:"presencePolicy"`
	PresenceIdleMinutes int    `json:"presenceIdleMinutes"`
	EngineType          string `json:"engineType"`
	Wid                 string `json:"wid"`
	// Proxy (anti-ban). ProxyMode: "none" | "single" | "group". single → ProxyID fixo. group → ProxyGroupID + ProxyID
	// guarda o pick atual (sticky) ou o último usado (rotate).
	ProxyMode         string `json:"proxyMode"`
	ProxyID           *int   `json:"proxyId"`
	ProxyGroupID      *int   `json:"proxyGroupId"`
	ConnectionGroupID *int   `json:"connectionGroupId"`
	// LastRisk* guardam o último sinal de risco de banimento (IQ 401/403/429/463) para a tela avisar mesmo depois de
	// perder o evento SSE.
	LastRiskCode    int        `json:"lastRiskCode"`
	LastRiskAction  string     `json:"lastRiskAction"`
	LastRiskMessage string     `json:"lastRiskMessage"`
	LastRiskAt      *time.Time `json:"lastRiskAt"`

	Queues []Queue `json:"queues,omitempty"`
}

// connectionColumns e detailColumns traduzem a chave usada pelos chamadores (nome da coluna na tela/API) para a coluna da
// tabela que a guarda.
var connectionColumns = map[string]string{
	"status": "status", "name": "name", "isDefault": "isDefault",
	"greetingMessage": "greetingMessage", "farewellMessage": "farewellMessage",
	"number": "identifier", "profilePicUrl": "avatarUrl",
	"firstConnection": "firstConnectedAt", "lastConnectedAt": "lastConnectedAt",
}

var detailColumns = map[string]bool{
	"syncHistory": true, "syncPeriod": true, "keepAlive": true, "presencePolicy": true, "presenceIdleMinutes": true,
	"engineType": true, "wid": true, "proxyMode": true, "proxyId": true, "proxyGroupId": true, "connectionGroupId": true,
	"lastRiskCode": true, "lastRiskAction": true, "lastRiskMessage": true, "lastRiskAt": true,
}

func (w *Whatsapp) parts() (ChannelConnection, WhatsAppConnectionDetails) {
	cc := ChannelConnection{
		ID: w.ID, TenantID: w.TenantID, Kind: KindWhatsApp, Name: w.Name, Status: w.Status, IsDefault: w.IsDefault,
		GreetingMessage: w.GreetingMessage, FarewellMessage: w.FarewellMessage,
		Identifier: w.Number, AvatarURL: w.ProfilePicUrl, FirstConnectedAt: w.FirstConnection, LastConnectedAt: w.LastConnectedAt,
	}
	d := WhatsAppConnectionDetails{
		ConnectionID: w.ID, EngineType: w.EngineType, PresencePolicy: w.PresencePolicy, PresenceIdleMinutes: w.PresenceIdleMinutes,
		ConnectionGroupID: w.ConnectionGroupID, ProxyMode: w.ProxyMode, ProxyID: w.ProxyID, ProxyGroupID: w.ProxyGroupID,
		SyncHistory: w.SyncHistory, SyncPeriod: w.SyncPeriod, KeepAlive: w.KeepAlive, WID: w.Wid,
		LastRiskCode: w.LastRiskCode, LastRiskAction: w.LastRiskAction, LastRiskMessage: w.LastRiskMessage, LastRiskAt: w.LastRiskAt,
	}
	return cc, d
}

func joinWhatsapp(cc ChannelConnection, d WhatsAppConnectionDetails) Whatsapp {
	return Whatsapp{
		ID: cc.ID, Status: cc.Status, Name: cc.Name, IsDefault: cc.IsDefault, GreetingMessage: cc.GreetingMessage,
		FarewellMessage: cc.FarewellMessage, TenantID: cc.TenantID, Number: cc.Identifier, ProfilePicUrl: cc.AvatarURL,
		CreatedAt: cc.CreatedAt, UpdatedAt: cc.UpdatedAt, FirstConnection: cc.FirstConnectedAt, LastConnectedAt: cc.LastConnectedAt,
		SyncHistory: d.SyncHistory, SyncPeriod: d.SyncPeriod, KeepAlive: d.KeepAlive, PresencePolicy: d.PresencePolicy,
		PresenceIdleMinutes: d.PresenceIdleMinutes, EngineType: d.EngineType, Wid: d.WID, ProxyMode: d.ProxyMode,
		ProxyID: d.ProxyID, ProxyGroupID: d.ProxyGroupID, ConnectionGroupID: d.ConnectionGroupID,
		LastRiskCode: d.LastRiskCode, LastRiskAction: d.LastRiskAction, LastRiskMessage: d.LastRiskMessage, LastRiskAt: d.LastRiskAt,
	}
}

func fresh(db *gorm.DB) *gorm.DB { return db.Session(&gorm.Session{NewDB: true}) }

// TenantWhatsappIDs é a subconsulta com os ids das conexões WhatsApp da empresa, para filtrar os detalhes (que não têm
// tenantId próprio).
func TenantWhatsappIDs(db *gorm.DB, tenantID uuid.UUID) *gorm.DB {
	return fresh(db).Model(&ChannelConnection{}).Select("id").Where(`"tenantId" = ? AND kind = ?`, tenantID, KindWhatsApp)
}

// CreateWhatsapp grava a conexão e os seus detalhes na mesma transação e devolve w já com id, datas e padrões do banco.
// Com w.ID zero o banco gera o id.
func CreateWhatsapp(db *gorm.DB, w *Whatsapp) error {
	return fresh(db).Transaction(func(tx *gorm.DB) error {
		cc, d := w.parts()
		if err := tx.Create(&cc).Error; err != nil {
			return err
		}
		d.ConnectionID = cc.ID
		if err := tx.Create(&d).Error; err != nil {
			return err
		}
		return FirstWhatsapp(tx.Where("id = ?", cc.ID), w)
	})
}

// FirstWhatsapp lê a primeira conexão WhatsApp que casa com as condições de q (Where/Select/Order). Devolve
// gorm.ErrRecordNotFound se não houver.
func FirstWhatsapp(q *gorm.DB, dest *Whatsapp) error {
	var cc ChannelConnection
	if err := q.Model(&ChannelConnection{}).Where(`kind = ?`, KindWhatsApp).First(&cc).Error; err != nil {
		return err
	}
	d, err := loadDetails(q, []int{cc.ID})
	if err != nil {
		return err
	}
	*dest = joinWhatsapp(cc, d[cc.ID])
	return nil
}

// FindWhatsapps lê todas as conexões WhatsApp que casam com as condições de q.
func FindWhatsapps(q *gorm.DB, dest *[]Whatsapp) error {
	var ccs []ChannelConnection
	if err := q.Model(&ChannelConnection{}).Where(`kind = ?`, KindWhatsApp).Find(&ccs).Error; err != nil {
		return err
	}
	ids := make([]int, len(ccs))
	for i := range ccs {
		ids[i] = ccs[i].ID
	}
	d, err := loadDetails(q, ids)
	if err != nil {
		return err
	}
	out := make([]Whatsapp, len(ccs))
	for i := range ccs {
		out[i] = joinWhatsapp(ccs[i], d[ccs[i].ID])
	}
	*dest = out
	return nil
}

func loadDetails(q *gorm.DB, ids []int) (map[int]WhatsAppConnectionDetails, error) {
	out := make(map[int]WhatsAppConnectionDetails, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var rows []WhatsAppConnectionDetails
	if err := fresh(q).Where(`"connectionId" IN ?`, ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.ConnectionID] = r
	}
	for _, id := range ids {
		if _, ok := out[id]; !ok {
			out[id] = WhatsAppConnectionDetails{ConnectionID: id, EngineType: "whatsmeow", PresencePolicy: "never", PresenceIdleMinutes: 5, ProxyMode: "none"}
		}
	}
	return out, nil
}

// LoadWhatsappQueues preenche w.Queues com as filas ligadas à conexão.
func LoadWhatsappQueues(db *gorm.DB, w *Whatsapp) error {
	w.Queues = nil
	return fresh(db).Raw(
		`SELECT q.* FROM "Queues" q JOIN channel_connection_queues cq ON cq.queue_id = q.id WHERE cq.channel_connection_id = ?`, w.ID,
	).Scan(&w.Queues).Error
}

// UpdateWhatsapp aplica uma atualização parcial: cada chave vai para a tabela que a guarda. Chave desconhecida é erro
// (nunca ignora em silêncio).
func UpdateWhatsapp(db *gorm.DB, tenantID uuid.UUID, id int, fields map[string]interface{}) error {
	cc := map[string]interface{}{}
	det := map[string]interface{}{}
	for k, v := range fields {
		switch {
		case connectionColumns[k] != "":
			cc[connectionColumns[k]] = v
		case detailColumns[k]:
			det[k] = v
		default:
			return fmt.Errorf("campo de conexão WhatsApp desconhecido: %q", k)
		}
	}
	return fresh(db).Transaction(func(tx *gorm.DB) error {
		if len(cc) > 0 {
			if err := tx.Model(&ChannelConnection{}).
				Where(`id = ? AND "tenantId" = ? AND kind = ?`, id, tenantID, KindWhatsApp).Updates(cc).Error; err != nil {
				return err
			}
		}
		if len(det) > 0 {
			if err := tx.Model(&WhatsAppConnectionDetails{}).
				Where(`"connectionId" = ? AND "connectionId" IN (?)`, id, TenantWhatsappIDs(tx, tenantID)).Updates(det).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// UpdateWhatsappDetailsWhere aplica uma atualização em massa nos detalhes das conexões da empresa que casam com cond.
func UpdateWhatsappDetailsWhere(db *gorm.DB, tenantID uuid.UUID, cond string, args []interface{}, fields map[string]interface{}) error {
	return fresh(db).Model(&WhatsAppConnectionDetails{}).
		Where(cond, args...).Where(`"connectionId" IN (?)`, TenantWhatsappIDs(db, tenantID)).Updates(fields).Error
}

// DeleteWhatsapp remove a conexão, os detalhes e as ligações com filas.
func DeleteWhatsapp(db *gorm.DB, tenantID uuid.UUID, id int) error {
	return fresh(db).Transaction(func(tx *gorm.DB) error {
		var owned int64
		if err := tx.Model(&ChannelConnection{}).Where(`id = ? AND "tenantId" = ? AND kind = ?`, id, tenantID, KindWhatsApp).Count(&owned).Error; err != nil || owned == 0 {
			return err
		}
		if err := tx.Exec(`DELETE FROM channel_connection_queues WHERE channel_connection_id = ?`, id).Error; err != nil {
			return err
		}
		if err := tx.Where(`"connectionId" = ?`, id).Delete(&WhatsAppConnectionDetails{}).Error; err != nil {
			return err
		}
		return tx.Where(`id = ? AND "tenantId" = ?`, id, tenantID).Delete(&ChannelConnection{}).Error
	})
}

// SetWhatsappQueues troca as filas da conexão. Só mexe nas ligações das filas informadas e das atuais.
func SetWhatsappQueues(db *gorm.DB, tenantID uuid.UUID, queueID int, whatsappIDs []int) error {
	return fresh(db).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(
			`DELETE FROM channel_connection_queues WHERE queue_id = ? AND channel_connection_id IN (?)`,
			queueID, TenantWhatsappIDs(tx, tenantID)).Error; err != nil {
			return err
		}
		if len(whatsappIDs) == 0 {
			return nil
		}
		return tx.Exec(
			`INSERT INTO channel_connection_queues (queue_id, channel_connection_id)
			 SELECT ?, id FROM "ChannelConnections" WHERE id IN ? AND "tenantId" = ? AND kind = ? ON CONFLICT DO NOTHING`,
			queueID, whatsappIDs, tenantID, KindWhatsApp).Error
	})
}

// SetWhatsappConnectionQueues troca as filas de UMA conexão pelas informadas (só filas da mesma empresa).
func SetWhatsappConnectionQueues(db *gorm.DB, tenantID uuid.UUID, whatsappID int, queueIDs []int) error {
	return fresh(db).Transaction(func(tx *gorm.DB) error {
		var owned int64
		if err := tx.Model(&ChannelConnection{}).Where(`id = ? AND "tenantId" = ? AND kind = ?`, whatsappID, tenantID, KindWhatsApp).Count(&owned).Error; err != nil || owned == 0 {
			return err
		}
		if err := tx.Exec(`DELETE FROM channel_connection_queues WHERE channel_connection_id = ?`, whatsappID).Error; err != nil {
			return err
		}
		if len(queueIDs) == 0 {
			return nil
		}
		return tx.Exec(
			`INSERT INTO channel_connection_queues (channel_connection_id, queue_id)
			 SELECT ?, id FROM "Queues" WHERE id IN ? AND "tenantId" = ? ON CONFLICT DO NOTHING`,
			whatsappID, queueIDs, tenantID).Error
	})
}
