package controllers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// isContactUniqueViolation reconhece a violação de unicidade de número ou LID de
// contato (índices por tenant idx_contacts_tenant_number / idx_contacts_tenant_lid
// e as constraints legadas uni_Contacts_*).
func isContactUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	if !strings.Contains(msg, "duplicate key") && !strings.Contains(msg, "23505") {
		return false
	}
	return strings.Contains(msg, "Contacts_number") || strings.Contains(msg, "contacts_tenant_number") ||
		strings.Contains(msg, "Contacts_lid") || strings.Contains(msg, "contacts_tenant_lid")
}

// respondContactExists devolve 409 com um código estável, em vez de um 500 que o
// usuário lê como "o sistema quebrou" quando só tentou cadastrar um contato repetido.
func respondContactExists(c *gin.Context) {
	c.JSON(http.StatusConflict, gin.H{
		"error": "Já existe um contato com este número nesta conta",
		"code":  "CONTACT_EXISTS",
	})
}
