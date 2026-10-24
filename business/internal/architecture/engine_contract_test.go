package architecture

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Contrato de eventos engine -> business, verificado pelo código dos dois lados enquanto não existe o módulo
// compartilhado (tarefa 2.1). O engine publica "wbot.<tenant>.<sessão>.<evento>"; o business só trata o que está na lista
// de chaves do StartEventListener. Um nome diferente nos dois lados (como session.pairingcode x session.pairing_code)
// faz o evento ser publicado e descartado em silêncio.

var enginePublishRe = regexp.MustCompile(`publishEvent\([^"\n]*"((?:session|message|contact)\.[A-Za-z_.]+)"`)
var listenerKeyRe = regexp.MustCompile(`"wbot\.\*\.\*\.((?:session|message|contact)\.[A-Za-z_.]+)"`)

// Eventos que só existem de um lado, com o motivo. Entrada nova exige justificativa.
var (
	emittedButNotListened = map[string]string{}
	listenedButNotEmitted = map[string]string{
		"message.poll_vote": "o engine ainda não decifra nem emite o voto (tarefa 2.4); o tratador do business já existe",
	}
)

func repoRoot(t *testing.T) string {
	t.Helper()
	return filepath.Dir(businessRoot(t))
}

func engineEmittedEvents(t *testing.T) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	dir := filepath.Join(repoRoot(t), "engine-go", "internal")
	_, err := os.Stat(dir)
	require.NoError(t, err, "engine-go precisa estar no checkout para o teste de contrato")
	require.NoError(t, filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		b, rerr := os.ReadFile(path)
		require.NoError(t, rerr)
		for _, m := range enginePublishRe.FindAllStringSubmatch(string(b), -1) {
			out[m[1]] = true
		}
		return nil
	}))
	return out
}

func businessListenedEvents(t *testing.T) map[string]bool {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(businessRoot(t), "internal", "services", "event_listener.go"))
	require.NoError(t, err)
	out := map[string]bool{}
	for _, m := range listenerKeyRe.FindAllStringSubmatch(string(b), -1) {
		out[m[1]] = true
	}
	return out
}

func diff(a, b map[string]bool, allowed map[string]string) []string {
	var out []string
	for k := range a {
		if !b[k] {
			if _, ok := allowed[k]; !ok {
				out = append(out, k)
			}
		}
	}
	sort.Strings(out)
	return out
}

func TestContratoEngine_TodoEventoEmitidoTemTratador(t *testing.T) {
	emitted, listened := engineEmittedEvents(t), businessListenedEvents(t)
	require.NotEmpty(t, emitted, "nenhum evento do engine encontrado: o padrão de busca quebrou")
	require.NotEmpty(t, listened, "nenhuma chave do listener encontrada: o padrão de busca quebrou")
	assert.Empty(t, diff(emitted, listened, emittedButNotListened),
		"o engine emite evento que o business não escuta (publicado e descartado)")
}

func TestContratoEngine_TodaChaveEscutadaTemEmissor(t *testing.T) {
	emitted, listened := engineEmittedEvents(t), businessListenedEvents(t)
	assert.Empty(t, diff(listened, emitted, listenedButNotEmitted),
		"o business escuta evento que o engine não emite (tratador morto ou nome divergente)")
}

func TestContratoEngine_ExcecoesTemJustificativa(t *testing.T) {
	for _, m := range []map[string]string{emittedButNotListened, listenedButNotEmitted} {
		for k, why := range m {
			assert.GreaterOrEqual(t, len(strings.TrimSpace(why)), 15, "exceção %q sem justificativa", k)
		}
	}
	// Exceção que deixou de ser necessária tem de sair da lista.
	emitted, listened := engineEmittedEvents(t), businessListenedEvents(t)
	for k := range listenedButNotEmitted {
		assert.False(t, emitted[k], "%q já é emitido pelo engine: tire da lista de exceções", k)
	}
	for k := range emittedButNotListened {
		assert.False(t, listened[k], "%q já é escutado pelo business: tire da lista de exceções", k)
	}
}
