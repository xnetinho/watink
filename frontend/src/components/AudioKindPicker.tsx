import React from "react";
import { i18n } from "../translate/i18n";

export type AudioKind = "voice" | "file";

const tr = (key: string): string => String(i18n.t(key) ?? "");

interface Props {
  value: AudioKind;
  onChange: (kind: AudioKind) => void;
  /** Prefixo único dos ids/nomes quando há mais de um seletor na tela. */
  name?: string;
  className?: string;
  /** Esconde a explicação de cada opção (espaços apertados, como o cartão da prévia do anexo). */
  compact?: boolean;
}

/**
 * Como um ÁUDIO sai no WhatsApp: mensagem de voz (bolha com microfone e onda) ou arquivo de áudio (anexo). Serve ao
 * anexar e ao encaminhar. A gravação feita pelo Watink não passa por aqui: sai sempre como mensagem de voz.
 */
export const AudioKindPicker: React.FC<Props> = ({ value, onChange, name = "audio-kind", className, compact = false }) => (
  <fieldset className={className ?? "flex flex-col gap-1"} aria-label={tr("messagesInput.audioKind.label")} data-testid="audio-kind-picker">
    <legend className="mb-0.5 text-xs font-medium text-[hsl(var(--text-muted))]">{tr("messagesInput.audioKind.label")}</legend>
    {(["voice", "file"] as AudioKind[]).map((k) => (
      <label key={k} className="flex cursor-pointer items-start gap-2 text-sm">
        <input type="radio" name={name} className="mt-1" checked={value === k} onChange={() => onChange(k)} />
        <span className="flex flex-col">
          <span className="font-medium">{tr(`messagesInput.audioKind.${k}`)}</span>
          {!compact && <span className="text-xs text-[hsl(var(--text-muted))]">{tr(`messagesInput.audioKind.${k}Hint`)}</span>}
        </span>
      </label>
    ))}
  </fieldset>
);

export default AudioKindPicker;
