package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/alltomatos/watinkdev/business/internal/calls"
	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/alltomatos/watinkdev/business/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CallController expõe as chamadas de voz do WhatsApp. As regras vivem em
// calls.Service; aqui só há autenticação, validação de entrada e escopo.
type CallController struct {
	svc *calls.Service
}

func NewCallController(svc *calls.Service) *CallController { return &CallController{svc: svc} }

func callUserID(c *gin.Context) (int, bool) {
	v, ok := c.Get("userId")
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	}
	return 0, false
}

func respondCallError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, calls.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "chamada não encontrada"})
	case errors.Is(err, calls.ErrAlreadyAnswered):
		c.JSON(http.StatusConflict, gin.H{"error": "a chamada já foi atendida", "code": "ALREADY_ANSWERED"})
	case errors.Is(err, calls.ErrConnectionBusy):
		c.JSON(http.StatusConflict, gin.H{"error": "a conexão já tem uma chamada em andamento", "code": "CONNECTION_BUSY"})
	case errors.Is(err, calls.ErrUserBusy):
		c.JSON(http.StatusConflict, gin.H{"error": "você já está em outra chamada", "code": "USER_BUSY"})
	case errors.Is(err, calls.ErrNotActive):
		c.JSON(http.StatusConflict, gin.H{"error": "a chamada não está em andamento", "code": "NOT_ACTIVE"})
	default:
		utils.RespondWithInternalError(c, err, "CallController")
	}
}

// hasProxy diz se a conexão tem proxy configurado. Com proxy a mídia não pode
// sair pelo IP do servidor (ADR 0021), então não há chamada.
func hasProxy(w *models.Whatsapp) bool {
	return (w.ProxyMode != "" && w.ProxyMode != "none") || w.ProxyID != nil || w.ProxyGroupID != nil
}

type placeCallRequest struct {
	TicketID int `json:"ticketId" binding:"required"`
}

// Place godoc
// @Summary      Efetuar chamada de voz
// @Description  Inicia uma chamada de voz 1:1 para o contato de um ticket individual, pela conexão do ticket.
// @Tags         calls
// @Accept       json
// @Produce      json
// @Param        body body placeCallRequest true "Ticket"
// @Success      202  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      409  {object}  map[string]interface{}
// @Failure      422  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /calls [post]
func (cc *CallController) Place(c *gin.Context) {
	db, tenantID, ok := auth.GetScoped(c, "Tickets")
	if !ok {
		return
	}
	userID, ok := callUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "não autenticado"})
		return
	}
	var req placeCallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithBindError(c, err)
		return
	}

	var ticket models.Ticket
	if err := db.Session(&gorm.Session{}).Preload("Contact").Where("id = ?", req.TicketID).First(&ticket).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ticket não encontrado"})
		return
	}
	if ticket.IsGroup || ticket.IsCommunity || ticket.IsSubGroup || ticket.Contact.IsGroup {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "chamadas só estão disponíveis em conversas com um contato (não em grupos)", "code": "NOT_INDIVIDUAL"})
		return
	}
	var wa models.Whatsapp
	if err := dbFresh(db).Where(`id = ? AND "tenantId" = ?`, ticket.WhatsappID, tenantID).First(&wa).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "conexão não encontrada"})
		return
	}
	if wa.Status != "CONNECTED" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "a conexão precisa estar conectada para ligar", "code": "NOT_CONNECTED"})
		return
	}
	if hasProxy(&wa) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "conexões com proxy não fazem chamadas de voz", "code": "PROXY_BLOCKED"})
		return
	}

	log, err := cc.svc.Place(c.Request.Context(), tenantID, userID, &ticket, &ticket.Contact)
	if err != nil {
		respondCallError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, log)
}

func dbFresh(db *gorm.DB) *gorm.DB { return db.Session(&gorm.Session{NewDB: true}) }

// Accept godoc
// @Summary      Atender chamada
// @Tags         calls
// @Produce      json
// @Param        id path string true "ID da chamada"
// @Success      200 {object} models.CallLog
// @Failure      404 {object} map[string]interface{}
// @Failure      409 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /calls/{id}/accept [post]
func (cc *CallController) Accept(c *gin.Context) {
	_, tenantID, ok := auth.GetScoped(c, "CallLogs")
	if !ok {
		return
	}
	userID, ok := callUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "não autenticado"})
		return
	}
	l, err := cc.svc.Accept(c.Request.Context(), tenantID, userID, c.Param("id"))
	if err != nil {
		respondCallError(c, err)
		return
	}
	c.JSON(http.StatusOK, l)
}

