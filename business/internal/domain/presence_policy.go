package domain

// PresencePolicy define quando o número de uma conexão WhatsApp fica "disponível" (online) no WhatsApp.
//
// Disponível tem dois efeitos que importam ao operador: o WhatsApp deixa de empurrar notificações para o celular dele
// (trata o engine como um aparelho ativo, como o WhatsApp Web aberto) e o contato passa a ver o número como online.
// É também o que faz o WhatsApp entregar a presença (online / última vez visto) dos contatos.
type PresencePolicy string

const (
	// PresenceNever: nunca marca disponível. Notificações normais; presença do contato praticamente não chega.
	PresenceNever PresencePolicy = "never"
	// PresenceOnDemand: disponível só enquanto algum operador está com uma conversa desta conexão aberta; depois de
	// PresenceIdleMinutes sem operador, volta a indisponível.
	PresenceOnDemand PresencePolicy = "on_demand"
	// PresenceAlways: sempre disponível (reaplicado a cada conexão da sessão).
	PresenceAlways PresencePolicy = "always"
)

const (
	DefaultPresenceIdleMinutes = 5
	MaxPresenceIdleMinutes     = 240
)

// ParsePresencePolicy valida o valor vindo da API. Vazio vale o padrão (never).
func ParsePresencePolicy(v string) (PresencePolicy, bool) {
	switch PresencePolicy(v) {
	case "":
		return PresenceNever, true
	case PresenceNever, PresenceOnDemand, PresenceAlways:
		return PresencePolicy(v), true
	}
	return PresenceNever, false
}

// NormalizePresenceIdle limita o tempo de espera ao intervalo aceito; fora dele volta ao padrão.
func NormalizePresenceIdle(minutes int) int {
	if minutes < 1 || minutes > MaxPresenceIdleMinutes {
		return DefaultPresenceIdleMinutes
	}
	return minutes
}
