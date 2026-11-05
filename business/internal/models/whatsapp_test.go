package models_test

import (
	"testing"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestWhatsapp_RoundTripPelasTabelasAgnosticas(t *testing.T) {
	db := testutil.NewTestDB(t)
	tenant, other := uuid.New(), uuid.New()
	proxy := 7
	w := models.Whatsapp{TenantID: tenant, Name: "zap", Status: "CONNECTED", Number: "5511999990000", ProfilePicUrl: "http://x/p.jpg",
		KeepAlive: true, PresencePolicy: "always", PresenceIdleMinutes: 9, EngineType: "whatsmeow", ProxyMode: "single", ProxyID: &proxy}
	require.NoError(t, models.CreateWhatsapp(db, &w))
	require.NotZero(t, w.ID)

	var cc models.ChannelConnection
	require.NoError(t, db.First(&cc, w.ID).Error)
	require.Equal(t, "whatsapp", cc.Kind)
	require.Equal(t, "5511999990000", cc.Identifier, "número mora na identidade agnóstica")
	require.Equal(t, "http://x/p.jpg", cc.AvatarURL)
	var d models.WhatsAppConnectionDetails
	require.NoError(t, db.First(&d, `"connectionId" = ?`, w.ID).Error)
	require.True(t, d.KeepAlive)
	require.Equal(t, "always", d.PresencePolicy)

	var got models.Whatsapp
	require.NoError(t, models.FirstWhatsapp(db.Where("id = ?", w.ID), &got))
	require.Equal(t, w.Number, got.Number)
	require.Equal(t, 9, got.PresenceIdleMinutes)
	require.Equal(t, &proxy, got.ProxyID)

	require.NoError(t, models.UpdateWhatsapp(db, tenant, w.ID, map[string]interface{}{"status": "OPENING", "keepAlive": false, "number": "5511888880000"}))
	require.NoError(t, models.FirstWhatsapp(db.Where("id = ?", w.ID), &got))
	require.Equal(t, "OPENING", got.Status)
	require.False(t, got.KeepAlive)
	require.Equal(t, "5511888880000", got.Number)

	require.Error(t, models.UpdateWhatsapp(db, tenant, w.ID, map[string]interface{}{"qrcode": "x"}), "campo desconhecido não pode passar em silêncio")

	// outra empresa não altera nem remove
	require.NoError(t, models.UpdateWhatsapp(db, other, w.ID, map[string]interface{}{"status": "BANNED", "keepAlive": true}))
	require.NoError(t, models.DeleteWhatsapp(db, other, w.ID))
	require.NoError(t, models.FirstWhatsapp(db.Where("id = ?", w.ID), &got))
	require.Equal(t, "OPENING", got.Status)
	require.False(t, got.KeepAlive)

	q := models.Queue{Name: "F", Color: "#000", TenantID: tenant}
	require.NoError(t, db.Create(&q).Error)
	require.NoError(t, models.SetWhatsappQueues(db, tenant, q.ID, []int{w.ID}))
	require.NoError(t, models.LoadWhatsappQueues(db, &got))
	require.Len(t, got.Queues, 1)
	require.NoError(t, models.SetWhatsappQueues(db, tenant, q.ID, nil))
	require.NoError(t, models.LoadWhatsappQueues(db, &got))
	require.Empty(t, got.Queues)

	require.NoError(t, models.DeleteWhatsapp(db, tenant, w.ID))
	require.ErrorIs(t, models.FirstWhatsapp(db.Where("id = ?", w.ID), &got), gorm.ErrRecordNotFound)
}