// Reject godoc
// @Summary      Recusar chamada
// @Tags         calls
// @Param        id path string true "ID da chamada"
// @Success      204
// @Failure      404 {object} map[string]interface{}
// @Failure      409 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /calls/{id}/reject [post]
func (cc *CallController) Reject(c *gin.Context) {
	_, tenantID, ok := auth.GetScoped(c, "CallLogs")
	if !ok {
		return
	}
	userID, ok := callUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "não autenticado"})
		return
	}
	if err := cc.svc.Reject(c.Request.Context(), tenantID, userID, c.Param("id")); err != nil {
		respondCallError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// End godoc
// @Summary      Encerrar chamada
// @Tags         calls
// @Param        id path string true "ID da chamada"
// @Success      204
// @Failure      404 {object} map[string]interface{}
// @Failure      409 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /calls/{id}/end [post]
func (cc *CallController) End(c *gin.Context) {
	_, tenantID, ok := auth.GetScoped(c, "CallLogs")
	if !ok {
		return
	}
	userID, ok := callUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "não autenticado"})
		return
	}
	if err := cc.svc.End(c.Request.Context(), tenantID, userID, c.Param("id")); err != nil {
		respondCallError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type pauseRequest struct {
	Paused bool `json:"paused"`
}

// Pause godoc
// @Summary      Pausar ou retomar o recebimento de chamadas
// @Description  Avisa o servidor que este operador pausou o toque, para ele não contar como elegível.
// @Tags         calls
// @Accept       json
// @Param        body body pauseRequest true "Estado"
// @Success      204
// @Security     BearerAuth
// @Router       /calls/pause [put]
func (cc *CallController) Pause(c *gin.Context) {
	tenantID, err := auth.TenantUUIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant inválido"})
		return
	}
	userID, ok := callUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "não autenticado"})
		return
	}
	var req pauseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithBindError(c, err)
		return
	}
	cc.svc.SetPaused(tenantID, userID, req.Paused)
	c.Status(http.StatusNoContent)
}

// historyScope monta o escopo do histórico: alcance de empresa vê tudo da
// empresa; o resto só as chamadas de tickets que enxerga.
func historyScope(c *gin.Context, tenantID uuid.UUID, userID int) calls.HistoryScope {
	alcance, _ := c.Get("alcance")
	a, _ := alcance.(string)
	return calls.HistoryScope{TenantID: tenantID, UserID: userID, Tenant: a == "tenant" || a == "plataforma"}
}

// List godoc
// @Summary      Histórico de chamadas
// @Description  Lista as chamadas da empresa (ou só as de tickets visíveis ao usuário, sem alcance de empresa).
// @Tags         calls
// @Produce      json
// @Param        status    query string false "ringing|active|ended|missed|rejected|failed|interrupted"
// @Param        direction query string false "incoming|outgoing"
// @Param        ticketId  query int    false "Filtrar por ticket"
// @Param        page      query int    false "Página"
// @Param        pageSize  query int    false "Itens por página (máx. 100)"
// @Success      200 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /calls [get]
func (cc *CallController) List(c *gin.Context) {
	tenantID, err := auth.TenantUUIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant inválido"})
		return
	}
	userID, _ := callUserID(c)
	f := calls.HistoryFilter{Status: c.Query("status"), Direction: c.Query("direction")}
	f.TicketID, _ = strconv.Atoi(c.Query("ticketId"))
	f.Page, _ = strconv.Atoi(c.Query("page"))
	f.PageSize, _ = strconv.Atoi(c.Query("pageSize"))

	rows, total, err := cc.svc.History(historyScope(c, tenantID, userID), f)
	if err != nil {
		utils.RespondWithInternalError(c, err, "CallController.List")
		return
	}
	c.JSON(http.StatusOK, gin.H{"calls": rows, "total": total})
}

// Show godoc
// @Summary      Detalhe de uma chamada
// @Tags         calls
// @Produce      json
// @Param        id path string true "ID da chamada"
// @Success      200 {object} models.CallLog
// @Failure      404 {object} map[string]interface{}
// @Security     BearerAuth
// @Router       /calls/{id} [get]
func (cc *CallController) Show(c *gin.Context) {
	tenantID, err := auth.TenantUUIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant inválido"})
		return
	}
	userID, _ := callUserID(c)
	l, err := cc.svc.Get(historyScope(c, tenantID, userID), c.Param("id"))
	if err != nil {
		respondCallError(c, err)
		return
	}
	c.JSON(http.StatusOK, l)
}
