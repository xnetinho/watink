package services

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alltomatos/watinkdev/business/internal/models"
	"github.com/alltomatos/watinkdev/business/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type editEnv struct {
	db     *gorm.DB
	tenant uuid.UUID
	t0     time.Time
}

func newEditEnv(t *testing.T) *editEnv {
	t.Helper()
	db := testutil.NewTestDB(t)
	tenant := uuid.New()
	require.NoError(t, db.Create(&models.Tenant{ID: tenant, Name: "T", Status: "active"}).Error)
	wa := models.Whatsapp{Name: "W", TenantID: tenant, Status: "CONNECTED"}
	require.NoError(t, db.Create(&wa).Error)
	ct := models.Contact{Name: "C", Number: "5511999990001", TenantID: tenant}
	require.NoError(t, db.Create(&ct).Error)
	tk := models.Ticket{ContactID: ct.ID, WhatsappID: wa.ID, TenantID: tenant, Status: "open"}
	require.NoError(t, db.Create(&tk).Error)
	t0 := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	require.NoError(t, db.Create(&models.Message{ID: "M1", Body: "Oi", TicketID: tk.ID, TenantID: tenant, CreatedAt: t0, UpdatedAt: t0}).Error)
	return &editEnv{db: db, tenant: tenant, t0: t0}
}

func (e *editEnv) edit(t *testing.T, body string, at time.Time, by string) RecordEditResult {
	t.Helper()
	r, err := RecordEdit(context.Background(), e.db, RecordEditInput{TenantID: e.tenant, MessageID: "M1", Body: body, EditedAt: at, EditedBy: by})
	require.NoError(t, err)
	return r
}

func (e *editEnv) chain(t *testing.T) []models.MessageEdit {
	t.Helper()
	c, err := MessageEdits(context.Background(), e.db, e.tenant, "M1")
	require.NoError(t, err)
	return c
}

func (e *editEnv) message(t *testing.T) models.Message {
	t.Helper()
	var m models.Message
	require.NoError(t, e.db.Where("id = ?", "M1").Take(&m).Error)
	return m
}

func bodies(c []models.MessageEdit) []string {
	out := make([]string, len(c))
	for i, v := range c {
		out[i] = v.Body
	}
	return out
}

func TestRecordEdit_PrimeiraEdicaoGuardaOOriginalComoVersao1(t *testing.T) {
	e := newEditEnv(t)
	r := e.edit(t, "Olá", e.t0.Add(time.Minute), models.EditedByContact)

	assert.True(t, r.Applied)
	c := e.chain(t)
	require.Len(t, c, 2)
	assert.Equal(t, []string{"Oi", "Olá"}, bodies(c))
	assert.Equal(t, 1, c[0].Version)
	assert.Equal(t, 2, c[1].Version)
	assert.Equal(t, e.t0, c[0].EditedAt.UTC(), "a versão 1 é do instante em que a mensagem foi criada")
	m := e.message(t)
	assert.Equal(t, "Olá", m.Body)
	assert.Equal(t, 1, m.EditCount)
}

func TestRecordEdit_DezEdicoesDeixamAsDezVersoesMaisAOriginal(t *testing.T) {
	e := newEditEnv(t)
	for i := 1; i <= 10; i++ {
		e.edit(t, "texto "+string(rune('a'+i-1)), e.t0.Add(time.Duration(i)*time.Minute), models.EditedByContact)
	}
	c := e.chain(t)
	require.Len(t, c, 11, "original + 10 edições")
	for i, v := range c {
		assert.Equal(t, i+1, v.Version, "versões contíguas, sem buraco")
	}
	assert.Equal(t, "Oi", c[0].Body)
	assert.Equal(t, "texto j", c[10].Body)
	m := e.message(t)
	assert.Equal(t, "texto j", m.Body)
	assert.Equal(t, 10, m.EditCount)
}

