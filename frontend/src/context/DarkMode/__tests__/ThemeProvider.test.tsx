import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, act } from "@testing-library/react";
import React from "react";
import { ThemeProvider, useThemeContext } from "..";

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

function setSystem(dark: boolean) {
  systemDark = dark;
  act(() => listeners.forEach((l) => l({ matches: dark })));
}

let ctx: ReturnType<typeof useThemeContext>;
function Probe() {
  ctx = useThemeContext();
  return null;
}
const mount = () =>
  render(
    <ThemeProvider>
      <Probe />
    </ThemeProvider>
  );
const isDark = () => document.documentElement.classList.contains("dark");

describe("ThemeProvider — modo automático", () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.className = "";
    systemDark = false;
    stubMatchMedia();
  });

  it("sem nada salvo, segue o navegador na primeira visita", () => {
    systemDark = true;
    mount();
    expect(isDark()).toBe(true);
  });

  it("applyDbTheme('auto') segue o navegador, mesmo com darkMode=false salvo antes", () => {
    localStorage.setItem("darkMode", "false");
    systemDark = true;
    mount();
    expect(isDark()).toBe(false);
    act(() => ctx.applyDbTheme("auto"));
    expect(isDark()).toBe(true);
  });

  it("em automático, reage à mudança do tema do sistema com a aba aberta", () => {
    mount();
    act(() => ctx.applyDbTheme("auto"));
    expect(isDark()).toBe(false);
    setSystem(true);
    expect(isDark()).toBe(true);
    setSystem(false);
    expect(isDark()).toBe(false);
  });

  it("automático persiste: no próximo carregamento já nasce no modo do sistema", () => {
    mount();
    act(() => ctx.applyDbTheme("auto"));
    document.documentElement.className = "";
    systemDark = true;
    mount();
    expect(isDark()).toBe(true);
  });

  it("temas fixos continuam fixos e ignoram o sistema", () => {
    systemDark = true;
    mount();
    act(() => ctx.applyDbTheme("whaticket"));
    expect(isDark()).toBe(false);
    setSystem(true);
    expect(isDark()).toBe(false);
    act(() => ctx.applyDbTheme("dark"));
    expect(isDark()).toBe(true);
    setSystem(false);
    expect(isDark()).toBe(true);
  });

  it("sair do automático para tema fixo para de seguir o sistema", () => {
    mount();
    act(() => ctx.applyDbTheme("auto"));
    act(() => ctx.applyDbTheme("whatsapp"));
    expect(isDark()).toBe(false);
    setSystem(true);
    expect(isDark()).toBe(false);
    expect(localStorage.getItem("themeFollowSystem")).toBe("false");
  });

  it("escolha manual (toggleTheme) encerra o automático", () => {
    mount();
    act(() => ctx.applyDbTheme("auto"));
    act(() => ctx.toggleTheme());
    expect(isDark()).toBe(true);
    setSystem(false);
    expect(isDark()).toBe(true);
  });

  it("valor desconhecido da setting cai no padrão claro (não quebra)", () => {
    mount();
    act(() => ctx.applyDbTheme("valor-inexistente"));
    expect(isDark()).toBe(false);
  });

  it("remove o listener ao sair do automático (sem vazamento)", () => {
    mount();
    act(() => ctx.applyDbTheme("auto"));
    expect(listeners.length).toBe(1);
    act(() => ctx.applyDbTheme("whaticket"));
    expect(listeners.length).toBe(0);
  });
});
