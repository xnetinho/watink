package architecture

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Duas regras de infraestrutura que o compilador não impõe e cuja violação só aparece em produção:
//
//  1. Toda imagem base dos Dockerfiles fica fixada por digest. Com tag móvel (alpine:latest, golang:1.25rc1) dois builds
//     do mesmo commit podem produzir binários diferentes, e a imagem de runtime muda sem ninguém pedir.
//  2. As portas 8084 (API de grupos) e 8085 (áudio das chamadas) do engine NÃO TÊM autenticação: a defesa é a rede
//     interna do Docker. Publicá-las em `ports:` as exporia ao host.

var fromRe = regexp.MustCompile(`(?mi)^\s*FROM\s+(?:--platform=\S+\s+)?(\S+)(?:\s+AS\s+\S+)?\s*$`)

// infraRoot é a raiz do repositório (pai de business/).
func infraRoot(t *testing.T) string {
	t.Helper()
	return filepath.Dir(businessRoot(t))
}

func dockerfiles(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	require.NoError(t, filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", ".git", "build", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if n := d.Name(); n == "Dockerfile" || strings.HasPrefix(n, "Dockerfile.") {
			out = append(out, path)
		}
		return nil
	}))
	return out
}

func TestDockerfiles_ImagemBaseFixadaPorDigest(t *testing.T) {
	root := infraRoot(t)
	files := dockerfiles(t, root)
	require.NotEmpty(t, files)
	var bad []string
	for _, f := range files {
		b, err := os.ReadFile(f)
		require.NoError(t, err)
		stages := map[string]bool{}
		for _, m := range regexp.MustCompile(`(?mi)^\s*FROM\s+\S+\s+AS\s+(\S+)`).FindAllStringSubmatch(string(b), -1) {
			stages[strings.ToLower(m[1])] = true
		}
		for _, m := range fromRe.FindAllStringSubmatch(string(b), -1) {
			img := m[1]
			if stages[strings.ToLower(img)] || strings.EqualFold(img, "scratch") {
				continue // estágio anterior do próprio Dockerfile
			}
			if !strings.Contains(img, "@sha256:") {
				rel, _ := filepath.Rel(root, f)
				bad = append(bad, rel+": FROM "+img)
			}
		}
	}
	assert.Empty(t, bad, "imagem base sem digest (@sha256:...): fixe a imagem para o build ser reproduzível")
}

func TestDockerfiles_NaoUsamTagLatestNemReleaseCandidate(t *testing.T) {
	root := infraRoot(t)
	var bad []string
	for _, f := range dockerfiles(t, root) {
		b, err := os.ReadFile(f)
		require.NoError(t, err)
		for _, m := range fromRe.FindAllStringSubmatch(string(b), -1) {
			tag := strings.ToLower(m[1])
			if i := strings.Index(tag, "@"); i >= 0 {
				tag = tag[:i]
			}
			if strings.HasSuffix(tag, ":latest") || strings.Contains(tag, "rc") && regexp.MustCompile(`\d+rc\d+`).MatchString(tag) {
				rel, _ := filepath.Rel(root, f)
				bad = append(bad, rel+": FROM "+m[1])
			}
		}
	}
	assert.Empty(t, bad, "tag móvel (latest) ou release candidate em imagem de produção")
}

// internalPortRe pega 8084 e 8085 em qualquer forma de publicação ("8084:8084", "127.0.0.1:8085:8085", "8084").
var internalPortRe = regexp.MustCompile(`\b(8084|8085)\b`)

func TestCompose_PortasInternasDoEngineNaoSaoPublicadas(t *testing.T) {
	root := infraRoot(t)
	files, err := filepath.Glob(filepath.Join(root, "docker-compose*.yml"))
	require.NoError(t, err)
	require.NotEmpty(t, files)

	var bad []string
	for _, f := range files {
		b, err := os.ReadFile(f)
		require.NoError(t, err)
		inPorts, portsIndent := false, 0
		for i, line := range strings.Split(string(b), "\n") {
			trimmed := strings.TrimSpace(line)
			indent := len(line) - len(strings.TrimLeft(line, " "))
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}
			if trimmed == "ports:" {
				inPorts, portsIndent = true, indent
				continue
			}
			if inPorts {
				if strings.HasPrefix(trimmed, "-") && indent >= portsIndent {
					// ignora comentário no fim da linha
					item := trimmed
					if j := strings.Index(item, "#"); j >= 0 {
						item = item[:j]
					}
					if internalPortRe.MatchString(item) {
						rel, _ := filepath.Rel(root, f)
						bad = append(bad, rel+":"+itoa(i+1)+": "+trimmed)
					}
					continue
				}
				inPorts = false
			}
		}
	}
	assert.Empty(t, bad, "porta 8084/8085 publicada em `ports:`: elas não têm autenticação e só podem usar `expose:` (rede interna)")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}
