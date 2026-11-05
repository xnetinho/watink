package controllers

import (
	"net/http"
	"strings"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/services"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/alltomatos/watinkdev/business/pkg/utils"
	"github.com/gin-gonic/gin"
)

// SubscribePresence pede ao WhatsApp a presença do contato do ticket. A resposta chega depois, pelo evento
// contactPresence; o WhatsApp só informa se o contato permite (privacidade). Canais sem presença respondem 204.
//
// @Summary      Assinar presença do contato
// @Tags         tickets
// @Param        ticketId  path  int  true  "ID do ticket"
// @Success      202
// @Security     BearerAuth
// @Router       /tickets/{ticketId}/presence [post]
func (mc *MessageController) SubscribePresence(c *gin.Context) {
	db, tenantID, ok := auth.GetScoped(c, "Tickets")
	if !ok {
		return
	}
	var ticket models.Ticket
	if err := db.Preload("Contact").Where("id = ? AND \"tenantId\" = ?", c.Param("ticketId"), tenantID).First(&ticket).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}
	// Só quem o canal expõe tem presença (o Telegram não a dá a bots). Grupo também não.
	conn, ch, found, cerr := channelOf(db, tenantID, ticket)
	watcher, can := ch.(domain.PresenceWatcher)
	if !found || cerr != nil || !can || ticket.Contact.IsGroup {
		c.Status(http.StatusNoContent)
		return
	}
	// O WhatsApp só avisa quando a presença muda: devolve já o último estado conhecido, e o resto chega pelo evento.
	var last *services.PresenceInfo
	for _, id := range []string{jidNumber(contactJID(ticket.Contact)), ticket.Contact.Number} {
		if p, ok := services.LastPresence(tenantID.String(), id, time.Now()); ok {
			last = &p
			break
		}
	}
	if err := watcher.WatchPresence(c.Request.Context(), &conn, contactJID(ticket.Contact)); err != nil {
		utils.RespondWithInternalError(c, err, "SubscribePresence")
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"presence": last})
}

func jidNumber(jid string) string {
	base, _, _ := strings.Cut(jid, "@")
	n, _, _ := strings.Cut(base, ":")
	return n
}
