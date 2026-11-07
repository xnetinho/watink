import React from "react";
import { i18n } from "../../../translate/i18n";
import { senderDeviceFromId } from "../../../lib/senderDevice";
import { Message } from "../types";

const tr = (key: string): string => String(i18n.t(key) ?? "");

/** Aparelho estimado do contato, só em mensagem recebida do WhatsApp e quando o id segue um padrão conhecido. */
export const senderDeviceOf = (m: Message, channel?: string) =>
  m.fromMe || m.isDeleted || m.mediaType === "call" || (channel && channel !== "whatsapp") ? null : senderDeviceFromId(m.id);

/** Rodapé do balão: "~Android". O "~" avisa que é uma estimativa (o WhatsApp não informa o aparelho). */
const SenderDeviceBadge: React.FC<{ message: Message; channel?: string }> = ({ message, channel }) => {
  const device = senderDeviceOf(message, channel);
  if (!device) return null;
  return (
    <span
      className="absolute bottom-0 left-1.5 text-[11px] text-[hsl(var(--text-muted))]"
      title={tr("chat.senderDevice.estimated")}
      data-testid="sender-device"
      data-device={device}
    >
      ~{tr(`chat.senderDevice.${device}`)}
    </span>
  );
};

export default SenderDeviceBadge;
