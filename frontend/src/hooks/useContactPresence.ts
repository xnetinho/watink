import { useEffect, useState } from "react";
import api from "../services/api";
import { subscribeToSocket } from "../services/sse-client";
import type { Presence } from "../lib/presence/format";

/**
 * Presença (online / última vez visto) do contato de um ticket. Pede a assinatura ao abrir a conversa e escuta as
 * atualizações. Só o WhatsApp informa, e só se o contato permitir: sem resposta, devolve undefined e nada é mostrado.
 */
export function useContactPresence(ticketId: number | string | undefined, ids: Array<string | null | undefined>, enabled = true): Presence | undefined {
  const [presence, setPresence] = useState<Presence | undefined>();

  useEffect(() => {
    setPresence(undefined);
    // O WhatsApp pode identificar o contato pelo telefone ou pelo LID: aceita os dois.
    const mine = ids.map((i) => (i ?? "").split("@")[0].replace(/\D/g, "")).filter(Boolean);
    if (!enabled || !ticketId || mine.length === 0) return;
    // O WhatsApp só avisa quando a presença muda: a resposta traz o último estado conhecido, se houver.
    api
      .post<{ presence?: { online: boolean; lastSeen?: number } | null }>(`/tickets/${ticketId}/presence`)
      .then(({ data }) => {
        if (data?.presence) setPresence({ online: data.presence.online, lastSeen: data.presence.lastSeen || undefined });
      })
      .catch(() => null);
    return subscribeToSocket({
      contactPresence: (d: { number?: string; online: boolean; lastSeen?: number }) => {
        if (!d.number || !mine.includes(d.number)) return;
        setPresence({ online: d.online, lastSeen: d.lastSeen || undefined });
      },
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps -- ids é uma lista nova a cada render; a chave é o conteúdo
  }, [ticketId, ids.join("|"), enabled]);

  return presence;
}
