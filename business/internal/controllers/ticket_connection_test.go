package controllers

import (
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Identificação do ticket: canal + conexão, no mesmo formato para qualquer canal.
func TestAttachChannelInfo_IdentificaCanalEConexao(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: tenant, Name: "T", Status: "active"}).Error)
	wa := models.Whatsapp{Name: "Comercial", TenantID: tenant, Status: "CONNECTED", ProfilePicUrl: "https://pps.whatsapp.net/comercial.jpg"}
	wa2 := models.Whatsapp{Name: "Suporte", TenantID: tenant, Status: "CONNECTED"}
	require.NoError(t, db.Create(&wa).Error)
	require.NoError(t, db.Create(&wa2).Error)
	tg := models.ChannelConnection{TenantID: tenant, Kind: "telegram", Name: "Jarvis", Status: "CONNECTED"}
	ml := models.ChannelConnection{TenantID: tenant, Kind: "mercadolivre", Name: "Loja ML", Status: "CONNECTED"}
	require.NoError(t, db.Create(&tg).Error)
	require.NoError(t, db.Create(&ml).Error)

	tickets := []models.Ticket{
		{ID: 1, WhatsappID: wa.ID},
		{ID: 2, WhatsappID: wa2.ID},
		{ID: 3, ChannelConnectionID: &tg.ID},
		{ID: 4, ChannelConnectionID: &ml.ID}, // canal futuro: o mesmo formato, sem código novo
		{ID: 5, WhatsappID: 9999},            // conexão apagada
	}
	attachChannelInfo(db, tickets)

	byID := map[int]models.Ticket{}
	for _, tk := range tickets {
		byID[tk.ID] = tk
	}
	assert.Equal(t, "whatsapp", byID[1].Channel)
	require.NotNil(t, byID[1].Connection)
	assert.Equal(t, models.TicketConnectionInfo{Kind: "whatsapp", ID: wa.ID, Name: "Comercial", Source: "whatsapp", ProfilePicUrl: "https://pps.whatsapp.net/comercial.jpg"}, *byID[1].Connection,
		"a foto da conexão viaja junto: é o avatar das mensagens que o Watink envia")
	assert.Empty(t, byID[2].Connection.ProfilePicUrl, "conexão sem foto: vazio (a tela cai nas iniciais)")
	assert.Equal(t, "Suporte", byID[2].Connection.Name, "duas conexões do mesmo canal se distinguem pelo nome")

	assert.Equal(t, "telegram", byID[3].Channel)
	assert.Equal(t, models.TicketConnectionInfo{Kind: "telegram", ID: tg.ID, Name: "Jarvis", Source: "channel"}, *byID[3].Connection)

	assert.Equal(t, "mercadolivre", byID[4].Channel)
	assert.Equal(t, "Loja ML", byID[4].Connection.Name)

	assert.Equal(t, "whatsapp", byID[5].Channel, "sem conexão, o legado continua sendo WhatsApp")
	assert.Nil(t, byID[5].Connection, "conexão apagada: sem identificação, nunca um nome inventado")
}

func TestAttachChannelInfo_ListaVaziaEUmaConsultaPorTabela(t *testing.T) {
	db := testutil.NewTestDB(t)
	attachChannelInfo(db, nil)
	attachChannelInfo(db, []models.Ticket{})
}
