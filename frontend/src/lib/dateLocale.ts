import { enUS, es, ptBR } from "date-fns/locale";
import type { Locale } from "date-fns";
import { i18n } from "../translate/i18n";

/** Locale do date-fns conforme o idioma ativo da interface (datas relativas como "há 3 horas"). */
export const dateLocale = (): Locale => {
  const lang = (i18n.language || "pt").toLowerCase();
  if (lang.startsWith("en")) return enUS;
  if (lang.startsWith("es")) return es;
  return ptBR;
};
