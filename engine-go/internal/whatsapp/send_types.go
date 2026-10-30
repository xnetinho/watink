package whatsapp

// Payload types — exported so main.go can unmarshal directly into these structs.

// QuotedMedia descreve a mídia citada para o celular desenhar a caixa de citação como o cliente oficial (áudio com
// duração e microfone, imagem com miniatura), em vez de só o texto do rótulo. Não leva chaves: citar não reenvia a mídia.
type QuotedMedia struct {
	Type      string `json:"type"` // audio | image | video | document | sticker
	MimeType  string `json:"mimeType,omitempty"`
	Voice     bool   `json:"voice,omitempty"`
	Seconds   int    `json:"seconds,omitempty"`
	Caption   string `json:"caption,omitempty"`
	FileName  string `json:"fileName,omitempty"`
	Thumbnail string `json:"thumbnail,omitempty"` // JPEG em base64
}

type TextCommandPayload struct {
	SessionID   int          `json:"sessionId"`
	MessageID   string       `json:"messageId"`
	To          string       `json:"to"`
	Body        string       `json:"body"`
	QuotedMsgID string       `json:"quotedMsgId,omitempty"`
	QuotedJID   string       `json:"quotedJid,omitempty"`
	QuotedBody  string       `json:"quotedBody,omitempty"`
	QuotedMedia *QuotedMedia `json:"quotedMedia,omitempty"`
	Mentions    []string     `json:"mentions,omitempty"`
	// Forwarded marca a mensagem como encaminhada (selo "Encaminhada"); ForwardScore é quantas vezes já foi.
	Forwarded    bool `json:"forwarded,omitempty"`
	ForwardScore int  `json:"forwardScore,omitempty"`
}

type MediaCommandPayload struct {
	SessionID    int          `json:"sessionId"`
	MessageID    string       `json:"messageId"`
	To           string       `json:"to"`
	Body         string       `json:"body"`
	MediaURL     string       `json:"mediaUrl"`
	MediaType    string       `json:"mediaType"`
	MimeType     string       `json:"mimeType"`
	FileName     string       `json:"fileName"`
	MediaData    string       `json:"mediaData"`
	QuotedMsgID  string       `json:"quotedMsgId,omitempty"`
	QuotedJID    string       `json:"quotedJid,omitempty"`
	QuotedBody   string       `json:"quotedBody,omitempty"`
	QuotedMedia  *QuotedMedia `json:"quotedMedia,omitempty"`
	Mentions     []string     `json:"mentions,omitempty"`
	Forwarded    bool         `json:"forwarded,omitempty"`
	ForwardScore int          `json:"forwardScore,omitempty"`
	// ViewOnce envia a mídia como visualização única (foto, vídeo ou áudio). Ignorado nos demais tipos.
	ViewOnce bool `json:"viewOnce,omitempty"`
}

// ReactionCommandPayload sends (or removes, when Reaction=="") an emoji
// reaction to an existing message. TargetFromMe tells SendReaction who sent
// the message being reacted to: our own client JID when true (we sent it),
// or the chat JID (To) otherwise -- whatsmeow's BuildReaction needs that
// sender, not the reactor.
type ReactionCommandPayload struct {
	SessionID    int    `json:"sessionId"`
	MessageID    string `json:"messageId"`
	To           string `json:"to"`
	TargetMsgID  string `json:"targetMsgId"`
	TargetFromMe bool   `json:"targetFromMe"`
	Reaction     string `json:"reaction"`
}

// EditCommandPayload edita uma mensagem NOSSA já enviada. TargetMsgID é o id da mensagem original; Body é o texto novo
// (ou a legenda nova, quando a original é uma mídia). MessageID identifica o comando para os acks.
type EditCommandPayload struct {
	SessionID   int    `json:"sessionId"`
	MessageID   string `json:"messageId"`
	To          string `json:"to"`
	TargetMsgID string `json:"targetMsgId"`
	Body        string `json:"body"`
	// MediaType da original ("", "image", "video", "document"): define se Body vira texto ou legenda.
	MediaType string `json:"mediaType,omitempty"`
}

// PresenceCommandPayload sets the chat-composing indicator ("digitando...")
// for one chat. State is "composing" (start typing) or "paused" (stop —
// sent automatically before the real message goes out, or on cancel).
type PresenceCommandPayload struct {
	SessionID int    `json:"sessionId"`
	To        string `json:"to"`
	State     string `json:"state"`
}

// PresenceSubscribePayload pede ao WhatsApp as atualizações de online/última vez visto de um contato.
type PresenceSubscribePayload struct {
	SessionID int    `json:"sessionId"`
	To        string `json:"to"`
}

// PresencePolicyPayload muda a política de presença de uma sessão em andamento (never | on_demand | always).
type PresencePolicyPayload struct {
	SessionID   int    `json:"sessionId"`
	Policy      string `json:"policy"`
	IdleMinutes int    `json:"idleMinutes"`
}

