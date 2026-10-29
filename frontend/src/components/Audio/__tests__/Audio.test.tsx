import React from "react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, act } from "@testing-library/react";
import Audio from "../index";
import { __clearWaveformCache } from "../../../lib/audio/waveform";

// ── Helpers ────────────────────────────────────────────────────────────────────

// HTMLMediaElement não tem implementação real no jsdom — stub mínimo.
function stubAudio() {
  const playMock = vi.fn().mockResolvedValue(undefined);
  const pauseMock = vi.fn();
  Object.defineProperty(window.HTMLMediaElement.prototype, "play", {
    configurable: true,
    writable: true,
    value: playMock,
  });
  Object.defineProperty(window.HTMLMediaElement.prototype, "pause", {
    configurable: true,
    writable: true,
    value: pauseMock,
  });
  return { playMock, pauseMock };
}

// ── Tests ──────────────────────────────────────────────────────────────────────

describe("Audio", () => {
  beforeEach(() => {
    localStorage.clear();
    stubAudio();
  });

  it("renderiza botão de play e indicadores de tempo", () => {
    render(<Audio url="/media/test.ogg" />);
    expect(screen.getByRole("button", { name: /reproduzir/i })).toBeInTheDocument();
    // como o WhatsApp Web: um tempo só (a duração; vira o tempo corrente ao tocar)
    expect(screen.getByTestId("audio-time").textContent).toBe("0:00");
  });

  it("usa mimetype prop como source type", () => {
    const { container } = render(<Audio url="/media/test.mp3" mimetype="audio/mpeg" />);
    const sources = container.querySelectorAll("source");
    expect(sources[0].getAttribute("type")).toBe("audio/mpeg");
  });

  it("fallback para audio/ogg quando mimetype nao passado", () => {
    const { container } = render(<Audio url="/media/test.ogg" />);
    const sources = container.querySelectorAll("source");
    expect(sources[0].getAttribute("type")).toBe("audio/ogg");
  });

  it("exibe botao de velocidade com valor inicial 1x", () => {
    render(<Audio url="/media/test.ogg" />);
    expect(screen.getByRole("button", { name: /velocidade 1x/i })).toBeInTheDocument();
  });

  it("cicla velocidade ao clicar no botao: 1x→1.5x→2x→0.5x→1x", () => {
    render(<Audio url="/media/test.ogg" />);
    const btn = screen.getByRole("button", { name: /velocidade/i });
    expect(btn.textContent).toBe("1x");
    fireEvent.click(btn);
    expect(btn.textContent).toBe("1.5x");
    fireEvent.click(btn);
    expect(btn.textContent).toBe("2x");
    fireEvent.click(btn);
    expect(btn.textContent).toBe("0.5x");
    fireEvent.click(btn);
    expect(btn.textContent).toBe("1x");
  });

  it("persiste velocidade no localStorage ao ciclar", () => {
    render(<Audio url="/media/test.ogg" />);
    const btn = screen.getByRole("button", { name: /velocidade/i });
    fireEvent.click(btn); // 1.5x
    expect(localStorage.getItem("audioMessageRate")).toBe("1.5");
  });

  it("restaura velocidade salva no localStorage ao montar", () => {
    localStorage.setItem("audioMessageRate", "2");
    render(<Audio url="/media/test.ogg" />);
    const btn = screen.getByRole("button", { name: /velocidade/i });
    expect(btn.textContent).toBe("2x");
  });

  it("chama play ao clicar no botao de play", async () => {
    const { playMock } = stubAudio();
    render(<Audio url="/media/test.ogg" />);
    const playBtn = screen.getByRole("button", { name: /reproduzir/i });
    fireEvent.click(playBtn);
    expect(playMock).toHaveBeenCalled();
  });
});

// ── formatTime (função pura — testada via comportamento DOM) ──────────────────

describe("formatTime helper via Audio", () => {
  it("exibe 0:00 quando duracao e zero", () => {
    render(<Audio url="/x.ogg" />);
    const zeros = screen.getAllByText("0:00");
    expect(zeros.length).toBeGreaterThanOrEqual(1);
  });
});


// ── Layout estilo WhatsApp Web ────────────────────────────────────────────────

