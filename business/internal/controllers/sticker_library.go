package controllers

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/alltomatos/watinkdev/business/pkg/mediastore"
	"github.com/alltomatos/watinkdev/business/pkg/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// StickerLibraryController guarda as figurinhas da EMPRESA: qualquer atendente salva e remove, todos os atendentes e todas as
// conexões WhatsApp enxergam as mesmas. Sem limite de itens.
type StickerLibraryController struct{}

func NewStickerLibraryController() *StickerLibraryController { return &StickerLibraryController{} }

// List devolve a biblioteca da empresa, a mais recente primeiro.
//
// @Summary      Listar figurinhas da biblioteca
// @Tags         stickers
// @Produce      json
// @Success      200  {array}  models.StickerLibraryItem
// @Security     BearerAuth
// @Router       /stickers [get]
func (ctrl *StickerLibraryController) List(c *gin.Context) {
	db, tenantID, ok := auth.GetScoped(c, "StickerLibrary")
	if !ok {
		return
	}
	var items []models.StickerLibraryItem
	if err := db.Session(&gorm.Session{NewDB: true}).Where(`"tenantId" = ?`, tenantID).Order(`"createdAt" DESC, id DESC`).Find(&items).Error; err != nil {
		utils.RespondWithInternalError(c, err, "ListStickers")
		return
	}
	c.JSON(http.StatusOK, items)
}

// Save guarda na biblioteca a figurinha de uma mensagem (recebida ou enviada). A mensagem precisa ser do tipo figurinha e ter o
// arquivo guardado; o que vai para a biblioteca é o arquivo que a mensagem já tem, nunca algo que o cliente HTTP mande. Salvar a
// mesma figurinha de novo (mesmo conteúdo) devolve a que já existe, sem duplicar.
//
// @Summary      Salvar figurinha na biblioteca
// @Tags         stickers
// @Accept       json
// @Produce      json
// @Success      200  {object}  savedSticker
// @Security     BearerAuth
// @Router       /stickers [post]
func (ctrl *StickerLibraryController) Save(c *gin.Context) {
	db, tenantID, ok := auth.GetScoped(c, "Messages")
	if !ok {
		return
	}
	var body struct {
		MessageID string `json:"messageId"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.MessageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	var msg models.Message
	if err := db.Session(&gorm.Session{NewDB: true}).Where(`id = ? AND "tenantId" = ?`, body.MessageID, tenantID).First(&msg).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}
	if msg.IsDeleted || msg.MediaType != "sticker" || msg.MediaUrl == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "message_not_a_sticker"})
		return
	}
	data, local, err := mediastore.ReadLocal(msg.MediaUrl)
	if !local || err != nil || len(data) == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "media_unavailable"})
		return
	}
	sum := sha256.Sum256(data)
	item := models.StickerLibraryItem{
		TenantID: tenantID,
		Hash:     hex.EncodeToString(sum[:]),
		MediaUrl: msg.MediaUrl,
		Animated: dataFieldBool(msg.DataJson, "animated"),
	}
	if uid, ok := currentUserID(c); ok && uid > 0 {
		item.SavedBy = &uid
	}
	writeDB := db.Session(&gorm.Session{NewDB: true})
	// A figurinha pode já existir (mesmo conteúdo): não duplica, e a resposta é a que está guardada. RowsAffected diz se esta
	// chamada foi quem inseriu; se ninguém inseriu, ela já estava na biblioteca e a tela avisa isso.
	res := writeDB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "tenantId"}, {Name: "hash"}}, DoNothing: true}).Create(&item)
	if res.Error != nil {
		utils.RespondWithInternalError(c, res.Error, "SaveSticker")
		return
	}
	alreadySaved := res.RowsAffected == 0
	var saved models.StickerLibraryItem
	if err := writeDB.Where(`"tenantId" = ? AND hash = ?`, tenantID, item.Hash).First(&saved).Error; err != nil {
		utils.RespondWithInternalError(c, err, "SaveSticker.Reload")
		return
	}
	c.JSON(http.StatusOK, savedSticker{StickerLibraryItem: saved, AlreadySaved: alreadySaved})
}

// savedSticker é a resposta de "salvar": o item da biblioteca (mesmo JSON do modelo) mais se ele já estava lá.
type savedSticker struct {
	models.StickerLibraryItem
	AlreadySaved bool `json:"alreadySaved"`
}

// Delete tira a figurinha da biblioteca da empresa. O arquivo continua no armazenamento (outras mensagens podem apontar para ele).
//
// @Summary      Remover figurinha da biblioteca
// @Tags         stickers
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /stickers/{stickerId} [delete]
func (ctrl *StickerLibraryController) Delete(c *gin.Context) {
	db, tenantID, ok := auth.GetScoped(c, "StickerLibrary")
	if !ok {
		return
	}
	id, err := strconv.Atoi(c.Param("stickerId"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	res := db.Session(&gorm.Session{NewDB: true}).Where(`id = ? AND "tenantId" = ?`, id, tenantID).Delete(&models.StickerLibraryItem{})
	if res.Error != nil {
		utils.RespondWithInternalError(c, res.Error, "DeleteSticker")
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sticker not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Sticker removed"})
}
