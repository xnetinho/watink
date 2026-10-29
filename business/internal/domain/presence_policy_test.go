package domain

import "testing"

func TestParsePresencePolicy(t *testing.T) {
	for in, want := range map[string]PresencePolicy{"": PresenceNever, "never": PresenceNever, "on_demand": PresenceOnDemand, "always": PresenceAlways} {
		got, ok := ParsePresencePolicy(in)
		if !ok || got != want {
			t.Errorf("%q: %q %v, esperado %q", in, got, ok, want)
		}
	}
	for _, bad := range []string{"ALWAYS", "on-demand", "sempre", "1", " never"} {
		if _, ok := ParsePresencePolicy(bad); ok {
			t.Errorf("%q deveria ser recusado", bad)
		}
	}
}

func TestNormalizePresenceIdle(t *testing.T) {
	for in, want := range map[int]int{0: 5, -3: 5, 1: 1, 30: 30, 240: 240, 241: 5, 99999: 5} {
		if got := NormalizePresenceIdle(in); got != want {
			t.Errorf("%d: %d, esperado %d", in, got, want)
		}
	}
}
