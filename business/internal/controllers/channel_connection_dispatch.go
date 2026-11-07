package controllers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ConnectionRouter expõe as conexões de TODOS os canais sob /channels/connections (fatia 5 da migração). Ele não tem regra de
// negócio: olha o `kind` da conexão e entrega ao controller do canal (WhatsApp ou os demais). As rotas /whatsapp/* seguem
// atendendo, com o mesmo corpo, até a fatia 6.
type ConnectionRouter struct {
	whatsapp *WhatsappController
	sessions *SessionController
	others   *ChannelConnectionController
}

func NewConnectionRouter(w *WhatsappController, s *SessionController, o *ChannelConnectionController) *ConnectionRouter {
	return &ConnectionRouter{whatsapp: w, sessions: s, others: o}
}

// kindOf devolve o canal da conexão :id (vazio se não existe nesta empresa).
func kindOf(c *gin.Context) (string, bool) {
	db, tenantID, ok := auth.GetScoped(c, "ChannelConnections")
	if !ok {
		return "", false
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid connection id"})
		return "", false
	}
	var conn models.ChannelConnection
	if err := db.Session(&gorm.Session{NewDB: true}).Select("id", "kind").
		Where(`id = ? AND "tenantId" = ?`, id, tenantID).First(&conn).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "connection not found"})
		return "", false
	}
	return conn.Kind, true
}

func notSupported(c *gin.Context) {
	c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "not_supported_by_channel"})
}

// whatsappOnly roda h só se a conexão :id for do WhatsApp; senão 422.
func (r *ConnectionRouter) whatsappOnly(h gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		kind, ok := kindOf(c)
		if !ok {
			return
		}
		if kind != models.KindWhatsApp {
			notSupported(c)
			return
		}
		h(c)
	}
}

// byKind roda hw para WhatsApp e ho para os demais canais.
func (r *ConnectionRouter) byKind(hw, ho gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		kind, ok := kindOf(c)
		if !ok {
			return
		}
		if kind == models.KindWhatsApp {
			hw(c)
			return
		}
		ho(c)
	}
}

// Create cria a conexão pelo controller do canal: kind=whatsapp usa o do WhatsApp (sem token), os demais exigem token.
func (r *ConnectionRouter) Create(c *gin.Context) {
	raw, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	var probe struct {
		Kind string `json:"kind"`
	}
	_ = json.Unmarshal(raw, &probe)
	c.Request.Body = io.NopCloser(bytes.NewReader(raw))
	if probe.Kind == models.KindWhatsApp {
		r.whatsapp.CreateWhatsapp(c)
		return
	}
	r.others.CreateConnection(c)
}

func (r *ConnectionRouter) Get() gin.HandlerFunc {
	return r.byKind(r.whatsapp.ShowWhatsapp, r.others.GetConnection)
}
func (r *ConnectionRouter) Update() gin.HandlerFunc {
	return r.byKind(r.whatsapp.UpdateWhatsapp, r.others.UpdateConnection)
}
func (r *ConnectionRouter) Delete() gin.HandlerFunc {
	return r.byKind(r.whatsapp.DeleteWhatsapp, r.others.DeleteConnection)
}
func (r *ConnectionRouter) Stats() gin.HandlerFunc { return r.whatsappOnly(r.whatsapp.StatsWhatsapp) }
func (r *ConnectionRouter) KeepAlive() gin.HandlerFunc {
	return r.whatsappOnly(r.whatsapp.ToggleKeepAlive)
}
func (r *ConnectionRouter) PresencePolicy() gin.HandlerFunc {
	return r.whatsappOnly(r.whatsapp.UpdatePresencePolicy)
}

// StartSession e StopSession usam o parâmetro :id (as rotas antigas usavam :whatsappId).
func (r *ConnectionRouter) StartSession() gin.HandlerFunc {
	return r.whatsappOnly(aliasParam("id", "whatsappId", r.sessions.StartSession))
}
func (r *ConnectionRouter) StopSession() gin.HandlerFunc {
	return r.whatsappOnly(aliasParam("id", "whatsappId", r.sessions.StopSession))
}
func (r *ConnectionRouter) RestartAll() gin.HandlerFunc { return r.sessions.RestartAllSessions }

// aliasParam copia o parâmetro de rota `from` para `to`, para reaproveitar handlers escritos com o nome antigo.
func aliasParam(from, to string, h gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Params = append(c.Params, gin.Param{Key: to, Value: c.Param(from)})
		h(c)
	}
}

// List devolve as conexões de todos os canais numa lista só. O WhatsApp vem na visão achatada (identidade + detalhes, com os aliases
// temporários number/profilePicUrl/firstConnection); os demais canais, como o ChannelConnection. Ambos levam connectionId.
func (r *ConnectionRouter) List(c *gin.Context) {
	db, tenantID, ok := auth.GetScoped(c, "ChannelConnections")
	if !ok {
		return
	}
	var conns []models.ChannelConnection
	if err := db.Session(&gorm.Session{NewDB: true}).Where(`"tenantId" = ?`, tenantID).Order("id").Find(&conns).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list_failed"})
		return
	}
	wa, err := r.whatsapp.sessionRepo.FindAll(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list_failed"})
		return
	}
	waByID := make(map[int]interface{}, len(wa))
	for i := range wa {
		waByID[wa[i].ID] = wa[i]
	}
	out := make([]interface{}, 0, len(conns))
	for _, cn := range conns {
		if cn.Kind == models.KindWhatsApp {
			if v, ok := waByID[cn.ID]; ok {
				out = append(out, withKind(v, cn))
				continue
			}
		}
		out = append(out, connectionAliases(cn))
	}
	c.JSON(http.StatusOK, out)
}

// withKind acrescenta `kind`, `identifier` e os aliases de transição ao JSON da visão achatada do WhatsApp.
func withKind(v interface{}, cn models.ChannelConnection) gin.H {
	raw, _ := json.Marshal(v)
	m := gin.H{}
	_ = json.Unmarshal(raw, &m)
	m["kind"] = cn.Kind
	m["connectionId"] = cn.ID
	m["identifier"] = cn.Identifier
	m["displayName"] = cn.DisplayName
	m["avatarUrl"] = cn.AvatarURL
	m["firstConnectedAt"] = cn.FirstConnectedAt
	return m
}

// connectionAliases devolve o canal não-WhatsApp com connectionId e os aliases number/profilePicUrl/firstConnection.
func connectionAliases(cn models.ChannelConnection) gin.H {
	raw, _ := json.Marshal(cn)
	m := gin.H{}
	_ = json.Unmarshal(raw, &m)
	m["connectionId"] = cn.ID
	m["number"] = cn.Identifier
	m["profilePicUrl"] = cn.AvatarURL
	m["firstConnection"] = cn.FirstConnectedAt
	return m
}
