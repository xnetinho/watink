/** Plataformas que o WhatsApp declara no <offer> de uma chamada (o servidor só repassa estas). */
const KNOWN = ["android", "iphone", "web", "smba", "smbi", "ipad", "darwin", "uwp"] as const;
export type CallerPlatform = (typeof KNOWN)[number];

/** A chave de tradução do aparelho do chamador, ou null se desconhecido ou ausente (nunca mostra texto que veio do aparelho). */
export function callerPlatformKey(platform: string | undefined): string | null {
  if (!platform) return null;
  const p = platform.toLowerCase();
  return (KNOWN as readonly string[]).includes(p) ? `calls.platform.${p}` : null;
}
