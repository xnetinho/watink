/**
 * ThemeContext — orchestrator for CSS design tokens
 *
 * - applyThemeTokens() sets CSS variables on :root consumed by var(--token) in components
 * - System preference detection on init (prefers-color-scheme: dark)
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

interface BrandOverrides {
  primary?: string;
  secondary?: string;
  [key: string]: string | undefined;
}

interface ThemeContextValue {
  darkMode: boolean;
  toggleTheme: () => void;
  setDarkMode: (v: boolean) => void;
  appTheme: string;
  setAppTheme: (v: string) => void;
  setBrand: (b: BrandOverrides) => void;
  /** Aplica o valor da setting `theme` do tenant (paleta + modo, inclusive "auto"). */
  applyDbTheme: (dbValue: string) => void;
}

const ThemeContext = createContext<ThemeContextValue | undefined>(undefined);

const SYSTEM_DARK_QUERY = "(prefers-color-scheme: dark)";

const systemPrefersDark = (): boolean =>
  window.matchMedia?.(SYSTEM_DARK_QUERY)?.matches ?? false;

// "auto" = o modo segue o navegador. Persistido à parte de `darkMode` para que
// o primeiro paint (antes da setting do tenant chegar) já respeite o modo.
const getInitialFollowSystem = (): boolean =>
  localStorage.getItem("themeFollowSystem") === "true";

const getInitialDarkMode = (): boolean => {
  if (getInitialFollowSystem()) return systemPrefersDark();
  const stored = localStorage.getItem("darkMode");
  if (stored !== null) return stored === "true";
  return systemPrefersDark();
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
  const [darkMode, setDarkMode] = useState<boolean>(getInitialDarkMode);
  const [followSystem, setFollowSystem] = useState<boolean>(getInitialFollowSystem);
  const [appTheme, setAppThemeState] = useState<string>(getInitialAppTheme);
  const [brand, setBrand] = useState<BrandOverrides>({});

  const toggleTheme = useCallback(() => {
    setFollowSystem(false);
    localStorage.setItem("themeFollowSystem", "false");
    setDarkMode((prev) => {
      const next = !prev;
      localStorage.setItem("darkMode", String(next));
      return next;
    });
  }, []);

  // Escolha explícita de claro/escuro encerra o modo automático.
  const setDarkModeValue = useCallback((v: boolean) => {
    setFollowSystem(false);
    localStorage.setItem("themeFollowSystem", "false");
    setDarkMode(v);
    localStorage.setItem("darkMode", String(v));
  }, []);

  const setAppTheme = useCallback((v: string) => {
    setAppThemeState(v);
    localStorage.setItem("appTheme", v);
  }, []);

  const applyDbTheme = useCallback(
    (dbValue: string) => {
      const mapped = DB_THEME_MAP[dbValue] ?? DEFAULT_DB_THEME;
      setAppTheme(mapped.appTheme);
      if (mapped.follow === "system") {
        setFollowSystem(true);
        localStorage.setItem("themeFollowSystem", "true");
        setDarkMode(systemPrefersDark());
      } else {
        setDarkModeValue(mapped.darkMode ?? false);
      }
    },
    [setAppTheme, setDarkModeValue]
  );

  // Modo automático: reage à mudança do tema do sistema com a aba aberta.
  useEffect(() => {
    if (!followSystem) return;
    const mql = window.matchMedia?.(SYSTEM_DARK_QUERY);
    if (!mql) return;
    setDarkMode(mql.matches);
    const onChange = (e: MediaQueryListEvent) => setDarkMode(e.matches);
    mql.addEventListener("change", onChange);
    return () => mql.removeEventListener("change", onChange);
  }, [followSystem]);

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
      setDarkMode: setDarkModeValue,
      appTheme,
      setAppTheme,
      setBrand,
      applyDbTheme,
    }),
    [darkMode, toggleTheme, setDarkModeValue, appTheme, setAppTheme, setBrand, applyDbTheme]
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
