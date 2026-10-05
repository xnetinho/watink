package controllers

import (
	"errors"
	"net/http"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/alltomatos/watinkdev/business/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type createTicketRequest struct {
	ContactID  int    `json:"contactId" binding:"required"`
	WhatsappID *int   `json:"whatsappId"`
	QueueID    *int   `json:"queueId"`
	Status     string `json:"status"`
}

type connectionOption struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Number string `json:"number"`
}

var errNoConnectedConnection = errors.New("no connected connection")

type errConnectionRequired struct{ options []connectionOption }

func (errConnectionRequired) Error() string { return "connection required" }

// pickConnection resolve a conexão de um novo ticket: a preferida do usuário
// (se conectada), senão a única conectada; 0 conectadas → errNoConnectedConnection;
// 2+ conectadas → errConnectionRequired (o frontend pergunta qual usar).
func pickConnection(connected []models.Whatsapp, userPreferred *int) (int, error) {
	if userPreferred != nil {
		for _, w := range connected {
			if w.ID == *userPreferred {
				return w.ID, nil
			}
		}
	}
	switch len(connected) {
	case 0:
		return 0, errNoConnectedConnection
	case 1:
		return connected[0].ID, nil
	}
	opts := make([]connectionOption, 0, len(connected))
	for _, w := range connected {
		opts = append(opts, connectionOption{ID: w.ID, Name: w.Name, Number: w.Number})
	}
	return 0, errConnectionRequired{options: opts}
}

// @Summary      Iniciar conversa (criar ticket)
// @Description  Abre um ticket para um contato. Se já existir ticket aberto/pendente do contato na mesma conexão, devolve o existente (200). Sem whatsappId, usa a conexão do usuário ou a única conectada; com 2+ conectadas responde 409 code=CONNECTION_REQUIRED listando as opções.
// @Tags         tickets
// @Accept       json
// @Produce      json
// @Param        body  body      createTicketRequest  true  "contactId obrigatório"
// @Success      201   {object}  map[string]interface{}
// @Success      200   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]string
// @Failure      409   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /tickets [post]
func (tc *TicketController) CreateTicket(c *gin.Context) {
	db, tenantID, ok := auth.GetScoped(c, "Tickets")
	if !ok {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req createTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "contactId é obrigatório"})
		return
	}
	status := "open"
	if req.Status == "pending" {
		status = "pending"
	}

	q := func() *gorm.DB { return db.Session(&gorm.Session{NewDB: true}) }

	var contact models.Contact
	if err := q().Where(`id = ? AND "tenantId" = ?`, req.ContactID, tenantID).First(&contact).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contato não encontrado"})
		return
	}

	whatsappID, ok := tc.resolveConnection(c, q, tenantID, userID, req.WhatsappID)
	if !ok {
		return
	}

	var existing models.Ticket
	err := q().Where(`"tenantId" = ? AND "contactId" = ? AND "whatsappId" = ? AND status IN ?`,
		tenantID, contact.ID, whatsappID, []string{"open", "pending"}).First(&existing).Error
	if err == nil {
		var visible int64
		db.Model(&models.Ticket{}).Where("id = ?", existing.ID).Count(&visible)
		if visible == 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "Este contato já está em atendimento por outro atendente", "code": "TICKET_OWNED_BY_OTHER"})
			return
		}
		if err := q().Preload("Contact").Preload("User").First(&existing, existing.ID).Error; err != nil {
			utils.RespondWithInternalError(c, err, "CreateTicket")
			return
		}
		c.JSON(http.StatusOK, existing)
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		utils.RespondWithInternalError(c, err, "CreateTicket")
		return
	}

	queueID, err := tc.resolveQueue(q, tenantID, userID, req.QueueID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Fila não encontrada"})
		return
	}

	ticket := models.Ticket{
		Status:     status,
		ContactID:  contact.ID,
		UserID:     &userID,
		WhatsappID: whatsappID,
		IsGroup:    contact.IsGroup,
		QueueID:    queueID,
		TenantID:   tenantID,
	}
	if err := q().Create(&ticket).Error; err != nil {
		utils.RespondWithInternalError(c, err, "CreateTicket")
		return
	}
	if err := q().Preload("Contact").Preload("User").First(&ticket, ticket.ID).Error; err != nil {
		utils.RespondWithInternalError(c, err, "CreateTicket")
		return
	}

	tc.broadcast.EmitToTenantRoom(tenantID.String(), "ticket", gin.H{"action": "update", "ticket": ticket})
	c.JSON(http.StatusCreated, ticket)
}

// resolveConnection devolve o id da conexão a usar; ok=false significa que a
// resposta de erro já foi escrita.
func (tc *TicketController) resolveConnection(c *gin.Context, q func() *gorm.DB, tenantID uuid.UUID, userID int, requested *int) (int, bool) {
	var connected []models.Whatsapp
	if err := q().Where(`"tenantId" = ? AND status = ?`, tenantID, "CONNECTED").Order("id").Find(&connected).Error; err != nil {
		utils.RespondWithInternalError(c, err, "CreateTicket")
		return 0, false
	}

	if requested != nil {
		for _, w := range connected {
			if w.ID == *requested {
				return w.ID, true
			}
		}
		c.JSON(http.StatusConflict, gin.H{"error": "A conexão escolhida não está conectada", "code": "CONNECTION_NOT_CONNECTED"})
		return 0, false
	}

	var preferred *int
	var u models.User
	if err := q().Select("id", "whatsappId").Where(`id = ? AND "tenantId" = ?`, userID, tenantID).First(&u).Error; err == nil {
		preferred = u.WhatsappID
	}

	id, err := pickConnection(connected, preferred)
	var need errConnectionRequired
	switch {
	case err == nil:
		return id, true
	case errors.Is(err, errNoConnectedConnection):
		c.JSON(http.StatusConflict, gin.H{"error": "Nenhuma conexão do WhatsApp está conectada. Conecte uma conexão para iniciar conversas.", "code": "NO_CONNECTED_CONNECTION"})
	case errors.As(err, &need):
		c.JSON(http.StatusConflict, gin.H{"error": "Escolha por qual conexão iniciar a conversa", "code": "CONNECTION_REQUIRED", "connections": need.options})
	default:
		utils.RespondWithInternalError(c, err, "CreateTicket")
	}
	return 0, false
}

// resolveQueue valida a fila informada ou, sem ela, usa a fila do usuário se
// ele tiver exatamente uma.
func (tc *TicketController) resolveQueue(q func() *gorm.DB, tenantID uuid.UUID, userID int, requested *int) (*int, error) {
	if requested != nil {
		var queue models.Queue
		if err := q().Where(`id = ? AND "tenantId" = ?`, *requested, tenantID).First(&queue).Error; err != nil {
			return nil, err
		}
		return requested, nil
	}
	var ids []int
	if err := q().Raw(`SELECT queue_id FROM user_queues WHERE user_id = ?`, userID).Scan(&ids).Error; err != nil {
		return nil, nil
	}
	if len(ids) == 1 {
		return &ids[0], nil
	}
	return nil, nil
}
