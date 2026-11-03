package waversion

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.mau.fi/whatsmeow/store"
)

type V = store.WAVersionContainer

var embedded = V{2, 3000, 1000}

func fixed(v V, err error) func(context.Context) (V, error) {
	return func(context.Context) (V, error) { return v, err }
}

func newR(set Setting, loadErr error, srcs ...Source) (*Resolver, *V) {
	applied := embedded
	return &Resolver{
		Sources:  srcs,
		Load:     func(context.Context) (Setting, error) { return set, loadErr },
		Get:      func() V { return applied },
		Set:      func(v V) { applied = v },
		Embedded: embedded,
	}, &applied
}

func TestAutoUsaPrimeiraFonteValida(t *testing.T) {
	r, applied := newR(Setting{Mode: ModeAuto}, nil,
		Source{"oficial", fixed(V{}, errors.New("fora do ar"))},
		Source{"wa-version", fixed(V{2, 3000, 2000}, nil)})
	v, src, changed, err := r.Resolve(context.Background())
	if !changed || src != "wa-version" || v != (V{2, 3000, 2000}) || *applied != v {
		t.Fatalf("v=%v src=%s changed=%v applied=%v", v, src, changed, *applied)
	}
	if err == nil || !strings.Contains(err.Error(), "oficial") {
		t.Fatalf("o erro da fonte que falhou deve aparecer: %v", err)
	}
}

func TestAutoNaoAceitaFonteMenorQueAEmbutida(t *testing.T) {
	r, applied := newR(Setting{Mode: ModeAuto}, nil, Source{"oficial", fixed(V{2, 3000, 10}, nil)})
	v, src, changed, _ := r.Resolve(context.Background())
	if changed || src != "embutida" || v != embedded || *applied != embedded {
		t.Fatalf("deveria ficar na embutida: v=%v src=%s changed=%v", v, src, changed)
	}
}

func TestManualIgnoraAsFontes(t *testing.T) {
	r, applied := newR(Setting{Mode: ModeManual, Manual: "2.3000.1052747707-alpha"}, nil,
		Source{"oficial", func(context.Context) (V, error) { t.Fatal("manual não deve consultar a rede"); return V{}, nil }})
	v, src, changed, err := r.Resolve(context.Background())
	if err != nil || !changed || src != "manual" || v != (V{2, 3000, 1052747707}) || *applied != v {
		t.Fatalf("v=%v src=%s changed=%v err=%v", v, src, changed, err)
	}
}

func TestManualInvalidaCaiNoAutomatico(t *testing.T) {
	r, _ := newR(Setting{Mode: ModeManual, Manual: "lixo"}, nil, Source{"oficial", fixed(V{2, 3000, 5000}, nil)})
	v, src, _, err := r.Resolve(context.Background())
	if src != "oficial" || v != (V{2, 3000, 5000}) || err == nil || !strings.Contains(err.Error(), "manual inválida") {
		t.Fatalf("v=%v src=%s err=%v", v, src, err)
	}
}

func TestSairDoManualSemRedeVoltaParaEmbutida(t *testing.T) {
	r, applied := newR(Setting{Mode: ModeAuto}, nil, Source{"oficial", fixed(V{}, errors.New("x"))})
	*applied = V{2, 3000, 9999}
	v, src, changed, _ := r.Resolve(context.Background())
	if !changed || src != "embutida" || v != embedded {
		t.Fatalf("v=%v src=%s changed=%v", v, src, changed)
	}
}

func TestErroDeLeituraDaConfiguracaoNaoDerrubaAConexao(t *testing.T) {
	r, _ := newR(Setting{Mode: ModeManual, Manual: "2.3000.1"}, errors.New("permission denied"), Source{"oficial", fixed(V{2, 3000, 5000}, nil)})
	v, src, _, err := r.Resolve(context.Background())
	if src != "oficial" || v != (V{2, 3000, 5000}) || err == nil {
		t.Fatalf("v=%v src=%s err=%v", v, src, err)
	}
}

func TestParseVersion(t *testing.T) {
	v, err := ParseVersion("2.3000.1052747707-alpha")
	if err != nil || v != (V{2, 3000, 1052747707}) {
		t.Fatalf("v=%v err=%v", v, err)
	}
	for _, bad := range []string{"", "abc", "1.2", "3.3000.1", "0.0.0", "2.3000.x"} {
		if _, err := ParseVersion(bad); err == nil {
			t.Fatalf("%q deveria ser inválida", bad)
		}
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

func TestRefreshLimitaFrequenciaMasForceIgnora(t *testing.T) {
	calls := 0
	r, _ := newR(Setting{Mode: ModeAuto}, nil, Source{"a", func(context.Context) (V, error) { calls++; return V{2, 3000, 2000}, nil }})
	if !r.Refresh(context.Background(), false) || r.Refresh(context.Background(), false) || calls != 1 {
		t.Fatalf("a segunda chamada seguida deveria ser ignorada (calls=%d)", calls)
	}
	r.Refresh(context.Background(), true)
	if calls != 2 {
		t.Fatalf("force deveria consultar de novo (calls=%d)", calls)
	}
}

func TestStatusMostraAsQuatroVersoes(t *testing.T) {
	r, _ := newR(Setting{Mode: ModeAuto}, nil,
		Source{"oficial", fixed(V{2, 3000, 3000}, nil)},
		Source{"wa-version", fixed(V{}, errors.New("sem rede"))})
	st := r.Status(context.Background())
	if st.Embedded != "2.3000.1000" || st.Official.Version != "2.3000.3000" || st.Registry.Error != "sem rede" || st.Effective != "2.3000.1000" {
		t.Fatalf("status inesperado: %+v", st)
	}
}
