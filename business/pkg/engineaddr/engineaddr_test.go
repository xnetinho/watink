package engineaddr

import "testing"

func TestDerivesEveryAddressFromHost(t *testing.T) {
	t.Setenv("ENGINE_HOST", "watink-engine")
	if got := HealthURL(); got != "http://watink-engine:8083/health" {
		t.Fatalf("health: %q", got)
	}
	if got := GroupsURL(); got != "http://watink-engine:8084" {
		t.Fatalf("groups: %q", got)
	}
	if got := CallsAudioURL(); got != "ws://watink-engine:8085" {
		t.Fatalf("calls: %q", got)
	}
}

func TestWithoutHostEveryAddressIsEmpty(t *testing.T) {
	t.Setenv("ENGINE_HOST", "")
	if HealthURL() != "" || GroupsURL() != "" || CallsAudioURL() != "" {
		t.Fatal("sem ENGINE_HOST o endereço fica vazio (recurso desligado)")
	}
}

// As variáveis antigas deixaram de existir: definir uma delas não pode mudar nada.
func TestOldVariablesAreIgnored(t *testing.T) {
	t.Setenv("ENGINE_HOST", "watink-engine")
	t.Setenv("ENGINE_HEALTH_URL", "http://outro:1/health")
	t.Setenv("GROUPS_API_URL", "http://outro:2")
	t.Setenv("CALLS_AUDIO_URL", "ws://outro:3")
	if HealthURL() != "http://watink-engine:8083/health" || GroupsURL() != "http://watink-engine:8084" || CallsAudioURL() != "ws://watink-engine:8085" {
		t.Fatalf("variáveis antigas não podem sobrepor: %q %q %q", HealthURL(), GroupsURL(), CallsAudioURL())
	}
	t.Setenv("ENGINE_HOST", "")
	if GroupsURL() != "" {
		t.Fatal("sem ENGINE_HOST, GROUPS_API_URL sozinha não liga o recurso")
	}
}

func TestHostIsNormalized(t *testing.T) {
	for _, in := range []string{"  watink-engine  ", "http://watink-engine", "ws://watink-engine/", "https://watink-engine//"} {
		t.Setenv("ENGINE_HOST", in)
		if got := Host(); got != "watink-engine" {
			t.Fatalf("%q -> %q", in, got)
		}
	}
}
