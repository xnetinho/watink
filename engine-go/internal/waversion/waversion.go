// Package waversion decide qual versão do cliente WhatsApp Web o engine anuncia no handshake. O WhatsApp recusa versões
// velhas (405 "client outdated") e a embutida no whatsmeow envelhece entre um deploy e outro; por isso a versão é resolvida
// em tempo de execução, com fallback em cadeia, para as conexões não caírem.
package waversion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
)

const (
	// PinEnv fixa uma versão ("2.3000.1052747707") usada quando nenhuma fonte online responde.
	PinEnv = "WATINK_WA_VERSION"
	// DefaultURL é o registro público de versões do WhatsApp Web mantido pelo WPPConnect.
	DefaultURL = "https://raw.githubusercontent.com/wppconnect-team/wa-version/refs/heads/main/versions.json"

	fetchTimeout = 8 * time.Second
	maxJSONBytes = 4 << 20
	minRetry     = 5 * time.Minute
)

// Source é uma fonte de versão. Devolve erro quando indisponível ou com resposta inválida.
type Source struct {
	Name  string
	Fetch func(ctx context.Context) (store.WAVersionContainer, error)
}

// Resolver aplica a primeira versão válida da cadeia, sem nunca recuar para uma menor que a atual.
type Resolver struct {
	Sources []Source
	Get     func() store.WAVersionContainer
	Set     func(store.WAVersionContainer)

	mu   sync.Mutex
	last time.Time
}

// NewResolver monta a cadeia padrão: site oficial, registro do WPPConnect, versão fixada por WATINK_WA_VERSION; a
// versão embutida no whatsmeow é o último recurso (simplesmente continua valendo).
func NewResolver() *Resolver {
	client := &http.Client{Timeout: fetchTimeout}
	return &Resolver{
		Sources: []Source{
			{"oficial", func(ctx context.Context) (store.WAVersionContainer, error) {
				v, err := whatsmeow.GetLatestVersion(ctx, client)
				if err != nil {
					return store.WAVersionContainer{}, err
				}
				return *v, nil
			}},
			{"wa-version", func(ctx context.Context) (store.WAVersionContainer, error) {
				return fetchRegistry(ctx, client, DefaultURL)
			}},
			{"fixada", func(context.Context) (store.WAVersionContainer, error) { return parsePin(os.Getenv(PinEnv)) }},
		},
		Get: store.GetWAVersion,
		Set: store.SetWAVersion,
	}
}

func parsePin(raw string) (store.WAVersionContainer, error) {
	if strings.TrimSpace(raw) == "" {
		return store.WAVersionContainer{}, errors.New(PinEnv + " não definida")
	}
	return parseVersion(raw)
}

// parseVersion aceita "2.3000.1052747707" e a forma do registro, com sufixo ("2.3000.1052747707-alpha").
func parseVersion(raw string) (store.WAVersionContainer, error) {
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
	return parseVersion(doc.CurrentVersion)
}

// Resolve percorre a cadeia e aplica a versão. Devolve a versão vigente, a fonte e se mudou. Uma versão menor que a atual é
// ignorada (fonte desatualizada); se nenhuma fonte serve, a vigente continua e o erro vem junto.
func (r *Resolver) Resolve(ctx context.Context) (current store.WAVersionContainer, source string, changed bool, err error) {
	current = r.Get()
	var errs []string
	for _, src := range r.Sources {
		cctx, cancel := context.WithTimeout(ctx, fetchTimeout)
		v, ferr := src.Fetch(cctx)
		cancel()
		if ferr != nil {
			errs = append(errs, src.Name+": "+ferr.Error())
			continue
		}
		if v.LessThan(current) {
			errs = append(errs, fmt.Sprintf("%s: %s é mais antiga que a vigente %s", src.Name, v, current))
			continue
		}
		if v == current {
			return current, src.Name, false, nil
		}
		// shortcut: store.SetWAVersion escreve numa variável global sem lock; chamado só na subida e após 405, quando as
		// sessões já falharam. Se isso virar problema, serializar a conexão das sessões durante a troca.
		r.Set(v)
		return v, src.Name, true, nil
	}
	return current, "", false, errors.New(strings.Join(errs, "; "))
}

// Refresh resolve e registra o resultado no log. Chamadas em menos de 5 min da anterior são ignoradas (evita martelar as
// fontes quando várias sessões recebem 405 juntas). Devolve se a versão mudou.
func (r *Resolver) Refresh(ctx context.Context) bool {
	r.mu.Lock()
	if !r.last.IsZero() && time.Since(r.last) < minRetry {
		r.mu.Unlock()
		return false
	}
	r.last = time.Now()
	r.mu.Unlock()
	v, src, changed, err := r.Resolve(ctx)
	switch {
	case err != nil:
		log.Printf("[waversion] nenhuma fonte serviu, mantendo %s: %v", v, err)
	case changed:
		log.Printf("[waversion] versão do cliente atualizada para %s (fonte: %s)", v, src)
	default:
		log.Printf("[waversion] versão do cliente %s confirmada (fonte: %s)", v, src)
	}
	return changed
}
