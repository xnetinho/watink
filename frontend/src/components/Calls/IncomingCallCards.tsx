import React, { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { MAX_CARDS, shouldRing, visibleCards } from "@/lib/calls/ringCard";
import { t } from "@/lib/calls/t";
import { useCalls } from "@/context/Calls/CallsContext";
import ringtone from "@/assets/sound.mp3";
import IncomingCallCard from "./IncomingCallCard";
import { useCallCardDock } from "./CallCardDock";

/**
 * Cartões das chamadas que tocam agora. Encaixa no rodapé da lista de tickets quando ela está na tela e flutua no canto inferior
 * esquerdo nas demais páginas. O toque sonoro é um só para todos e para quando todas foram ignoradas ou o operador entra em chamada.
 */
const IncomingCallCards: React.FC<{ connectionName?: (connectionId: number) => string | undefined }> = ({ connectionName }) => {
  const { ringing, active } = useCalls();
  const { host } = useCallCardDock();
  const audioRef = useRef<HTMLAudioElement | null>(null);
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    if (ringing.length === 0) return undefined;
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, [ringing.length]);

  const inCall = !!active && active.phase !== "ended";
  const ring = shouldRing(
    ringing.map((c) => c.callId),
    new Set(ringing.filter((c) => c.ignored).map((c) => c.callId)),
    inCall,
  );
  useEffect(() => {
    const el = audioRef.current;
    if (!el) return undefined;
    if (ring) {
      el.loop = true;
      void el.play().catch(() => undefined);
    } else {
      el.pause();
      el.currentTime = 0;
    }
    return () => el.pause();
  }, [ring]);

  const { shown, hidden } = visibleCards(ringing, MAX_CARDS);
  const cards = (docked: boolean) => (
    <>
      {shown.map((c) => (
        <IncomingCallCard key={c.callId} call={c} now={now} connection={connectionName?.(c.connectionId)} docked={docked} />
      ))}
      {hidden > 0 && (
        <p className="text-center text-xs text-muted-foreground" data-testid="more-calls">
          +{hidden}
        </p>
      )}
    </>
  );

  return (
    <>
      <audio ref={audioRef} src={ringtone} preload="auto" aria-label={t("calls.incoming.title")} />
      {shown.length > 0 &&
        (host ? (
          createPortal(cards(true), host)
        ) : (
          <div className="fixed bottom-4 left-4 z-50 flex w-[290px] flex-col gap-2" data-testid="call-cards-floating">
            {cards(false)}
          </div>
        ))}
    </>
  );
};

export default IncomingCallCards;
