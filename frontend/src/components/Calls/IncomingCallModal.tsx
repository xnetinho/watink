import React, { useEffect, useRef } from "react";
import { Phone, PhoneOff } from "lucide-react";
import { Avatar } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { t } from "@/lib/calls/t";
import { useCalls } from "@/context/Calls/CallsContext";
import { displayName } from "@/lib/calls/format";
import ringtone from "@/assets/sound.mp3";

/**
 * Toque de chamada recebida. Aparece sobre QUALQUER tela e só existe para quem tem
 * calls:receive (o provedor já não popula `ringing` sem a permissão). Mostra uma
 * oferta por vez (a mais antiga); as demais esperam a vez.
 */
const IncomingCallModal: React.FC<{ connectionName?: (whatsappId: number) => string | undefined }> = ({
  connectionName,
}) => {
  const { ringing, accept, reject, active } = useCalls();
  const current = ringing[0];
  const audioRef = useRef<HTMLAudioElement | null>(null);

  // Som em loop enquanto houver toque e o operador não estiver em outra chamada.
  const shouldRing = !!current && !active;
  useEffect(() => {
    const el = audioRef.current;
    if (!el) return undefined;
    if (shouldRing) {
      el.loop = true;
      void el.play().catch(() => undefined);
    } else {
      el.pause();
      el.currentTime = 0;
    }
    return () => el.pause();
  }, [shouldRing]);

  if (!current) return <audio ref={audioRef} src={ringtone} preload="auto" />;

  const name = displayName(current.contact, t("calls.unknownContact"));
  const connection = connectionName?.(current.whatsappId);
  const busy = !!active;

  return (
    <>
      <audio ref={audioRef} src={ringtone} preload="auto" />
      <Dialog open onOpenChange={() => undefined}>
        <DialogContent
          className="max-w-sm"
          data-testid="incoming-call"
          onInteractOutside={(e) => e.preventDefault()}
          onEscapeKeyDown={(e) => e.preventDefault()}
        >
          <DialogHeader className="items-center text-center">
            <Avatar size="xl" src={current.contact.profilePicUrl} name={name} />
            <DialogTitle className="mt-2">{name}</DialogTitle>
            <DialogDescription>
              {t("calls.incoming.title")}
              {connection ? ` · ${t("calls.incoming.connection")}: ${connection}` : ""}
            </DialogDescription>
          </DialogHeader>
          {ringing.length > 1 && (
            <p className="text-center text-xs text-muted-foreground">+{ringing.length - 1}</p>
          )}
          <div className="flex justify-center gap-3">
            <Button variant="destructive" onClick={() => void reject(current.callId)} data-testid="reject-call">
              <PhoneOff />
              {t("calls.incoming.reject")}
            </Button>
            <Button onClick={() => void accept(current.callId)} disabled={busy} data-testid="accept-call">
              <Phone />
              {t("calls.incoming.accept")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
};

export default IncomingCallModal;
