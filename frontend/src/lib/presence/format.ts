export interface Presence {
  online: boolean;
  /** segundos desde a época; ausente quando o contato escondeu a última vez visto */
  lastSeen?: number;
}

const hhmm = (d: Date) => d.toLocaleTimeString("pt-BR", { hour: "2-digit", minute: "2-digit" });

const startOfDay = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();

/** Texto do cabeçalho, como no WhatsApp Web: "online", "visto por último hoje às 14:05"... ou nada. */
export function presenceLabel(p: Presence | undefined, now: Date = new Date()): string {
  if (!p) return "";
  if (p.online) return "online";
  if (!p.lastSeen) return "";
  const seen = new Date(p.lastSeen * 1000);
  const days = Math.round((startOfDay(now) - startOfDay(seen)) / 86_400_000);
  if (days <= 0) return `visto por último hoje às ${hhmm(seen)}`;
  if (days === 1) return `visto por último ontem às ${hhmm(seen)}`;
  return `visto por último em ${seen.toLocaleDateString("pt-BR")} às ${hhmm(seen)}`;
}
