import React from "react";
import { BellOff, Phone, PhoneOff, Video } from "lucide-react";
import { Avatar } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { t } from "@/lib/calls/t";
import { displayName } from "@/lib/calls/format";
import { ringRemaining, ringSecondsLeft } from "@/lib/calls/ringCard";
import { useCalls } from "@/context/Calls/CallsContext";
import type { ActiveCall } from "@/context/Calls/types";

interface Props {
  call: ActiveCall;
  /** Relógio atual (ms): vem do pai para todos os cartões andarem juntos e os testes controlarem o tempo. */
  now: number;
  /** Nome da conexão que recebeu a chamada. */
  connection?: string;
  /** Faz o cartão encher a largura do pai (rodapé da lista) em vez de largura fixa (flutuante). */
  docked?: boolean;
}

/**
 * Cartão de uma chamada recebida: um convite, não uma interrupção. Não cobre a tela nem prende o foco; o operador segue
 * atendendo e decide quando puder. Ignorar silencia o toque e deixa o cartão até a chamada acabar (o chamador continua ouvindo
 * tocar e outro atendente pode atender); só Recusar derruba a chamada para todos.
 */
const IncomingCallCard: React.FC<Props> = ({ call, now, connection, docked }) => {
  const { accept, reject, ignore, active, videoSink } = useCalls();
  const name = displayName(call.contact, t("calls.unknownContact"));
  const busy = !!active && active.phase !== "ended";
  const isVideo = call.media === "video";
  const left = ringRemaining(call.ringingSince, now);
  const secs = ringSecondsLeft(call.ringingSince, now);

  return (
    <div
      role="alert"
      data-testid="incoming-call-card"
      data-media={call.media}
      data-ignored={call.ignored ? "true" : "false"}
      className={cn(
        "overflow-hidden rounded-lg border border-border bg-card text-card-foreground shadow-lg",
        docked ? "w-full" : "w-[290px]",
      )}
    >
      <div className="flex items-center gap-2.5 px-3 pt-3">
        <Avatar size="md" src={call.contact.profilePicUrl} name={name} />
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-semibold leading-tight">{name}</p>
          <p className="flex items-center gap-1 truncate text-xs text-muted-foreground">
            {isVideo && <Video className="h-3 w-3 shrink-0" aria-hidden="true" />}
            {isVideo ? t("calls.incoming.videoTitle") : t("calls.incoming.title")}
          </p>
          {connection && (
            <p className="truncate text-[11px] text-muted-foreground">
              {t("calls.incoming.connection")}: {connection}
            </p>
          )}
        </div>
        <span className="shrink-0 text-xs tabular-nums text-muted-foreground" data-testid="ring-seconds">
          {call.ringingSince === undefined ? "" : `${secs} s`}
        </span>
      </div>

      <div className="mx-3 mt-2 h-1 overflow-hidden rounded-full bg-muted" aria-hidden="true">
        <div className="h-full bg-primary transition-[width] duration-1000 ease-linear" style={{ width: `${left * 100}%` }} data-testid="ring-bar" />
      </div>

      {isVideo && !videoSink.supported && (
        <p className="mx-3 mt-2 text-[11px] text-muted-foreground" data-testid="incoming-video-unsupported">
          {t("calls.incoming.videoUnsupported")}
        </p>
      )}
      {busy && (
        <p className="mx-3 mt-2 text-[11px] text-muted-foreground" data-testid="incoming-busy">
          {t("calls.incoming.busy")}
        </p>
      )}

      <div className="flex items-center gap-1.5 px-3 pb-3 pt-2.5">
        <Button size="sm" className="flex-1" onClick={() => void accept(call.callId)} disabled={busy} data-testid="accept-call">
          <Phone />
          {t("calls.incoming.accept")}
        </Button>
        <Button size="sm" variant="destructive" className="flex-1" onClick={() => void reject(call.callId)} data-testid="reject-call">
          <PhoneOff />
          {t("calls.incoming.reject")}
        </Button>
        <Button
          size="sm"
          variant="ghost"
          onClick={() => ignore(call.callId)}
          disabled={call.ignored}
          title={t("calls.incoming.ignoreHint")}
          aria-label={t("calls.incoming.ignore")}
          data-testid="ignore-call"
        >
          <BellOff />
          {call.ignored ? t("calls.incoming.silenced") : t("calls.incoming.ignore")}
        </Button>
      </div>
    </div>
  );
};

export default IncomingCallCard;