type MarkReadCommandPayload struct {
	ChatJID    string   `json:"chatJid"`
	SenderJID  string   `json:"senderJid"`
	MessageIDs []string `json:"messageIds"`
}

// ButtonsCommandPayload carries a legacy ButtonsMessage (up to 3 buttons).
type ButtonsCommandPayload struct {
	SessionID   int             `json:"sessionId"`
	MessageID   string          `json:"messageId"`
	To          string          `json:"to"`
	ContentText string          `json:"contentText"`
	FooterText  string          `json:"footerText,omitempty"`
	Buttons     []ButtonPayload `json:"buttons"`
}

type ButtonPayload struct {
	ID          string `json:"id"`
	DisplayText string `json:"displayText"`
}

// ListCommandPayload carries a ListMessage with sections and rows.
type ListCommandPayload struct {
	SessionID   int                  `json:"sessionId"`
	MessageID   string               `json:"messageId"`
	To          string               `json:"to"`
	Title       string               `json:"title"`
	ButtonText  string               `json:"buttonText"`
	Description string               `json:"description,omitempty"`
	FooterText  string               `json:"footerText,omitempty"`
	Sections    []ListSectionPayload `json:"sections"`
}

type ListSectionPayload struct {
	Title string           `json:"title"`
	Rows  []ListRowPayload `json:"rows"`
}

type ListRowPayload struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// PollCommandPayload carries a poll creation message.
type PollCommandPayload struct {
	SessionID       int      `json:"sessionId"`
	MessageID       string   `json:"messageId"`
	To              string   `json:"to"`
	Name            string   `json:"name"`
	Options         []string `json:"options"`
	SelectableCount int      `json:"selectableCount"`
}

// PollVoteCommandPayload: voto em uma enquete recebida (ou enviada por nós). Chat, PollSender e PollID identificam a
// mensagem original; Options são os NOMES das opções escolhidas (lista vazia remove o voto).
type PollVoteCommandPayload struct {
	SessionID  int      `json:"sessionId"`
	Chat       string   `json:"chat"`
	PollID     string   `json:"pollId"`
	PollSender string   `json:"pollSender"`
	PollFromMe bool     `json:"pollFromMe"`
	Options    []string `json:"options"`
}

// InteractiveCommandPayload carries a NativeFlow (modern interactive) message.
type InteractiveCommandPayload struct {
	SessionID  int                        `json:"sessionId"`
	MessageID  string                     `json:"messageId"`
	To         string                     `json:"to"`
	BodyText   string                     `json:"bodyText"`
	FooterText string                     `json:"footerText,omitempty"`
	Buttons    []InteractiveButtonPayload `json:"buttons"`
}

type InteractiveButtonPayload struct {
	Name   string `json:"name"`
	Params string `json:"params"`
}

// SyncContactPayload carries a contact sync request.
// SaveContactPayload grava um contato criado no Watink na agenda do celular da conexão.
type SaveContactPayload struct {
	SessionID int    `json:"sessionId"`
	Number    string `json:"number"`
	Name      string `json:"name"`
}

type SyncContactPayload struct {
	SessionID int    `json:"sessionId"`
	Number    string `json:"number"`
}

// CarouselCommandPayload carries an interactive carousel: multiple cards, each with
// an image header + body text + NativeFlow buttons. Renders on personal accounts
// via the same <biz> native_flow node trick used by SendInteractive.
type CarouselCommandPayload struct {
	SessionID int            `json:"sessionId"`
	MessageID string         `json:"messageId"`
	To        string         `json:"to"`
	BodyText  string         `json:"bodyText,omitempty"`
	Cards     []CarouselCard `json:"cards"`
}

type CarouselCard struct {
	ImageURL  string                     `json:"imageUrl,omitempty"`
	ImageData string                     `json:"imageData,omitempty"` // base64 alternative
	Title     string                     `json:"title,omitempty"`     // card body text
	Footer    string                     `json:"footer,omitempty"`
	Buttons   []InteractiveButtonPayload `json:"buttons,omitempty"` // {name, params} (NativeFlow)
}

// TemplateCommandPayload carries a HydratedFourRowTemplate message.
// Supports up to 3 buttons: quickreply, url, or call.
type TemplateCommandPayload struct {
	SessionID   int              `json:"sessionId"`
	MessageID   string           `json:"messageId"`
	To          string           `json:"to"`
	ContentText string           `json:"contentText"`
	FooterText  string           `json:"footerText,omitempty"`
	Buttons     []TemplateButton `json:"buttons"`
}

type TemplateButton struct {
	// Type: "quickreply" | "url" | "call"
	Type        string `json:"type"`
	DisplayText string `json:"displayText"`
	ID          string `json:"id,omitempty"`          // quickreply
	URL         string `json:"url,omitempty"`         // url
	PhoneNumber string `json:"phoneNumber,omitempty"` // call
	Index       uint32 `json:"index"`
}
