package utils_test

import (
	"testing"

	"github.com/alltomatos/watinkdev/business/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestContactJID(t *testing.T) {
	lid := "123456789@lid"
	emptyLid := ""

	assert.Equal(t, "120363001@g.us", utils.ContactJID("120363001", nil, true))
	assert.Equal(t, "120363001@g.us", utils.ContactJID("120363001@g.us", nil, true))
	assert.Equal(t, "123456789@lid", utils.ContactJID("5511999999999", &lid, false))
	assert.Equal(t, "5511999999999", utils.ContactJID("5511999999999", &emptyLid, false))
	assert.Equal(t, "5511999999999", utils.ContactJID("5511999999999", nil, false))
}

func TestUserJID(t *testing.T) {
	assert.Equal(t, "5511999999999@s.whatsapp.net", utils.UserJID("5511999999999"))
	assert.Equal(t, "5511999999999@s.whatsapp.net", utils.UserJID("5511999999999@s.whatsapp.net"))
}

func TestExtractNumberFromJID(t *testing.T) {
	assert.Equal(t, "5511999999999", utils.ExtractNumberFromJID("5511999999999@s.whatsapp.net"))
	assert.Equal(t, "5511999999999", utils.ExtractNumberFromJID("5511999999999:12@s.whatsapp.net"))
	assert.Equal(t, "5511999999999", utils.ExtractNumberFromJID("5511999999999"))
}
