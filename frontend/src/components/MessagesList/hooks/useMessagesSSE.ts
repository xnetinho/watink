import { useEffect, MutableRefObject } from "react";
import api from "../../../services/api";
import { subscribeToSocket } from "../../../services/sse-client";
import { Message, MessagesAction } from "../types";

export function useMessagesSocket(
  ticketId: string | number,
  dispatch: React.Dispatch<MessagesAction>,
  shouldScrollRef: MutableRefObject<"smooth" | "auto" | null>
): void {
  useEffect(() => {
    const handleAppMessage = (data: { action: string; message: Message }) => {
      if (String(data.message?.ticketId) !== String(ticketId)) return;
      if (data.action === "create") {
        dispatch({ type: "ADD_MESSAGE", payload: data.message });
        shouldScrollRef.current = "smooth";
        // A conversa está aberta e à vista: a mensagem que acabou de chegar já foi lida. Sem isto o
        // contador só zerava ao abrir o ticket e crescia a cada mensagem nova com a conversa aberta.
        if (!data.message.fromMe && document.visibilityState === "visible") {
          api.put(`/tickets/${ticketId}`, { unreadMessages: 0 }).catch(() => null);
        }
      }
      if (data.action === "update") {
        dispatch({ type: "UPDATE_MESSAGE", payload: data.message });
      }
    };

    return subscribeToSocket(
      { appMessage: handleAppMessage },
      (socket) => socket.emit("joinChat", ticketId)
    );
  }, [ticketId, dispatch, shouldScrollRef]);
}
