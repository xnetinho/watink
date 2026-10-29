package services

import (
	"os"
	"regexp"
	"testing"

	wbot "github.com/alltomatos/watinkdev/contracts/wbot"
)

// Todo `case "x":` de HandleEvent precisa ter o evento assinado em eventRoutingKeys: sem assinatura o broker nunca entrega
// o evento e o tratador vira código morto (contact.saved ficou assim e o contato ficou "Gravando..." para sempre).
func TestEveryHandledEngineEventIsSubscribed(t *testing.T) {
	src, err := os.ReadFile("event_listener.go")
	if err != nil {
		t.Fatal(err)
	}
	subscribed := map[string]bool{}
	for _, k := range eventRoutingKeys() {
		subscribed[k] = true
	}
	handled := regexp.MustCompile(`case "([a-z]+\.[a-z_.]+)":`).FindAllStringSubmatch(string(src), -1)
	if len(handled) < 10 {
		t.Fatalf("regex achou só %d tratadores; o teste ficou cego", len(handled))
	}
	for _, m := range handled {
		if m[1] == "engine.ready" || !subscribed[wbot.EventPattern(m[1])] {
			if m[1] == "engine.ready" {
				continue
			}
			t.Errorf("tratador de %q sem assinatura em eventRoutingKeys", m[1])
		}
	}
}
