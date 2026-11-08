import React, { useContext, useState } from "react";
import { Phone, PhoneIncoming, PhoneMissed, PhoneOff, PhoneOutgoing, Video } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useCalls } from "@/context/Calls/CallsContext";
import { WhatsAppsContext } from "@/context/WhatsApp/WhatsAppsContext";
import { PLACE_BLOCK_KEY, canCallBack, placeBlock } from "@/lib/calls/canPlace";
import { StatusChip } from "@/components/ui/status-chip";
import { cn } from "@/lib/utils";
import api from "@/services/api";
import toastError from "@/errors/toastError";
import { t } from "@/lib/calls/t";
import { callerPlatformKey } from "@/lib/calls/platform";
import {
  callDurationLabel,
  callTitleKey,
  callTone,
  parseCallData,
  recordingView,
  type CallMessageData,
} from "@/lib/calls/callMessage";

const Icon: React.FC<{ d: CallMessageData }> = ({ d }) => {
  const cls = "h-5 w-5 shrink-0";
  if (d.status === "missed") return <PhoneMissed className={cls} />;
  if (d.status === "rejected" || d.status === "failed" || d.status === "interrupted") return <PhoneOff className={cls} />;
  if (d.media === "video") return <Video className={cls} />;
  return d.direction === "outgoing" ? <PhoneOutgoing className={cls} /> : <PhoneIncoming className={cls} />;
};

/**
 * Mensagem de sistema de uma chamada no histórico do ticket: direção, resultado,
 * quem atendeu, duração e, quando houver, o player da gravação. A URL do áudio só é
 * pedida ao clicar (é assinada, expira e a escuta fica registrada em auditoria).
 */
const CallMessage: React.FC<{ dataJson: unknown; body?: string; ticketId?: number; connectionId?: number }> = ({ dataJson, body, ticketId, connectionId }) => {
  const d = parseCallData(dataJson);
  const { canPlace, place, active, canSendVideo } = useCalls();
  const { whatsApps } = useContext(WhatsAppsContext);
  const [url, setUrl] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  if (!d) return <span className="text-sm">{body}</span>;

  const duration = callDurationLabel(d);
  // O aparelho do chamador só existe em chamada recebida; é declarado pelo WhatsApp (sem "~": não é estimativa).
  const platformKey = d.direction === "incoming" ? callerPlatformKey(d.platform) : null;
  const rec = recordingView(d);
  const reason = d.endReason ? t(`calls.endReason.${d.endReason}`) : "";

  // "Ligar de volta": na chamada recebida que ninguém atendeu, com as mesmas regras de bloqueio do botão de telefone. Volta em vídeo
  // se era videochamada e o navegador consegue enviar câmera; senão, em voz.
  const callBack = canPlace && ticketId !== undefined && canCallBack(d);
  const inCall = !!active && active.phase !== "ended";
  const block = callBack ? placeBlock({}, whatsApps.find((w) => w.id === connectionId), inCall) : null;
  const backVideo = d.media === "video" && canSendVideo;

  const listen = async () => {
    setLoading(true);
    try {
      const { data } = await api.get<{ url: string }>(`/calls/${encodeURIComponent(d.callId)}/recording`);
      setUrl(data.url);
    } catch (err) {
      toastError(err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-w-[220px] flex-col gap-2 py-1" data-testid="call-message" data-status={d.status}>
      <div className="flex items-center gap-2">
        <span className={cn("text-primary", d.status === "missed" && "text-status-warning-text", callTone(d.status) === "error" && "text-status-error-text")}>
          <Icon d={d} />
        </span>
        <div className="min-w-0">
          <p className="text-sm font-semibold leading-tight">{t(callTitleKey(d))}</p>
          <p className="text-xs text-muted-foreground">
            {[duration && `${t("calls.message.duration")}: ${duration}`, d.handledByName && `${t("calls.message.handledBy")} ${d.handledByName}`, platformKey && t(platformKey)]
              .filter(Boolean)
              .join(" · ")}
          </p>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <StatusChip status={callTone(d.status)} size="sm" label={t(`calls.status.${d.status}`)} />
        {reason && d.status !== "ended" && <span className="text-xs text-muted-foreground">{reason}</span>}
      </div>

      {callBack && (
        <Button
          type="button"
          size="sm"
          variant="outline"
          className="self-start"
          disabled={block !== null}
          title={block ? t(PLACE_BLOCK_KEY[block]) : undefined}
          onClick={() => void place(ticketId, backVideo)}
          data-testid="call-back"
          data-block={block ?? ""}
          data-video={backVideo ? "true" : "false"}
        >
          {backVideo ? <Video /> : <Phone />}
          {t("calls.message.callBack")}
        </Button>
      )}

      {rec === "ready" &&
        (url ? (
          <audio controls src={url} className="h-9 w-full" data-testid="call-recording-player" />
        ) : (
          <button
            type="button"
            onClick={() => void listen()}
            disabled={loading}
            className="self-start text-xs font-medium text-primary underline-offset-2 hover:underline disabled:opacity-50"
            data-testid="call-recording-listen"
          >
            {t("calls.message.listen")}
          </button>
        ))}
      {rec === "failed" && <p className="text-xs text-status-error-text" data-testid="call-recording-failed">{t("calls.message.recordingFailed")}</p>}
      {rec === "deleted" && <p className="text-xs text-muted-foreground" data-testid="call-recording-deleted">{t("calls.message.recordingDeleted")}</p>}
    </div>
  );
};

export default CallMessage;
