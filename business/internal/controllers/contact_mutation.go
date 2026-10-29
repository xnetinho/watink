package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/alltomatos/watinkdev/business/pkg/utils"
	wbot "github.com/alltomatos/watinkdev/contracts/wbot"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// @Summary      Criar contato
// @Tags         contacts
// @Accept       json
// @Produce      json
// @Param        body  body      map[string]interface{}  true  "Dados do contato"
// @Success      200   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /contacts [post]
func (cc *ContactController) CreateContact(c *gin.Context) {
	db, tenantID, ok := auth.GetScoped(c, "Contacts")
	if !ok {
		return
	}

	var input struct {
		// Conexão de origem, obrigatória: o contato pertence a uma conexão e é por ela que recebe mensagens.
		ConnectionSource string `json:"connectionSource"`
		ConnectionID     int    `json:"connectionId"`
		Name             string `json:"name"`
		Number           string `json:"number"`
		ProfilePicUrl    string `json:"profilePicUrl"`
		Email            string `json:"email"`
		IsGroup          bool   `json:"isGroup"`
		WalletUserID     *int   `json:"walletUserId"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithBindError(c, err)
		return
	}

	name, err := utils.ValidateStringField(input.Name, "name", 255)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	number, err := utils.ValidateStringField(input.Number, "number", 50)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	profilePicUrl, err := utils.ValidateStringField(input.ProfilePicUrl, "profilePicUrl", 2048)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	email, err := utils.ValidateStringField(input.Email, "email", 255)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	conn, found := resolveConnection(db, tenantID, input.ConnectionSource, input.ConnectionID)
	if !found {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "connection_required"})
		return
	}

	contact := &domain.Contact{
		ConnectionSource: conn.Ref.Source,
		ConnectionID:     conn.Ref.ID,
		Name:             name,
		Number:           number,
		ProfilePicUrl:    profilePicUrl,
		Email:            email,
		IsGroup:          input.IsGroup,
		WalletUserID:     input.WalletUserID,
		TenantID:         tenantID,
	}

	// Contato de WhatsApp criado aqui também vai para a agenda do celular, como o WhatsApp Web faz. Conexão fora do ar
	// não impede o cadastro: o contato nasce e "failed" diz que o celular não o recebeu.
	// Um contato criado à mão só faz sentido onde a empresa PODE iniciar a conversa. O bot do Telegram só responde a quem
	// escreveu primeiro (Capabilities.InitiateConversation=false), então não há o que cadastrar: o contato dele nasce
	// quando a pessoa fala com o bot.
	if ad, err := domain.DefaultChannelRegistry().Get(domain.ChannelKind(conn.Kind)); err == nil && !ad.Capabilities().InitiateConversation {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "channel_cannot_initiate"})
		return
	}

	toPhone := conn.Kind == string(domain.ChannelKindWhatsApp) && !input.IsGroup && number != ""
	if toPhone {
		contact.PhoneSync = "pending"
		if conn.Status != "CONNECTED" || cc.publisher == nil {
			contact.PhoneSync, contact.PhoneSyncError = "failed", "connection_not_connected"
		}
	}

	if err := cc.contactRepo.Create(c.Request.Context(), contact); err != nil {
		if isContactUniqueViolation(err) {
			respondContactExists(c)
			return
		}
		utils.RespondWithInternalError(c, err, "CreateContact")
		return
	}

	if toPhone && contact.PhoneSync == "pending" {
		cmd := map[string]interface{}{
			"id": uuid.New().String(), "timestamp": time.Now().UnixMilli(), "tenantId": tenantID.String(), "type": wbot.CommandContactSave,
			"payload": map[string]interface{}{"sessionId": conn.Ref.ID, "number": number, "name": name},
		}
		key := wbot.Command(tenantID.String(), strconv.Itoa(conn.Ref.ID), wbot.CommandContactSave)
		if err := cc.publisher.PublishCommand(key, cmd); err != nil {
			contact.PhoneSync, contact.PhoneSyncError = "failed", "publish_failed"
			_ = cc.contactRepo.Update(c.Request.Context(), contact, map[string]interface{}{"phoneSync": contact.PhoneSync, "phoneSyncError": contact.PhoneSyncError})
		}
	}

	cc.broadcast.EmitToTenantRoom(tenantID.String(), "contact", gin.H{"action": "create", "contact": contact})

	c.JSON(http.StatusOK, contact)
}

// @Summary      Atualizar contato
// @Tags         contacts
// @Accept       json
// @Produce      json
// @Param        contactId  path      int                     true  "ID do contato"
// @Param        body       body      map[string]interface{}  true  "Campos a atualizar"
// @Success      200        {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /contacts/{contactId} [put]
func (cc *ContactController) UpdateContact(c *gin.Context) {
	_, tenantID, ok := auth.GetScoped(c, "Contacts")
	if !ok {
		return
	}
	id, _ := strconv.Atoi(c.Param("contactId"))

	var input struct {
		Name          string `json:"name"`
		Number        string `json:"number"`
		ProfilePicUrl string `json:"profilePicUrl"`
		Email         string `json:"email"`
		WalletUserID  *int   `json:"walletUserId"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithBindError(c, err)
		return
	}

	contact, err := cc.contactRepo.FindByID(c.Request.Context(), id, tenantID)
	if err != nil || contact == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contact not found"})
		return
	}

	fields := map[string]interface{}{}
	if input.Name != "" {
		name, err := utils.ValidateStringField(input.Name, "name", 255)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		fields["name"] = name
	}
	if input.Number != "" {
		number, err := utils.ValidateStringField(input.Number, "number", 50)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		fields["number"] = number
	}
	if input.ProfilePicUrl != "" {
		profilePicUrl, err := utils.ValidateStringField(input.ProfilePicUrl, "profilePicUrl", 2048)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		fields["profilePicUrl"] = profilePicUrl
	}
	if input.Email != "" {
		email, err := utils.ValidateStringField(input.Email, "email", 255)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		fields["email"] = email
	}
	if input.WalletUserID != nil {
		fields["walletUserId"] = input.WalletUserID
	}

	if err := cc.contactRepo.Update(c.Request.Context(), contact, fields); err != nil {
		if isContactUniqueViolation(err) {
			respondContactExists(c)
			return
		}
		utils.RespondWithInternalError(c, err, "UpdateContact")
		return
	}

	if updated, err := cc.contactRepo.FindByID(c.Request.Context(), id, tenantID); err == nil && updated != nil {
		contact = updated
	}
	cc.broadcast.EmitToTenantRoom(tenantID.String(), "contact", gin.H{"action": "update", "contact": contact})

	c.JSON(http.StatusOK, contact)
}

