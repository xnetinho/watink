// Valor da setting `theme` do tenant -> PALETA do ThemeContext.
//
// O tenant escolhe só a paleta (identidade visual). Claro/escuro é preferência
// pessoal de cada usuário (ver context/DarkMode). Valores legados continuam
// aceitos: "dark" (antigo "Escuro Noturno") vira a paleta padrão.
export interface DbThemeEntry {
  appTheme: string;
}

export const DB_THEME_MAP: Record<string, DbThemeEntry> = {
  whaticket: { appTheme: "google" },
  whatsapp: { appTheme: "whatsapp" },
  dark: { appTheme: "google" },
  auto: { appTheme: "google" },
};

export const DEFAULT_DB_THEME: DbThemeEntry = DB_THEME_MAP.whaticket;
