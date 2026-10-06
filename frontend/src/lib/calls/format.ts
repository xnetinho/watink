/** "75" -> "01:15"; acima de 1 h mostra "1:02:03". Nunca negativo. */
export function formatDuration(totalSeconds: number): string {
  const s = Math.max(0, Math.floor(totalSeconds));
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  const mm = String(m).padStart(2, "0");
  const ss = String(sec).padStart(2, "0");
  return h > 0 ? `${h}:${mm}:${ss}` : `${mm}:${ss}`;
}

/** Segundos decorridos desde `connectedAt` (ms), ou 0 se a mídia ainda não conectou. */
export function elapsedSeconds(connectedAt: number | null, now: number): number {
  if (!connectedAt) return 0;
  return Math.max(0, Math.floor((now - connectedAt) / 1000));
}

/** Bytes/s -> "12,4 kbps". */
export function formatBitrate(bytesPerSec: number | undefined): string {
  if (bytesPerSec == null || !Number.isFinite(bytesPerSec)) return "—";
  return `${((bytesPerSec * 8) / 1000).toFixed(1).replace(".", ",")} kbps`;
}

/** Nível RMS (0..1) para a largura de uma barra, em %, com ganho para voz. */
export function levelPercent(level: number | undefined): number {
  if (!level || level <= 0) return 0;
  return Math.min(100, Math.round(Math.sqrt(level) * 140));
}

/** Nome a exibir: nome do contato, senão o número, senão o texto padrão. */
export function displayName(contact: { name?: string; number?: string }, fallback: string): string {
  return contact.name?.trim() || contact.number?.trim() || fallback;
}
