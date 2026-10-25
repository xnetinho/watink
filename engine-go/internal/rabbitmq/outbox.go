package rabbitmq

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// outboxEntry é um evento já serializado, à espera de confirmação do broker.
type outboxEntry struct {
	Seq        uint64 `json:"seq"`
	RoutingKey string `json:"rk"`
	Body       []byte `json:"body"`
}

// outbox guarda em disco os eventos que ainda não foram confirmados pelo broker. O WhatsApp não reentrega mensagens
// recebidas: se o engine publicasse direto e o broker estivesse fora (ou o processo caísse antes da confirmação), o
// evento se perdia. Aqui o evento é gravado ANTES de ser publicado e só sai do arquivo depois do ack do broker.
//
// Formato: um arquivo de linhas JSON ("add" e "del"), recompactado quando fica grande. Não há índice: a fila pendente é
// pequena (só o que ainda não foi confirmado).
//
// shortcut: arquivo único com fsync por gravação, suficiente para o volume de um engine; troque por um banco embutido
// se o pendente passar de dezenas de milhares de eventos.
type outbox struct {
	mu      sync.Mutex
	path    string
	f       *os.File
	pending map[uint64]outboxEntry
	order   []uint64
	nextSeq uint64
	lines   int
}

const outboxCompactAfter = 5000

type outboxRecord struct {
	Op string       `json:"op"` // "add" | "del"
	E  *outboxEntry `json:"e,omitempty"`
	ID uint64       `json:"id,omitempty"`
}

// openOutbox abre (ou cria) o arquivo e recarrega o que ficou pendente de uma execução anterior.
func openOutbox(path string) (*outbox, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("outbox dir: %w", err)
	}
	o := &outbox{path: path, pending: map[uint64]outboxEntry{}}
	if err := o.load(); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("outbox open: %w", err)
	}
	o.f = f
	return o, nil
}

func (o *outbox) load() error {
	f, err := os.Open(o.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("outbox read: %w", err)
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		var r outboxRecord
		if json.Unmarshal(sc.Bytes(), &r) != nil {
			continue // linha truncada por queda no meio da escrita: ignora, o resto continua válido
		}
		o.lines++
		switch r.Op {
		case "add":
			if r.E != nil {
				o.pending[r.E.Seq] = *r.E
				if r.E.Seq >= o.nextSeq {
					o.nextSeq = r.E.Seq + 1
				}
			}
		case "del":
			delete(o.pending, r.ID)
		}
	}
	o.order = o.order[:0]
	for seq := range o.pending {
		o.order = append(o.order, seq)
	}
	sortSeqs(o.order)
	return sc.Err()
}

func sortSeqs(s []uint64) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}

// write acrescenta um registro. sync=true força o disco: só o "add" precisa, porque é ele que protege contra perda. Um
// "del" perdido numa queda apenas faz o evento ser reenviado (e a deduplicação do business o descarta).
func (o *outbox) write(r outboxRecord, sync bool) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if _, err := o.f.Write(append(b, '\n')); err != nil {
		return err
	}
	o.lines++
	if sync {
		return o.f.Sync()
	}
	return nil
}

// Add grava o evento em disco e devolve o número de sequência.
func (o *outbox) Add(routingKey string, body []byte) (uint64, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	e := outboxEntry{Seq: o.nextSeq, RoutingKey: routingKey, Body: body}
	if err := o.write(outboxRecord{Op: "add", E: &e}, true); err != nil {
		return 0, fmt.Errorf("outbox add: %w", err)
	}
	o.nextSeq++
	o.pending[e.Seq] = e
	o.order = append(o.order, e.Seq)
	return e.Seq, nil
}

// Confirm retira o evento depois que o broker o confirmou.
func (o *outbox) Confirm(seq uint64) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if _, ok := o.pending[seq]; !ok {
		return nil
	}
	if err := o.write(outboxRecord{Op: "del", ID: seq}, false); err != nil {
		return fmt.Errorf("outbox confirm: %w", err)
	}
	delete(o.pending, seq)
	for i, s := range o.order {
		if s == seq {
			o.order = append(o.order[:i], o.order[i+1:]...)
			break
		}
	}
	if o.lines > outboxCompactAfter && o.lines > 4*len(o.pending) {
		return o.compactLocked()
	}
	return nil
}

// Pending devolve, em ordem de chegada, o que ainda não foi confirmado.
func (o *outbox) Pending() []outboxEntry {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make([]outboxEntry, 0, len(o.order))
	for _, s := range o.order {
		out = append(out, o.pending[s])
	}
	return out
}

// Len é a quantidade de eventos pendentes.
func (o *outbox) Len() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.pending)
}

// compactLocked reescreve o arquivo só com o que está pendente.
func (o *outbox) compactLocked() error {
	tmp := o.path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	lines := 0
	for _, s := range o.order {
		e := o.pending[s]
		b, _ := json.Marshal(outboxRecord{Op: "add", E: &e})
		if _, err := f.Write(append(b, '\n')); err != nil {
			_ = f.Close()
			return err
		}
		lines++
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	_ = f.Close()
	_ = o.f.Close()
	if err := os.Rename(tmp, o.path); err != nil {
		return err
	}
	nf, err := os.OpenFile(o.path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	o.f = nf
	o.lines = lines
	return nil
}

// Close fecha o arquivo. O conteúdo pendente permanece para a próxima execução.
func (o *outbox) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.f == nil {
		return nil
	}
	err := o.f.Close()
	o.f = nil
	return err
}
