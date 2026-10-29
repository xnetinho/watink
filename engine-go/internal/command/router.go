package command

import (
	"fmt"
	wbot "github.com/alltomatos/watinkdev/contracts/wbot"
	"strings"
)

// ParseRoutingKey extracts tenantID, sessionID string and command type from a routing key
// of the form wbot.Command(tenantId, connectionId, command).
// Returns an error for keys with fewer than 4 segments or unknown command types.
func ParseRoutingKey(routingKey string) (tenantID string, sessionID string, cmd string, err error) {
	parts := strings.Split(routingKey, ".")
	if len(parts) < 4 {
		return "", "", "", fmt.Errorf("invalid routing key: %s", routingKey)
	}
	tenantID = parts[1]
	sessionID = parts[2]
	cmd = strings.Join(parts[3:], ".")

	switch cmd {
	case wbot.CommandSessionStart, wbot.CommandSessionStop, wbot.CommandSessionDelete,
		wbot.CommandMessageText, wbot.CommandMessageMedia,
		wbot.CommandMessageButtons, wbot.CommandMessageList, wbot.CommandMessagePoll, wbot.CommandMessagePollVote, wbot.CommandMessageInteractive, wbot.CommandMessageTemplate, wbot.CommandMessageCarousel,
		wbot.CommandMessageReact,
		wbot.CommandMessageEdit,
		wbot.CommandMessageMarkRead, wbot.CommandMediaDownload,
		wbot.CommandContactSync, wbot.CommandContactImport, wbot.CommandContactSave, wbot.CommandHistoryRecover,
		wbot.CommandChatPresence, wbot.CommandPresenceSubscribe, wbot.CommandPresencePolicy,
		wbot.CommandCallReady, wbot.CommandCallAccept, wbot.CommandCallReject, wbot.CommandCallEnd, wbot.CommandCallStart:
		return tenantID, sessionID, cmd, nil
	default:
		return "", "", "", fmt.Errorf("unknown command type: %s", cmd)
	}
}
