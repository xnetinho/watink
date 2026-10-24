package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// githubAPIBase é variável só para o teste apontar para um servidor local.
var githubAPIBase = "https://api.github.com"

// defaultReleasesRepo é o repositório cujas releases (publicadas por release.yml) indicam a versão disponível.
const defaultReleasesRepo = "xnetinho/watink"

var releasesRepoRe = regexp.MustCompile(`^[A-Za-z0-9._-]+/[A-Za-z0-9._-]+$`)

type githubRelease struct {
	TagName string `json:"tag_name"`
	Body    string `json:"body"`
}

func releasesRepo() string {
	if r := os.Getenv("WATINK_RELEASES_REPO"); releasesRepoRe.MatchString(r) {
		return r
	}
	return defaultReleasesRepo
}

// changelogLines extrai os itens de lista ("- x" ou "* x") do corpo da release.
func changelogLines(body string) []string {
	lines := []string{}
	for _, l := range strings.Split(body, "\n") {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "- ") || strings.HasPrefix(l, "* ") {
			lines = append(lines, strings.TrimSpace(l[2:]))
		}
	}
	return lines
}

// GetLatestRelease informa a última release publicada. É só informativo: sem release ou com a API indisponível,
// responde 200 com version "-" para a tela degradar sem erro.
func GetLatestRelease(c *gin.Context) {
	empty := gin.H{"version": "-", "changelog": []string{}, "breaking": false, "min_compatible_from": "", "migration_notes": ""}
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(fmt.Sprintf("%s/repos/%s/releases/latest", githubAPIBase, releasesRepo()))
	if err != nil {
		c.JSON(http.StatusOK, empty)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		c.JSON(http.StatusOK, empty)
		return
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil || rel.TagName == "" {
		c.JSON(http.StatusOK, empty)
		return
	}

	empty["version"] = rel.TagName
	empty["changelog"] = changelogLines(rel.Body)
	c.JSON(http.StatusOK, empty)
}

// StartUpdate não atualiza nada: o Watink roda em contêineres e a atualização é feita pelo operador com
// `docker compose pull && docker compose up -d`. Antes era um stub que ligava a manutenção e encerrava o processo.
func StartUpdate(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "update_not_supported",
		"message": "Atualize puxando as novas imagens: docker compose pull && docker compose up -d",
	})
}
