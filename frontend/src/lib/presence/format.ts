export type Translate = (key: string, opts?: Record<string, string | number>) => string;

export interface Presence {
  online: boolean;
  /** segundos desde a época; ausente quando o contato escondeu a última vez visto */
  lastSeen?: number;
}

const hhmm = (d: Date, locale: string) => d.toLocaleTimeString(locale, { hour: "2-digit", minute: "2-digit" });

const startOfDay = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();

/**
 * Texto do cabeçalho, como no WhatsApp Web ("online", "visto por último hoje às 14:05"...) ou vazio. Os textos vêm dos
 * arquivos de idioma (chat.presence.*) pelo tradutor recebido; a data e a hora seguem o idioma ativo.
 */
export function presenceLabel(p: Presence | undefined, t: Translate, locale: string, now: Date = new Date()): string {
  if (!p) return "";
  if (p.online) return t("chat.presence.online");
  if (!p.lastSeen) return "";
  const seen = new Date(p.lastSeen * 1000);
  const days = Math.round((startOfDay(now) - startOfDay(seen)) / 86_400_000);
  const time = hhmm(seen, locale);
  if (days <= 0) return t("chat.presence.today", { time });
  if (days === 1) return t("chat.presence.yesterday", { time });
  return t("chat.presence.date", { date: seen.toLocaleDateString(locale), time });
}
