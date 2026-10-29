package controllers

import (
	"fmt"
	wbot "github.com/alltomatos/watinkdev/contracts/wbot"
	"net/http"
	"strconv"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/alltomatos/watinkdev/business/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// @Summary      Importar contatos do WhatsApp
// @Description  Dispara a importação dos contatos de UMA conexão WhatsApp (connectionId no corpo)
// @Tags         contacts
// @Produce      json
// @Param        body  body      map[string]interface{}  true  "connectionId"
// @Success      202  {object}  map[string]string
// @Failure      409  {object}  map[string]string
// @Security     BearerAuth
// @Router       /contacts/import [post]
func (cc *ContactController) ImportContacts(c *gin.Context) {
	db, tenantID, ok := auth.GetScoped(c, "Contacts")
	if !ok {
		return
	}

	var input struct {
		ConnectionID int `json:"connectionId"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithBindError(c, err)
		return
	}
	// A importação é de UMA conexão WhatsApp, escolhida por quem pede; nunca "a primeira conectada".
	conn, found := resolveConnection(db, tenantID, domain.ConnectionSourceWhatsapp, input.ConnectionID)
	if !found {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "connection_required"})
		return
	}
	if conn.Status != "CONNECTED" {
		c.JSON(http.StatusConflict, gin.H{"error": "connection_not_connected"})
		return
	}

	command := map[string]interface{}{
		"id":        uuid.New().String(),
		"timestamp": time.Now().UnixMilli(),
		"tenantId":  tenantID.String(),
		"type":      "contact.import",
		"payload": map[string]interface{}{
			"sessionId": conn.Ref.ID,
		},
	}
	routingKey := wbot.Command(tenantID.String(), strconv.Itoa(conn.Ref.ID), wbot.CommandContactImport)
	if err := cc.publisher.PublishCommand(routingKey, command); err != nil {
		utils.RespondWithInternalError(c, err, "ImportContacts")
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "Contact import started"})
}

// @Summary      Sincronizar foto do contato via WhatsApp
// @Description  Busca a foto de perfil atualizada do contato; publica a mensagem assíncrona via AMQP e responde 202.
// @Tags         contacts
// @Produce      json
// @Param        contactId  path      int  true  "ID do contato"
// @Success      202        {object}  map[string]string
// @Failure      404        {object}  map[string]string
// @Failure      409        {object}  map[string]string
// @Security     BearerAuth
// @Router       /contacts/{contactId}/sync [post]
func (cc *ContactController) SyncContact(c *gin.Context) {
	_, tenantID, ok := auth.GetScoped(c, "Contacts")
	if !ok {
		return
	}
	id, ok := utils.ParseIntParam(c, "contactId")
	if !ok {
		return
	}

	contact, err := cc.contactRepo.FindByID(c.Request.Context(), id, tenantID)
	if err != nil {
		utils.RespondWithInternalError(c, err, "SyncContact")
		return
	}
	if contact == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contact not found"})
		return
	}

	sessions, err := cc.sessions.FindAll(c.Request.Context(), tenantID)
	if err != nil {
		utils.RespondWithInternalError(c, err, "SyncContact")
		return
	}
	var session *domain.ChannelSession
	for i := range sessions {
		if sessions[i].Status == "CONNECTED" {
			session = &sessions[i]
			break
		}
	}
	if session == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "No connected WhatsApp session available"})
		return
	}

	command := map[string]interface{}{
		"id":        uuid.New().String(),
		"timestamp": time.Now().UnixMilli(),
		"tenantId":  tenantID.String(),
		"type":      "contact.sync",
		"payload": map[string]interface{}{
			"sessionId": fmt.Sprintf("%d", session.ID),
			"number":    contact.Number,
		},
	}
	routingKey := wbot.Command(tenantID.String(), strconv.Itoa(session.ID), wbot.CommandContactSync)
	if err := cc.publisher.PublishCommand(routingKey, command); err != nil {
		utils.RespondWithInternalError(c, err, "SyncContact")
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "Contact sync requested"})
}
