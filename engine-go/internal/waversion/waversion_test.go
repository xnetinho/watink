package waversion

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.mau.fi/whatsmeow/store"
)

type V = store.WAVersionContainer

func fixed(v V, err error) func(context.Context) (V, error) {
	return func(context.Context) (V, error) { return v, err }
}

func newR(cur V, srcs ...Source) (*Resolver, *V) {
	applied := cur
	return &Resolver{Sources: srcs, Get: func() V { return applied }, Set: func(v V) { applied = v }}, &applied
}

func TestResolveUsaPrimeiraFonteValida(t *testing.T) {
	r, applied := newR(V{2, 3000, 100},
		Source{"a", fixed(V{}, errors.New("fora do ar"))},
		Source{"b", fixed(V{2, 3000, 200}, nil)},
		Source{"c", fixed(V{2, 3000, 300}, nil)})
	v, src, changed, err := r.Resolve(context.Background())
	if err != nil || !changed || src != "b" || v != (V{2, 3000, 200}) || *applied != v {
		t.Fatalf("v=%v src=%s changed=%v err=%v applied=%v", v, src, changed, err, *applied)
	}
}

func TestResolveNaoRecuaParaVersaoMaisAntiga(t *testing.T) {
	r, applied := newR(V{2, 3000, 500}, Source{"velha", fixed(V{2, 3000, 100}, nil)})
	v, _, changed, err := r.Resolve(context.Background())
	if err == nil || changed || v != (V{2, 3000, 500}) || *applied != v {
		t.Fatalf("deveria manter a vigente: v=%v changed=%v err=%v", v, changed, err)
	}
}

func TestResolveSemFonteMantemVigente(t *testing.T) {
	r, applied := newR(V{2, 3000, 500}, Source{"a", fixed(V{}, errors.New("x"))})
	if _, _, changed, err := r.Resolve(context.Background()); err == nil || changed || *applied != (V{2, 3000, 500}) {
		t.Fatalf("changed=%v err=%v applied=%v", changed, err, *applied)
	}
}

func TestResolveIgualNaoMuda(t *testing.T) {
	r, _ := newR(V{2, 3000, 500}, Source{"a", fixed(V{2, 3000, 500}, nil)})
	if _, src, changed, err := r.Resolve(context.Background()); err != nil || changed || src != "a" {
		t.Fatalf("src=%s changed=%v err=%v", src, changed, err)
	}
}

func TestParseVersionAceitaSufixoAlpha(t *testing.T) {
	v, err := parseVersion("2.3000.1052747707-alpha")
	if err != nil || v != (V{2, 3000, 1052747707}) {
		t.Fatalf("v=%v err=%v", v, err)
	}
	for _, bad := range []string{"", "abc", "1.2", "3.3000.1", "0.0.0"} {
		if _, err := parseVersion(bad); err == nil {
			t.Fatalf("%q deveria ser inválida", bad)
		}
	}
}

func TestParsePin(t *testing.T) {
	if _, err := parsePin(" "); err == nil {
		t.Fatal("vazia deveria falhar")
	}
	if v, err := parsePin("2.3000.42"); err != nil || v != (V{2, 3000, 42}) {
		t.Fatalf("v=%v err=%v", v, err)
	}
}

func TestFetchRegistry(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"currentBeta":null,"currentVersion":"2.3000.1052747707-alpha","versions":[]}`))
	}))
	defer ok.Close()
	if v, err := fetchRegistry(context.Background(), ok.Client(), ok.URL); err != nil || v != (V{2, 3000, 1052747707}) {
		t.Fatalf("v=%v err=%v", v, err)
	}
	for name, h := range map[string]http.HandlerFunc{
		"500":       func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(500) },
		"não json":  func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("<html>")) },
		"sem campo": func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{}`)) },
	} {
		s := httptest.NewServer(h)
		if _, err := fetchRegistry(context.Background(), s.Client(), s.URL); err == nil {
			t.Fatalf("%s deveria falhar", name)
		}
		s.Close()
	}
}

func TestRefreshLimitaFrequencia(t *testing.T) {
	calls := 0
	r, _ := newR(V{2, 3000, 1}, Source{"a", func(context.Context) (V, error) { calls++; return V{2, 3000, 2}, nil }})
	if !r.Refresh(context.Background()) || r.Refresh(context.Background()) || calls != 1 {
		t.Fatalf("segunda chamada seguida deveria ser ignorada (calls=%d)", calls)
	}
}
