import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, act, fireEvent, screen } from "@testing-library/react";
import React from "react";

const apiPut = vi.fn();
vi.mock("../../../services/api", () => ({ default: { put: (...a: unknown[]) => apiPut(...a) } }));

import { AuthContext } from "../../Auth/AuthContext";
import { ThemeProvider, useThemeContext } from "..";
import ThemeToggle from "../../../components/ThemeToggle";

type Listener = (e: { matches: boolean }) => void;
let systemDark = false;
let listeners: Listener[] = [];

function stubMatchMedia() {
  listeners = [];
  vi.stubGlobal("matchMedia", (q: string) => ({
    get matches() {
      return q.includes("dark") && systemDark;
    },
    media: q,
    addEventListener: (_: string, l: Listener) => listeners.push(l),
    removeEventListener: (_: string, l: Listener) => {
      listeners = listeners.filter((x) => x !== l);
    },
  }));
}
const setSystem = (dark: boolean) => {
  systemDark = dark;
  act(() => listeners.forEach((l) => l({ matches: dark })));
};

let ctx: ReturnType<typeof useThemeContext>;
function Probe() {
  ctx = useThemeContext();
  return null;
}
const mount = (user: Record<string, unknown> = {}, withToggle = false) =>
  render(
    <AuthContext.Provider value={{ user, loading: false, isAuth: true } as never}>
      <ThemeProvider>
        <Probe />
        {withToggle && <ThemeToggle />}
      </ThemeProvider>
    </AuthContext.Provider>
  );
const isDark = () => document.documentElement.classList.contains("dark");

describe("tema claro/escuro pessoal do usuário", () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.className = "";
    systemDark = false;
    apiPut.mockReset();
    apiPut.mockResolvedValue({ data: {} });
    stubMatchMedia();
  });

  it("sem escolha, segue o navegador e reage à mudança do sistema", () => {
    systemDark = true;
    mount();
    expect(isDark()).toBe(true);
    setSystem(false);
    expect(isDark()).toBe(false);
  });

  it("o ícone alterna claro <-> escuro a cada clique", () => {
    mount({}, true);
    expect(isDark()).toBe(false);
    fireEvent.click(screen.getByRole("button"));
    expect(isDark()).toBe(true);
    fireEvent.click(screen.getByRole("button"));
    expect(isDark()).toBe(false);
  });

  it("o clique grava a preferência no servidor e no navegador", () => {
    mount({}, true);
    fireEvent.click(screen.getByRole("button"));
    expect(apiPut).toHaveBeenCalledWith("/me/theme", { theme: "dark" });
    expect(localStorage.getItem("themePreference")).toBe("dark");
    fireEvent.click(screen.getByRole("button"));
    expect(apiPut).toHaveBeenLastCalledWith("/me/theme", { theme: "light" });
  });

  it("escolha manual vence o sistema, inclusive quando o sistema muda depois", () => {
    systemDark = true;
    mount({}, true);
    fireEvent.click(screen.getByRole("button"));
    expect(isDark()).toBe(false);
    setSystem(true);
    expect(isDark()).toBe(false);
  });

  it("login em outro dispositivo: a preferência do servidor (string JSON) é aplicada", () => {
    systemDark = false;
    mount({ configs: JSON.stringify({ theme: "dark" }) });
    expect(isDark()).toBe(true);
    expect(localStorage.getItem("themePreference")).toBe("dark");
  });

  it("aceita configs já como objeto", () => {
    mount({ configs: { theme: "dark" } });
    expect(isDark()).toBe(true);
  });

  it("a preferência salva no navegador vale já no primeiro render (sem piscar)", () => {
    localStorage.setItem("themePreference", "dark");
    mount();
    expect(isDark()).toBe(true);
  });

  it("configs inválido ou sem tema não quebra e cai no sistema", () => {
    systemDark = true;
    mount({ configs: "{não é json" });
    expect(isDark()).toBe(true);
    document.documentElement.className = "";
    localStorage.clear();
    mount({ configs: JSON.stringify({ dashboard: {} }) });
    expect(isDark()).toBe(true);
  });

  it("falha de rede ao salvar não desfaz a troca local", async () => {
    apiPut.mockRejectedValue(new Error("offline"));
    mount({}, true);
    fireEvent.click(screen.getByRole("button"));
    await act(async () => {});
    expect(isDark()).toBe(true);
  });

  it("a paleta do tenant não decide claro/escuro (legado 'dark' vira paleta padrão)", () => {
    mount();
    act(() => ctx.applyDbTheme("dark"));
    expect(ctx.appTheme).toBe("google");
    expect(isDark()).toBe(false);
    act(() => ctx.applyDbTheme("whatsapp"));
    expect(ctx.appTheme).toBe("whatsapp");
    expect(isDark()).toBe(false);
  });

  it("o rótulo do ícone descreve a ação", () => {
    mount({}, true);
    expect(screen.getByRole("button").getAttribute("aria-label")).toBe("Mudar para tema escuro");
    fireEvent.click(screen.getByRole("button"));
    expect(screen.getByRole("button").getAttribute("aria-label")).toBe("Mudar para tema claro");
  });
});
