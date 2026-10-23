export type RecordingMode = "off" | "optional" | "auto";

export interface RecordingConfig {
  mode: RecordingMode;
  ackBy: number | null;
  ackAt: string | null;
  available: boolean;
}

export const RECORDING_MODES: RecordingMode[] = ["off", "optional", "auto"];

/** Valor desconhecido vindo do servidor nunca liga a gravação: vira "off". */
export function normalizeMode(v: unknown): RecordingMode {
  return v === "optional" || v === "auto" ? v : "off";
}

/**
 * Sair de "off" para outro modo exige o aceite do termo (o servidor também exige).
 * Entre modos já ligados, ou voltando para "off", não exige de novo.
 */
export function needsAck(current: RecordingMode, next: RecordingMode): boolean {
  return current === "off" && next !== "off";
}

export interface SaveState {
  /** Salvar fica habilitado? */
  canSave: boolean;
  /** Mostrar o termo e o checkbox de aceite? */
  showAck: boolean;
}

/** Decide o que a tela oferece a partir do modo gravado, do escolhido e do aceite marcado. */
export function saveState(current: RecordingMode, selected: RecordingMode, acked: boolean, available: boolean): SaveState {
  const showAck = needsAck(current, selected);
  if (selected === current) return { canSave: false, showAck };
  if (selected !== "off" && !available) return { canSave: false, showAck };
  if (showAck && !acked) return { canSave: false, showAck };
  return { canSave: true, showAck };
}

/** Corpo do PUT /calls/recording-config: o aceite só vai quando é exigido e foi dado. */
export function buildSaveBody(current: RecordingMode, selected: RecordingMode, acked: boolean): { mode: RecordingMode; ack?: boolean } {
  return needsAck(current, selected) && acked ? { mode: selected, ack: true } : { mode: selected };
}
