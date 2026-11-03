// Package waversion decide qual versão do cliente WhatsApp Web o engine anuncia no handshake. O WhatsApp recusa versões
// velhas (405 "client outdated") e a embutida no whatsmeow envelhece entre um deploy e outro; por isso a versão é resolvida
// em tempo de execução. O superadmin escolhe o modo na tela: "auto" (site oficial, depois registro wa-version, depois a
// embutida) ou "manual" (só o número cadastrado, sem consultar a rede).
package waversion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
)

const (
	ModeAuto   = "auto"
	ModeManual = "manual"

	// DefaultURL é o registro público de versões do WhatsApp Web mantido pelo WPPConnect.
	DefaultURL = "https://raw.githubusercontent.com/wppconnect-team/wa-version/refs/heads/main/versions.json"

	fetchTimeout = 8 * time.Second
	maxJSONBytes = 4 << 20
	minRetry     = 5 * time.Minute
)

// Setting é a escolha do superadmin (tabela WaVersionSettings).
type Setting struct {
	Mode   string
	Manual string
}

// Source é uma fonte online de versão. Devolve erro quando indisponível ou com resposta inválida.
type Source struct {
	Name  string
	Fetch func(ctx context.Context) (store.WAVersionContainer, error)
}

// Resolver aplica a versão conforme a escolha do superadmin.
type Resolver struct {
	Sources  []Source // modo auto, em ordem de preferência
	Load     func(ctx context.Context) (Setting, error)
	Get      func() store.WAVersionContainer
	Set      func(store.WAVersionContainer)
	Embedded store.WAVersionContainer // piso do modo auto: a versão que veio compilada no whatsmeow

	mu     sync.Mutex
	last   time.Time
	source string
}

// NewResolver monta o resolvedor padrão. load lê a escolha do banco; nil equivale a sempre "auto".
func NewResolver(load func(ctx context.Context) (Setting, error)) *Resolver {
	client := &http.Client{Timeout: fetchTimeout}
	return &Resolver{
		Sources: []Source{
			{"oficial", func(ctx context.Context) (store.WAVersionContainer, error) { return fetchOfficial(ctx, client) }},
			{"wa-version", func(ctx context.Context) (store.WAVersionContainer, error) {
				return fetchRegistry(ctx, client, DefaultURL)
			}},
		},
		Load:     load,
		Get:      store.GetWAVersion,
		Set:      store.SetWAVersion,
		Embedded: store.GetWAVersion(),
	}
}

func fetchOfficial(ctx context.Context, client *http.Client) (store.WAVersionContainer, error) {
	v, err := whatsmeow.GetLatestVersion(ctx, client)
	if err != nil {
		return store.WAVersionContainer{}, err
	}
	return *v, nil
}

// ParseVersion aceita "2.3000.1052747707" e a forma do registro, com sufixo ("2.3000.1052747707-alpha").
func ParseVersion(raw string) (store.WAVersionContainer, error) {
	raw, _, _ = strings.Cut(strings.TrimSpace(raw), "-")
	v, err := store.ParseVersion(raw)
	if err != nil {
		return v, err
	}
	if v.IsZero() || v[0] != 2 {
		return store.WAVersionContainer{}, fmt.Errorf("versão %q fora do padrão 2.x.y", raw)
	}
	return v, nil
}

