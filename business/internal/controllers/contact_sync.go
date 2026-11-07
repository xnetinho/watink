package controllers

import (
	"net/http"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/alltomatos/watinkdev/business/pkg/utils"
	"github.com/gin-gonic/gin"
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
	conn, found := resolveConnection(db, tenantID, input.ConnectionID)
	if !found {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "connection_required"})
		return
	}
	if conn.Status != "CONNECTED" {
		c.JSON(http.StatusConflict, gin.H{"error": "connection_not_connected"})
		return
	}

	cconn, ch, found, cerr := channelOfConnection(db, tenantID, conn.ID)
	syncer, can := ch.(domain.ContactSyncer)
	if !found || cerr != nil || !can {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "contact_import_not_supported"})
		return
	}
	if err := syncer.ImportContacts(c.Request.Context(), &cconn); err != nil {
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
	db, tenantID, ok := auth.GetScoped(c, "Contacts")
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

	cconn, ch, found, cerr := channelOfConnection(db, tenantID, session.ID)
	syncer, can := ch.(domain.ContactSyncer)
	if !found || cerr != nil || !can {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "contact_sync_not_supported"})
		return
	}
	if err := syncer.SyncContact(c.Request.Context(), &cconn, contact.Number); err != nil {
		utils.RespondWithInternalError(c, err, "SyncContact")
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "Contact sync requested"})
}
