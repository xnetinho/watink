package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/alltomatos/watinkdev/business/pkg/auth"
	"github.com/alltomatos/watinkdev/business/pkg/utils"
	"github.com/gin-gonic/gin"
)

// themeModes são os únicos valores aceitos para a preferência de tema pessoal.
var themeModes = map[string]bool{"light": true, "dark": true}

// UpdateMyTheme grava a preferência de tema (claro/escuro) do usuário
// autenticado em Users.configs.theme. É auto-serviço, como UpdateMe: o id vem do
// token e só existe uma chave possível, então não há vetor de auto-promoção.
// Merge em vez de overwrite para não apagar outras chaves de configs
// (ex.: dashboard.widgets).
//
// @Summary      Salvar meu tema (claro/escuro)
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body      map[string]string  true  "theme: light|dark"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Security     BearerAuth
// @Router       /me/theme [put]
func (uc *UserController) UpdateMyTheme(c *gin.Context) {
	_, tenantID, ok := auth.GetScoped(c, "Users")
	if !ok {
		return
	}
	uid, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user context"})
		return
	}

	var req struct {
		Theme string `json:"theme"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithBindError(c, err)
		return
	}
	if !themeModes[req.Theme] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "theme must be 'light' or 'dark'"})
		return
	}

	user, err := uc.userRepo.FindByID(c.Request.Context(), uid, tenantID)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	configs := map[string]interface{}{}
	if user.Configs != "" {
		_ = json.Unmarshal([]byte(user.Configs), &configs)
	}
	configs["theme"] = req.Theme
	merged, err := json.Marshal(configs)
	if err != nil {
		utils.RespondWithInternalError(c, err, "UpdateMyTheme")
		return
	}

	if err := uc.userRepo.Update(c.Request.Context(), user, map[string]interface{}{"configs": string(merged)}); err != nil {
		utils.RespondWithInternalError(c, err, "UpdateMyTheme")
		return
	}

	c.JSON(http.StatusOK, gin.H{"theme": req.Theme})
}
