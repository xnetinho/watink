/**
 * ThemeContext — orchestrator for CSS design tokens
 *
 * - applyThemeTokens() sets CSS variables on :root consumed by var(--token) in components
 * - Claro/escuro é PESSOAL: preferência do usuário (`light`|`dark`), guardada em
 *   Users.configs.theme e espelhada no localStorage (primeiro paint sem piscar).
 *   Sem preferência, segue o navegador (prefers-color-scheme) e reage a mudanças.
 * - A paleta (apple/google/whatsapp/saas) continua sendo do tenant (white-label)
 * - brand overrides enable white-labeling per tenant
 */

import React, {
  createContext,
  useState,
  useContext,
  useMemo,
  useEffect,
  useCallback,
  ReactNode,
} from "react";
import { applyThemeTokens } from "../../theme/loader";
import { DB_THEME_MAP, DEFAULT_DB_THEME } from "../../theme/dbTheme";
import { AuthContext } from "../Auth/AuthContext";
import api from "../../services/api";

interface BrandOverrides {
  primary?: string;
  secondary?: string;
  [key: string]: string | undefined;
}

type ThemePreference = "light" | "dark";

interface ThemeContextValue {
  darkMode: boolean;
  /** Alterna claro/escuro e grava como preferência pessoal do usuário. */
  toggleTheme: () => void;
  appTheme: string;
  setAppTheme: (v: string) => void;
  setBrand: (b: BrandOverrides) => void;
  /** Aplica a PALETA da setting `theme` do tenant (claro/escuro não é do tenant). */
  applyDbTheme: (dbValue: string) => void;
}

const ThemeContext = createContext<ThemeContextValue | undefined>(undefined);

const SYSTEM_DARK_QUERY = "(prefers-color-scheme: dark)";
const PREF_KEY = "themePreference";

const systemPrefersDark = (): boolean =>
  window.matchMedia?.(SYSTEM_DARK_QUERY)?.matches ?? false;

const asPreference = (v: unknown): ThemePreference | null =>
  v === "light" || v === "dark" ? v : null;

const readStoredPreference = (): ThemePreference | null =>
  asPreference(localStorage.getItem(PREF_KEY));

// `configs` chega como string JSON em /auth/refresh_token e /me, mas já como
// objeto em outros pontos (ex.: dashboard) — aceita os dois.
const preferenceFromUser = (user: { configs?: unknown } | undefined): ThemePreference | null => {
  const raw = user?.configs;
  if (!raw) return null;
  try {
    const cfg = typeof raw === "string" ? JSON.parse(raw) : raw;
    return asPreference((cfg as { theme?: unknown }).theme);
  } catch {
    return null;
  }
};

const VALID_THEMES = ["apple", "google", "whatsapp", "saas"];

const getInitialAppTheme = (): string => {
  const stored = localStorage.getItem("appTheme");
  // Migra temas legados (valores inválidos) para o padrão google
  if (!stored || !VALID_THEMES.includes(stored)) {
    localStorage.setItem("appTheme", "google");
    return "google";
  }
  return stored;
};

export const ThemeProvider: React.FC<{ children: ReactNode }> = ({
  children,
}) => {
  const { user } = useContext(AuthContext);
  const [preference, setPreference] = useState<ThemePreference | null>(readStoredPreference);
  const [systemDark, setSystemDark] = useState<boolean>(systemPrefersDark);
  const darkMode = preference ? preference === "dark" : systemDark;
  const [appTheme, setAppThemeState] = useState<string>(getInitialAppTheme);
  const [brand, setBrand] = useState<BrandOverrides>({});

  // Segue o navegador enquanto o usuário não escolheu.
  useEffect(() => {
    const mql = window.matchMedia?.(SYSTEM_DARK_QUERY);
    if (!mql) return;
    setSystemDark(mql.matches);
    const onChange = (e: MediaQueryListEvent) => setSystemDark(e.matches);
    mql.addEventListener("change", onChange);
    return () => mql.removeEventListener("change", onChange);
  }, []);

  // Preferência salva no servidor vale em qualquer dispositivo/navegador.
  const serverPreference = preferenceFromUser(user as { configs?: unknown } | undefined);
  useEffect(() => {
    if (!serverPreference) return;
    setPreference(serverPreference);
    localStorage.setItem(PREF_KEY, serverPreference);
  }, [serverPreference]);

  const toggleTheme = useCallback(() => {
    const next: ThemePreference = darkMode ? "light" : "dark";
    setPreference(next);
    localStorage.setItem(PREF_KEY, next);
    // Falha de rede não desfaz a troca local — vale neste navegador e é regravada no próximo clique.
    api.put("/me/theme", { theme: next }).catch(() => undefined);
  }, [darkMode]);

  const applyDbTheme = useCallback((dbValue: string) => {
    setAppThemeState((DB_THEME_MAP[dbValue] ?? DEFAULT_DB_THEME).appTheme);
  }, []);

  const setAppTheme = useCallback((v: string) => {
    setAppThemeState(v);
    localStorage.setItem("appTheme", v);
  }, []);

  useEffect(() => {
    applyThemeTokens({
      mode: darkMode ? "dark" : "light",
      appTheme: appTheme as "apple" | "google" | "whatsapp" | "saas",
      brand: brand as { primary: string; primaryHover: string; sidebarBg: string },
    });
  }, [darkMode, appTheme, brand]);

  const contextValue = useMemo<ThemeContextValue>(
    () => ({
      darkMode,
      toggleTheme,
      appTheme,
      setAppTheme,
      setBrand,
      applyDbTheme,
    }),
    [darkMode, toggleTheme, appTheme, setAppTheme, setBrand, applyDbTheme]
  );

  return (
    <ThemeContext.Provider value={contextValue}>
      {children}
    </ThemeContext.Provider>
  );
};

export default ThemeProvider;

export const useThemeContext = (): ThemeContextValue => {
  const ctx = useContext(ThemeContext);
  if (!ctx) throw new Error("useThemeContext must be used inside ThemeProvider");
  return ctx;
};
