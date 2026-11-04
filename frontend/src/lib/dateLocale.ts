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

/** Tag BCP 47 do idioma ativo ("pt-BR", "en-US", "es-ES"), para Intl e toLocale*String. */
export const localeTag = (): string => {
  const lang = (i18n.language || "pt").toLowerCase();
  if (lang.startsWith("en")) return "en-US";
  if (lang.startsWith("es")) return "es-ES";
  return "pt-BR";
};