func TestRecordEdit_MesmoEventoDuasVezesNaoDuplica(t *testing.T) {
	e := newEditEnv(t)
	at := e.t0.Add(time.Minute)
	first := e.edit(t, "Olá", at, models.EditedByContact)
	again := e.edit(t, "Olá", at, models.EditedByContact)

	assert.True(t, first.Applied)
	assert.True(t, again.Duplicate)
	assert.False(t, again.Applied)
	assert.Len(t, e.chain(t), 2, "o engine entrega ao menos uma vez: a repetição não cria versão")
	assert.Equal(t, 1, e.message(t).EditCount)
}

func TestRecordEdit_ForaDeOrdemEntraNaCadeiaMasONaoSobrescreveOTextoAtual(t *testing.T) {
	e := newEditEnv(t)
	e.edit(t, "terceira", e.t0.Add(3*time.Minute), models.EditedByContact) // chegou primeiro, mas é a mais nova
	r := e.edit(t, "segunda", e.t0.Add(2*time.Minute), models.EditedByContact)

	assert.False(t, r.Applied, "uma edição mais antiga que a atual não pode voltar o texto")
	assert.Equal(t, "terceira", e.message(t).Body)
	assert.Equal(t, 2, e.message(t).EditCount)
	assert.Equal(t, []string{"Oi", "terceira", "segunda"}, bodies(e.chain(t)), "o histórico mostra o que chegou")
}

func TestRecordEdit_QuemEditouFicaRegistrado(t *testing.T) {
	e := newEditEnv(t)
	e.edit(t, "feita por mim", e.t0.Add(time.Minute), models.EditedByMe)
	c := e.chain(t)
	assert.Equal(t, models.EditedByContact, c[0].EditedBy, "a versão 1 é do autor da mensagem (aqui, o contato)")
	assert.Equal(t, models.EditedByMe, c[1].EditedBy)
}

func TestRecordEdit_MensagemNossaTemOriginalDoOperador(t *testing.T) {
	e := newEditEnv(t)
	require.NoError(t, e.db.Model(&models.Message{}).Where("id = ?", "M1").Update("fromMe", true).Error)
	e.edit(t, "corrigida", e.t0.Add(time.Minute), models.EditedByMe)
	assert.Equal(t, models.EditedByMe, e.chain(t)[0].EditedBy)
}

func TestRecordEdit_MensagemInexistenteOuDeOutraEmpresa(t *testing.T) {
	e := newEditEnv(t)
	_, err := RecordEdit(context.Background(), e.db, RecordEditInput{TenantID: e.tenant, MessageID: "NAO_EXISTE", Body: "x", EditedAt: e.t0})
	assert.ErrorIs(t, err, ErrEditTargetNotFound)

	_, err = RecordEdit(context.Background(), e.db, RecordEditInput{TenantID: uuid.New(), MessageID: "M1", Body: "x", EditedAt: e.t0})
	assert.ErrorIs(t, err, ErrEditTargetNotFound, "empresa diferente não enxerga nem edita a mensagem")
	assert.Equal(t, "Oi", e.message(t).Body)

	other, err := MessageEdits(context.Background(), e.db, uuid.New(), "M1")
	require.NoError(t, err)
	assert.Empty(t, other, "a cadeia também é isolada por empresa")
}

func TestRecordEdit_EdicoesSimultaneasNaoDisputamAVersao(t *testing.T) {
	e := newEditEnv(t)
	var wg sync.WaitGroup
	for i := 1; i <= 6; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := RecordEdit(context.Background(), e.db, RecordEditInput{
				TenantID: e.tenant, MessageID: "M1", Body: "v" + string(rune('0'+i)), EditedAt: e.t0.Add(time.Duration(i) * time.Minute), EditedBy: models.EditedByContact,
			})
			assert.NoError(t, err)
		}(i)
	}
	wg.Wait()

	c := e.chain(t)
	require.Len(t, c, 7, "original + 6 edições, sem versão repetida nem perdida")
	seen := map[int]bool{}
	for _, v := range c {
		assert.False(t, seen[v.Version], "versão %d repetida", v.Version)
		seen[v.Version] = true
	}
	assert.Equal(t, "v6", e.message(t).Body, "vence a edição de maior horário, não a última a gravar")
}