describe("Audio: avatar, onda e barra de avanço", () => {
  beforeEach(() => {
    localStorage.clear();
    stubAudio();
    __clearWaveformCache();
  });

  it("mostra a foto do remetente quando existe", () => {
    const { container } = render(<Audio url="/a.ogg" senderName="Ana Lima" senderPicUrl="https://x/y.jpg" />);
    const img = container.querySelector('[data-testid="audio-avatar"] img');
    expect(img?.getAttribute("src")).toBe("https://x/y.jpg");
  });

  it("sem foto, mostra as iniciais do nome", () => {
    render(<Audio url="/a.ogg" senderName="Ana Lima" />);
    expect(screen.getByTestId("audio-avatar").textContent).toContain("AL");
  });

  it("sem foto e sem nome, mostra '?' em vez de quebrar", () => {
    render(<Audio url="/a.ogg" />);
    expect(screen.getByTestId("audio-avatar").textContent).toContain("?");
  });

  it("microfone só na mensagem de voz, não no arquivo de áudio", () => {
    const { rerender } = render(<Audio url="/a.ogg" senderName="Ana" voice />);
    expect(screen.getByTestId("audio-voice-badge")).toBeInTheDocument();
    expect(screen.getByTestId("audio-player").getAttribute("data-voice")).toBe("true");
    rerender(<Audio url="/a.ogg" senderName="Ana" />);
    expect(screen.queryByTestId("audio-voice-badge")).toBeNull();
    expect(screen.getByTestId("audio-player").getAttribute("data-voice")).toBe("false");
  });

  it("desenha 64 barras de onda (de repouso até a real carregar)", () => {
    render(<Audio url="/a.ogg" />);
    expect(screen.getByTestId("audio-waveform").querySelectorAll("[data-played]").length).toBe(64);
  });

  it("a onda é um controle deslizante acessível, com tempo no valuetext", () => {
    render(<Audio url="/a.ogg" />);
    const slider = screen.getByRole("slider");
    expect(slider.getAttribute("aria-label")).toBe("Posição do áudio");
    expect(slider.getAttribute("aria-valuetext")).toBe("0:00 / 0:00");
  });

  it("clicar na onda leva o áudio àquela posição e pinta as barras já ouvidas", () => {
    const { container } = render(<Audio url="/a.ogg" />);
    const el = container.querySelector("audio") as HTMLAudioElement;
    Object.defineProperty(el, "duration", { configurable: true, value: 100 });
    fireEvent(el, new Event("loadedmetadata"));
    const wave = screen.getByTestId("audio-waveform");
    wave.getBoundingClientRect = () => ({ left: 0, width: 200, top: 0, right: 200, bottom: 10, height: 10, x: 0, y: 0, toJSON: () => ({}) });
    fireEvent.pointerDown(wave, { clientX: 100, pointerId: 1 }); // metade
    expect(el.currentTime).toBeCloseTo(50, 0);
    expect(wave.querySelectorAll('[data-played="true"]').length).toBe(32);
    expect(screen.getByTestId("audio-time").textContent).toBe("0:50");
  });

  it("clique fora da onda não passa dos limites (0 e duração)", () => {
    const { container } = render(<Audio url="/a.ogg" />);
    const el = container.querySelector("audio") as HTMLAudioElement;
    Object.defineProperty(el, "duration", { configurable: true, value: 60 });
    fireEvent(el, new Event("loadedmetadata"));
    const wave = screen.getByTestId("audio-waveform");
    wave.getBoundingClientRect = () => ({ left: 10, width: 100, top: 0, right: 110, bottom: 10, height: 10, x: 10, y: 0, toJSON: () => ({}) });
    fireEvent.pointerDown(wave, { clientX: 500, pointerId: 1 });
    expect(el.currentTime).toBe(60);
    fireEvent.pointerDown(wave, { clientX: -50, pointerId: 1 });
    expect(el.currentTime).toBe(0);
  });

  it("setas do teclado avançam 5 s e Home/End vão às pontas", () => {
    const { container } = render(<Audio url="/a.ogg" />);
    const el = container.querySelector("audio") as HTMLAudioElement;
    Object.defineProperty(el, "duration", { configurable: true, value: 30 });
    fireEvent(el, new Event("loadedmetadata"));
    const slider = screen.getByRole("slider");
    fireEvent.keyDown(slider, { key: "ArrowRight" });
    expect(el.currentTime).toBe(5);
    fireEvent.keyDown(slider, { key: "ArrowRight" });
    expect(el.currentTime).toBe(10);
    fireEvent.keyDown(slider, { key: "ArrowLeft" });
    expect(el.currentTime).toBe(5);
    fireEvent.keyDown(slider, { key: "End" });
    expect(el.currentTime).toBe(30);
    fireEvent.keyDown(slider, { key: "Home" });
    expect(el.currentTime).toBe(0);
  });

  it("sem duração conhecida, clicar na onda não faz nada", () => {
    const { container } = render(<Audio url="/a.ogg" />);
    const el = container.querySelector("audio") as HTMLAudioElement;
    el.currentTime = 0;
    const wave = screen.getByTestId("audio-waveform");
    wave.getBoundingClientRect = () => ({ left: 0, width: 200, top: 0, right: 200, bottom: 10, height: 10, x: 0, y: 0, toJSON: () => ({}) });
    fireEvent.pointerDown(wave, { clientX: 100, pointerId: 1 });
    expect(el.currentTime).toBe(0);
  });

  it("enquanto toca, a posição segue o relógio do áudio a cada quadro (bolinha fluida, não 4x/s do timeupdate)", () => {
    const frames: FrameRequestCallback[] = [];
    vi.spyOn(window, "requestAnimationFrame").mockImplementation((cb) => (frames.push(cb), frames.length));
    vi.spyOn(window, "cancelAnimationFrame").mockImplementation(() => undefined);
    const { container } = render(<Audio url="/a.ogg" />);
    const el = container.querySelector("audio") as HTMLAudioElement;
    Object.defineProperty(el, "duration", { configurable: true, value: 100 });
    fireEvent(el, new Event("loadedmetadata"));
    fireEvent(el, new Event("play"));
    for (const t of [10, 10.016, 10.033]) {
      el.currentTime = t;
      act(() => frames.splice(0).forEach((cb) => cb(0)));
    }
    expect(screen.getByTestId("audio-knob").style.left).toBe("10.033%");
  });
});