// @Summary      Remover contato
// @Tags         contacts
// @Produce      json
// @Param        contactId  path      int  true  "ID do contato"
// @Success      200        {object}  map[string]string
// @Security     BearerAuth
// @Router       /contacts/{contactId} [delete]
func (cc *ContactController) DeleteContact(c *gin.Context) {
	_, tenantID, ok := auth.GetScoped(c, "Contacts")
	if !ok {
		return
	}
	id, _ := strconv.Atoi(c.Param("contactId"))

	if err := cc.contactRepo.Delete(c.Request.Context(), id, tenantID); err != nil {
		utils.RespondWithInternalError(c, err, "DeleteContact")
		return
	}

	cc.broadcast.EmitToTenantRoom(tenantID.String(), "contact", gin.H{"action": "delete", "contactId": strconv.Itoa(id)})

	c.JSON(http.StatusOK, gin.H{"message": "Contact deleted successfully"})
}

// @Summary      Excluir contatos selecionados
// @Tags         contacts
// @Accept       json
// @Produce      json
// @Param        body  body      map[string]interface{}  true  "Lista de IDs a excluir"
// @Success      200   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /contacts/bulk-delete [post]
func (cc *ContactController) BulkDeleteContacts(c *gin.Context) {
	_, tenantID, ok := auth.GetScoped(c, "Contacts")
	if !ok {
		return
	}

	var input struct {
		IDs []int `json:"ids"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithBindError(c, err)
		return
	}
	if len(input.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids is required and must not be empty"})
		return
	}

	deleted, err := cc.contactRepo.BulkDelete(c.Request.Context(), input.IDs, tenantID)
	if err != nil {
		utils.RespondWithInternalError(c, err, "BulkDeleteContacts")
		return
	}

	cc.broadcast.EmitToTenantRoom(tenantID.String(), "contact", gin.H{"action": "bulk-delete", "contactIds": input.IDs})

	c.JSON(http.StatusOK, gin.H{"deleted": deleted})
}

// @Summary      Excluir todos os contatos do tenant
// @Tags         contacts
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /contacts/all [delete]
func (cc *ContactController) DeleteAllContacts(c *gin.Context) {
	_, tenantID, ok := auth.GetScoped(c, "Contacts")
	if !ok {
		return
	}

	deleted, err := cc.contactRepo.DeleteAll(c.Request.Context(), tenantID)
	if err != nil {
		utils.RespondWithInternalError(c, err, "DeleteAllContacts")
		return
	}

	cc.broadcast.EmitToTenantRoom(tenantID.String(), "contact", gin.H{"action": "delete-all"})

	c.JSON(http.StatusOK, gin.H{"deleted": deleted})
}
