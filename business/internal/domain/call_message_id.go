package domain

import "strings"

// CallMessageIDPrefix marca a mensagem de sistema de uma chamada no Watink: o id dela é "call:<call-id>". O WhatsApp conhece a mesma
// mensagem só por <call-id> (verificado no lt em 7/nov/2026: o alvo de uma reação feita no celular a um registro de chamada era
// exatamente o call-id). Quem fala com o WhatsApp precisa trocar de um nome para o outro.
const CallMessageIDPrefix = "call:"

// CallMessageID é o id da mensagem de sistema da chamada no Watink.
func CallMessageID(callID string) string { return CallMessageIDPrefix + callID }

// WhatsAppMessageID devolve o id que o WhatsApp usa para a mensagem: tira o prefixo do registro de chamada; qualquer outro id passa
// igual.
func WhatsAppMessageID(id string) string { return strings.TrimPrefix(id, CallMessageIDPrefix) }

// WatinkMessageIDs lista os ids sob os quais o Watink pode ter guardado a mensagem que o WhatsApp chama de id: o próprio id e o do
// registro de chamada. Serve para casar um evento que chega do WhatsApp com a mensagem local.
func WatinkMessageIDs(id string) []string {
	if strings.HasPrefix(id, CallMessageIDPrefix) {
		return []string{id}
	}
	return []string{id, CallMessageID(id)}
}
