package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	wbot "github.com/alltomatos/watinkdev/contracts/wbot"
	"github.com/stretchr/testify/require"
)

// TestWBotContract_BidirectionalCoverage guarda a superfície do contrato: cada evento tem emissor no engine e bind/handler
// no business; cada comando tem consumidor no engine e construtor no business. O teste usa símbolos/valores compartilhados,
// então adicionar uma constante sem conectar os dois lados falha aqui, em vez de virar descarte silencioso no RabbitMQ.
func TestWBotContract_BidirectionalCoverage(t *testing.T) {
	root := filepath.Dir(businessRoot(t))
	engine := readTree(t, filepath.Join(root, "engine-go"))
	business := readTree(t, filepath.Join(root, "business"))

	events := []struct {
		name, value, producer, listener string
	}{
		{"qrcode", wbot.EventSessionQRCode, `"session.qrcode"`, `"session.qrcode"`},
		{"pairing", wbot.EventSessionPairingCode, `"session.pairing_code"`, `"session.pairing_code"`},
		{"status", wbot.EventSessionStatus, `"session.status"`, `"session.status"`},
		{"history", wbot.EventSessionHistorySync, `"session.history_sync"`, `"session.history_sync"`},
		{"risk", wbot.EventSessionRisk, `"session.risk"`, `"session.risk"`},
		{"received", wbot.EventMessageReceived, `"message.received"`, `"message.received"`},
		{"ack", wbot.EventMessageAck, `"message.ack"`, `"message.ack"`},
		{"revoke", wbot.EventMessageRevoke, `"message.revoke"`, `"message.revoke"`},
		{"reaction", wbot.EventMessageReaction, `"message.reaction"`, `"message.reaction"`},
		{"media", wbot.EventMessageMedia, `"message.media"`, `"message.media"`},
		{"contact update", wbot.EventContactUpdate, `"contact.update"`, `"contact.update"`},
		{"contact import", wbot.EventContactImport, `"contact.import"`, `"contact.import"`},
		{"contact presence", wbot.EventContactPresence, `"contact.presence"`, "EventContactPresence"},
		{"poll vote", wbot.EventMessagePollVote, `"message.poll_vote"`, "EventMessagePollVote"},
	}
	for _, event := range events {
		t.Run("event/"+event.name, func(t *testing.T) {
			if event.producer != "" {
				require.Contains(t, engine, event.producer, "engine emitter missing for %s", event.value)
			}
			require.Contains(t, business, event.listener, "business handler/bind missing for %s", event.value)
		})
	}

	commands := []struct {
		name, value, engineToken, businessToken string
	}{
		{"session start", wbot.CommandSessionStart, "CommandSessionStart", "CommandSessionStart"},
		{"session stop", wbot.CommandSessionStop, "CommandSessionStop", "session.stop"},
		{"session delete", wbot.CommandSessionDelete, "CommandSessionDelete", "session.delete"},
		{"text", wbot.CommandMessageText, "CommandMessageText", "message.send.text"},
		{"media", wbot.CommandMessageMedia, "CommandMessageMedia", "message.send.media"},
		{"poll", wbot.CommandMessagePoll, "CommandMessagePoll", "message.send.poll"},
		{"presence subscribe", wbot.CommandPresenceSubscribe, "CommandPresenceSubscribe", "CommandPresenceSubscribe"},
		{"poll vote", wbot.CommandMessagePollVote, "CommandMessagePollVote", "CommandMessagePollVote"},
		{"react", wbot.CommandMessageReact, "CommandMessageReact", "CommandMessageReact"},
		{"media download", wbot.CommandMediaDownload, "CommandMediaDownload", "CommandMediaDownload"},
		{"contact sync", wbot.CommandContactSync, "CommandContactSync", "CommandContactSync"},
		{"contact import", wbot.CommandContactImport, "CommandContactImport", "CommandContactImport"},
		{"history recover", wbot.CommandHistoryRecover, "CommandHistoryRecover", "CommandHistoryRecover"},
		{"presence", wbot.CommandChatPresence, "CommandChatPresence", "chat.presence"},
	}
	for _, command := range commands {
		t.Run("command/"+command.name, func(t *testing.T) {
			require.Contains(t, engine, command.engineToken, "engine routing key missing for %s", command.value)
			require.Contains(t, business, command.businessToken, "business constructor missing for %s", command.value)
		})
	}
}

func readTree(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	require.NoError(t, filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		b.Write(data)
		b.WriteByte('\n')
		return nil
	}))
	return b.String()
}
