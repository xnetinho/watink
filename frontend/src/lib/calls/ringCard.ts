/** Quanto o WhatsApp deixa uma chamada recebida tocar (igual ao ringTimeout do engine, 45 s). */
export const RING_TOTAL_MS = 45_000;

/** Quantos cartões aparecem de uma vez; o resto vira "+N". */
export const MAX_CARDS = 2;

/** Fração do toque que ainda resta, de 0 a 1. Sem instante de início (oferta antiga) não inventa um prazo: devolve 1. */
export function ringRemaining(since: number | undefined, now: number, total: number = RING_TOTAL_MS): number {
  if (since === undefined) return 1;
  return Math.min(1, Math.max(0, 1 - (now - since) / total));
}

/** Segundos que faltam para o toque acabar, arredondados para cima. */
export function ringSecondsLeft(since: number | undefined, now: number, total: number = RING_TOTAL_MS): number {
  return Math.ceil(ringRemaining(since, now, total) * (total / 1000));
}

/** Os cartões visíveis (os mais antigos primeiro, que são os que vencem antes) e quantos ficaram de fora. */
export function visibleCards<T extends { ringingSince?: number }>(ringing: T[], max: number = MAX_CARDS): { shown: T[]; hidden: number } {
  const ordered = [...ringing].sort((a, b) => (a.ringingSince ?? 0) - (b.ringingSince ?? 0));
  return { shown: ordered.slice(0, max), hidden: Math.max(0, ordered.length - max) };
}

/** Ainda há toque para ouvir? Só quando existe oferta que o operador não silenciou e ele não está em outra chamada. */
export function shouldRing(ringingIds: string[], ignored: ReadonlySet<string>, inCall: boolean): boolean {
  if (inCall) return false;
  return ringingIds.some((id) => !ignored.has(id));
}