func fetchRegistry(ctx context.Context, client *http.Client, url string) (store.WAVersionContainer, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return store.WAVersionContainer{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return store.WAVersionContainer{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return store.WAVersionContainer{}, fmt.Errorf("status %d", resp.StatusCode)
	}
	var doc struct {
		CurrentVersion string `json:"currentVersion"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxJSONBytes)).Decode(&doc); err != nil {
		return store.WAVersionContainer{}, err
	}
	return ParseVersion(doc.CurrentVersion)
}

// setting lê a escolha; sem leitor, sem linha ou com erro de leitura o modo é "auto" (a conexão nunca fica refém do banco).
func (r *Resolver) setting(ctx context.Context) (Setting, error) {
	if r.Load == nil {
		return Setting{Mode: ModeAuto}, nil
	}
	s, err := r.Load(ctx)
	if err != nil {
		return Setting{Mode: ModeAuto}, err
	}
	if s.Mode != ModeManual {
		s.Mode = ModeAuto
	}
	return s, nil
}

// Resolve aplica a versão. Devolve a vigente, a fonte ("manual", "oficial", "wa-version" ou "embutida") e se mudou.
// O erro só descreve o que falhou pelo caminho; a vigente sempre é válida.
func (r *Resolver) Resolve(ctx context.Context) (current store.WAVersionContainer, source string, changed bool, err error) {
	set, serr := r.setting(ctx)
	var errs []string
	if serr != nil {
		errs = append(errs, "configuração: "+serr.Error())
	}
	if set.Mode == ModeManual {
		v, perr := ParseVersion(set.Manual)
		if perr == nil {
			current, changed = r.apply(v, "manual")
			return current, "manual", changed, joinErr(errs)
		}
		errs = append(errs, "versão manual inválida, usando o automático: "+perr.Error())
	}
	for _, src := range r.Sources {
		cctx, cancel := context.WithTimeout(ctx, fetchTimeout)
		v, ferr := src.Fetch(cctx)
		cancel()
		if ferr != nil {
			errs = append(errs, src.Name+": "+ferr.Error())
			continue
		}
		if v.LessThan(r.Embedded) {
			errs = append(errs, fmt.Sprintf("%s: %s é mais antiga que a embutida %s", src.Name, v, r.Embedded))
			continue
		}
		current, changed = r.apply(v, src.Name)
		return current, src.Name, changed, joinErr(errs)
	}
	// Nenhuma fonte serviu: volta à embutida (cobre sair do modo manual sem rede).
	current, changed = r.apply(r.Embedded, "embutida")
	return current, "embutida", changed, joinErr(errs)
}

func joinErr(errs []string) error {
	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "; "))
}

// apply grava a versão e a fonte. shortcut: store.SetWAVersion escreve numa variável global sem lock; é chamado na subida,
// após 405 e ao salvar a tela, nunca no caminho quente de mensagens. Se isso virar problema, serializar a conexão das sessões.
func (r *Resolver) apply(v store.WAVersionContainer, source string) (store.WAVersionContainer, bool) {
	r.mu.Lock()
	r.source = source
	r.mu.Unlock()
	if r.Get() == v {
		return v, false
	}
	r.Set(v)
	return v, true
}

// Refresh resolve e registra o resultado no log. Sem force, chamadas em menos de 5 min da anterior são ignoradas (várias
// sessões recebem 405 juntas); com force (salvar na tela) sempre resolve. Devolve se a versão mudou.
func (r *Resolver) Refresh(ctx context.Context, force bool) bool {
	r.mu.Lock()
	if !force && !r.last.IsZero() && time.Since(r.last) < minRetry {
		r.mu.Unlock()
		return false
	}
	r.last = time.Now()
	r.mu.Unlock()
	v, src, changed, err := r.Resolve(ctx)
	switch {
	case changed:
		log.Printf("[waversion] versão do cliente atualizada para %s (fonte: %s)", v, src)
	default:
		log.Printf("[waversion] versão do cliente %s confirmada (fonte: %s)", v, src)
	}
	if err != nil {
		log.Printf("[waversion] avisos: %v", err)
	}
	return changed
}

// Probe é o resultado de consultar uma fonte: a versão ou o motivo da falha.
type Probe struct {
	Version string `json:"version,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Status é o retrato mostrado na tela do superadmin.
type Status struct {
	Mode      string `json:"mode"`
	Manual    string `json:"manual,omitempty"`
	Embedded  string `json:"embedded"`
	Official  Probe  `json:"official"`
	Registry  Probe  `json:"registry"`
	Effective string `json:"effective"`
	Source    string `json:"source,omitempty"`
}

// Status consulta as fontes agora (sem aplicar nada) e devolve as versões lado a lado.
func (r *Resolver) Status(ctx context.Context) Status {
	set, _ := r.setting(ctx)
	r.mu.Lock()
	src := r.source
	r.mu.Unlock()
	st := Status{Mode: set.Mode, Manual: set.Manual, Embedded: r.Embedded.String(), Effective: r.Get().String(), Source: src}
	for _, s := range r.Sources {
		cctx, cancel := context.WithTimeout(ctx, fetchTimeout)
		v, err := s.Fetch(cctx)
		cancel()
		p := Probe{Version: v.String()}
		if err != nil {
			p = Probe{Error: err.Error()}
		}
		switch s.Name {
		case "oficial":
			st.Official = p
		case "wa-version":
			st.Registry = p
		}
	}
	return st
}
