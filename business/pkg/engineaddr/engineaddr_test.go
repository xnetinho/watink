package engineaddr

import "testing"

func clear(t *testing.T) {
	for _, k := range []string{"ENGINE_HOST", "ENGINE_HEALTH_URL", "GROUPS_API_URL", "CALLS_AUDIO_URL"} {
		t.Setenv(k, "")
	}
}

func TestDerivesEveryAddressFromHost(t *testing.T) {
	clear(t)
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

func TestOldVariablesOverrideOneByOne(t *testing.T) {
	clear(t)
	t.Setenv("ENGINE_HOST", "watink-engine")
	t.Setenv("CALLS_AUDIO_URL", "ws://outro:9000")
	if got := CallsAudioURL(); got != "ws://outro:9000" {
		t.Fatalf("calls deveria usar a sobreposição: %q", got)
	}
	if got := GroupsURL(); got != "http://watink-engine:8084" {
		t.Fatalf("groups não pode ser afetado pela sobreposição de calls: %q", got)
	}
}

func TestOldVariablesWorkWithoutHost(t *testing.T) {
	clear(t)
	t.Setenv("GROUPS_API_URL", "http://watink-engine:8084")
	if got := GroupsURL(); got != "http://watink-engine:8084" {
		t.Fatalf("groups: %q", got)
	}
	if HealthURL() != "" || CallsAudioURL() != "" {
		t.Fatal("sem ENGINE_HOST e sem sobreposição o endereço fica vazio (recurso desligado)")
	}
}

func TestHostIsNormalized(t *testing.T) {
	clear(t)
	for _, in := range []string{"  watink-engine  ", "http://watink-engine", "ws://watink-engine/", "https://watink-engine//"} {
		t.Setenv("ENGINE_HOST", in)
		if got := Host(); got != "watink-engine" {
			t.Fatalf("%q -> %q", in, got)
		}
	}
}
