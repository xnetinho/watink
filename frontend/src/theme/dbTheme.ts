// Valor da setting `theme` do tenant -> paleta + modo do ThemeContext.
//
// `darkMode` ausente  = claro fixo (comportamento histórico de whaticket/whatsapp).
// `darkMode: true`    = escuro fixo.
// `follow: "system"`  = o modo claro/escuro segue o navegador de cada usuário.
export interface DbThemeEntry {
  appTheme: string;
  darkMode?: boolean;
  follow?: "system";
}

export const DB_THEME_MAP: Record<string, DbThemeEntry> = {
  whaticket: { appTheme: "google" },
  whatsapp: { appTheme: "whatsapp" },
  dark: { appTheme: "apple", darkMode: true },
  auto: { appTheme: "google", follow: "system" },
};

export const DEFAULT_DB_THEME: DbThemeEntry = DB_THEME_MAP.whaticket;
