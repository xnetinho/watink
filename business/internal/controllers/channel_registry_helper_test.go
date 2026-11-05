package controllers

import (
	whatsappChannel "github.com/alltomatos/watinkdev/business/internal/channels/whatsapp"
	"github.com/alltomatos/watinkdev/business/internal/domain"
)

// useWhatsAppChannel registra o adaptador do WhatsApp no registro de canais com o publisher do teste: em produção o registro
// é feito uma vez em routes.go, e o envio de qualquer canal passa por ele.
func useWhatsAppChannel(pub domain.CommandPublisher) {
	domain.DefaultChannelRegistry().Register(whatsappChannel.NewAdapter(nil, pub))
}
