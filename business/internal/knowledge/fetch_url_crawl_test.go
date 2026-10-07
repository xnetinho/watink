package knowledge

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pageWithLink builds a page long enough to pass minExtractedLen, linking to
// the given path.
func pageWithLink(path string) string {
	body := "<html><head><title>P</title></head><body><article><h1>P</h1>"
	for i := 0; i < 40; i++ {
		body += "<p>Conteúdo de teste repetido para passar do limite mínimo de extração configurado.</p>"
	}
	if path != "" {
		body += fmt.Sprintf(`<a href="%s">next</a>`, path)
	}
	body += "</article></body></html>"
	return body
}

func TestCrawlSite_BFS_SameDomain(t *testing.T) {
	withPlainDialer(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(pageWithLink("/page2")))
	})
	mux.HandleFunc("/page2", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(pageWithLink("")))
	})
	mux.HandleFunc("/sitemap.xml", http.NotFound)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	docs, err := CrawlSite(context.Background(), srv.URL+"/", CrawlOptions{MaxPages: 5, MaxDepth: 2})
	require.NoError(t, err)
	assert.Len(t, docs, 2)
}

func TestCrawlSite_RespectsMaxPages(t *testing.T) {
	withPlainDialer(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(pageWithLink("/page2")))
	})
	mux.HandleFunc("/page2", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(pageWithLink("/page3")))
	})
	mux.HandleFunc("/page3", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(pageWithLink("")))
	})
	mux.HandleFunc("/sitemap.xml", http.NotFound)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	docs, err := CrawlSite(context.Background(), srv.URL+"/", CrawlOptions{MaxPages: 1, MaxDepth: 2})
	require.NoError(t, err)
	assert.Len(t, docs, 1)
}

func TestCrawlSite_UsesSitemapWhenPresent(t *testing.T) {
	withPlainDialer(t)
	mux := http.NewServeMux()
	var sitemapXML string
	mux.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(sitemapXML))
	})
	mux.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(pageWithLink("")))
	})
	mux.HandleFunc("/b", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(pageWithLink("")))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	sitemapXML = fmt.Sprintf(`<?xml version="1.0"?><urlset><url><loc>%s/a</loc></url><url><loc>%s/b</loc></url></urlset>`, srv.URL, srv.URL)

	docs, err := CrawlSite(context.Background(), srv.URL+"/", CrawlOptions{MaxPages: 5, MaxDepth: 2})
	require.NoError(t, err)
	assert.Len(t, docs, 2)
}

// Ressalvas CONHECIDAS do crawler (docs/agents/knowledge-base.md, "Crawl de site"). Estes testes registram o
// comportamento ATUAL para que a documentação não seja um palpite; ao corrigir, inverta a asserção.

// O filtro de mesmo domínio vale só no BFS: um sitemap que lista URL de OUTRO domínio é seguido sem filtro.
func TestCrawlSite_Sitemap_FollowsOtherDomain_KnownLimitation(t *testing.T) {
	withPlainDialer(t)
	other := http.NewServeMux()
	other.HandleFunc("/x", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(pageWithLink(""))) })
	otherSrv := httptest.NewServer(other)
	defer otherSrv.Close()

	mux := http.NewServeMux()
	var sitemapXML string
	mux.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(sitemapXML)) })
	mux.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(pageWithLink(""))) })
	srv := httptest.NewServer(mux)
	defer srv.Close()
	// Hosts DIFERENTES de verdade: sameHost compara só o nome (sem a porta), e os dois httptest são 127.0.0.1.
	otherHost := strings.Replace(otherSrv.URL, "127.0.0.1", "localhost", 1)
	sitemapXML = fmt.Sprintf(`<?xml version="1.0"?><urlset><url><loc>%s/a</loc></url><url><loc>%s/x</loc></url></urlset>`, srv.URL, otherHost)

	docs, err := CrawlSite(context.Background(), srv.URL+"/", CrawlOptions{MaxPages: 5})
	require.NoError(t, err)
	assert.Len(t, docs, 2, "hoje o sitemap segue a URL de outro host (limitação conhecida); ao corrigir, espere 1")
}

// <sitemapindex> (índice de sitemaps) não é suportado: nenhuma URL sai e o crawl cai no BFS.
func TestCrawlSite_Sitemapindex_NotSupported_KnownLimitation(t *testing.T) {
	withPlainDialer(t)
	mux := http.NewServeMux()
	var indexXML string
	mux.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(indexXML)) })
	mux.HandleFunc("/sitemap-pages.xml", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf(`<urlset><url><loc>%s/p1</loc></url><url><loc>%s/p2</loc></url></urlset>`, "http://"+r.Host, "http://"+r.Host)))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(pageWithLink(""))) })
	mux.HandleFunc("/p1", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(pageWithLink(""))) })
	mux.HandleFunc("/p2", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(pageWithLink(""))) })
	srv := httptest.NewServer(mux)
	defer srv.Close()
	indexXML = fmt.Sprintf(`<?xml version="1.0"?><sitemapindex><sitemap><loc>%s/sitemap-pages.xml</loc></sitemap></sitemapindex>`, srv.URL)

	docs, err := CrawlSite(context.Background(), srv.URL+"/", CrawlOptions{MaxPages: 10})
	require.NoError(t, err)
	got := map[string]bool{}
	for _, d := range docs {
		got[d.URL] = true
	}
	assert.False(t, got[srv.URL+"/p1"] || got[srv.URL+"/p2"], "hoje as páginas listadas dentro do índice NÃO são lidas (limitação conhecida)")
}

// O BFS não normaliza #fragmento: a mesma página com fragmentos diferentes é baixada duas vezes.
func TestCrawlSite_BFS_DoesNotNormalizeFragment_KnownLimitation(t *testing.T) {
	withPlainDialer(t)
	var hits int
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body := pageWithLink("")
		body = body[:len(body)-len("</article></body></html>")] + `<a href="/p#a">a</a><a href="/p#b">b</a></article></body></html>`
		_, _ = w.Write([]byte(body))
	})
	mux.HandleFunc("/p", func(w http.ResponseWriter, r *http.Request) { hits++; _, _ = w.Write([]byte(pageWithLink(""))) })
	srv := httptest.NewServer(mux)
	defer srv.Close()

	_, err := CrawlSite(context.Background(), srv.URL+"/", CrawlOptions{MaxPages: 10, MaxDepth: 2})
	require.NoError(t, err)
	assert.Greater(t, hits, 2, "hoje /p#a e /p#b são tratadas como páginas distintas (cada uma baixa 2x); ao corrigir, espere 2 (uma página, 2 downloads)")
}

// sameHost compara só o NOME do host: outra PORTA no mesmo host conta como "mesmo domínio".
func TestSameHost_IgnoresPort_KnownLimitation(t *testing.T) {
	a, _ := url.Parse("http://exemplo.com:8080/")
	b, _ := url.Parse("http://exemplo.com:9090/x")
	assert.True(t, sameHost(a, b), "hoje a porta é ignorada (limitação conhecida); ao corrigir, espere false")
	c, _ := url.Parse("http://outro.com/x")
	assert.False(t, sameHost(a, c))
	d, _ := url.Parse("ftp://exemplo.com/x")
	assert.False(t, sameHost(a, d), "só http/https")
}
