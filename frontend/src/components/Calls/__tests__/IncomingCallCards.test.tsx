import React from "react";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, render, screen, within } from "@testing-library/react";
import IncomingCallCards from "../IncomingCallCards";
import { CallCardDockProvider, CallCardDockSlot } from "../CallCardDock";
import { baseCall, makeCtx, withCalls } from "./helpers";

const play = vi.fn(() => Promise.resolve());
const pause = vi.fn();
beforeAll(() => {
  window.HTMLMediaElement.prototype.play = play as never;
  window.HTMLMediaElement.prototype.pause = pause;
});
beforeEach(() => {
  play.mockClear();
  pause.mockClear();
});

const ring = (over = {}) => baseCall({ ringingSince: Date.now(), ...over });
const renderCards = (ctx: ReturnType<typeof makeCtx>, node: React.ReactNode = <IncomingCallCards connectionName={(id) => (id === 1 ? "Vendas" : undefined)} />) =>
  render(withCalls(ctx, <CallCardDockProvider>{node}</CallCardDockProvider>));

describe("cartões de chamada recebida", () => {
  it("sem chamada tocando não mostra nada", () => {
    renderCards(makeCtx());
    expect(screen.queryByTestId("incoming-call-card")).not.toBeInTheDocument();
  });

  it("mostra nome, tipo, conexão, tempo e as três ações, sem janela modal que prenda o foco", () => {
    renderCards(makeCtx({ ringing: [ring()] }));
    const card = screen.getByTestId("incoming-call-card");
    expect(card).toHaveTextContent("Maria Souza");
    expect(card).toHaveTextContent("Chamada de voz recebida");
    expect(card).toHaveTextContent("Vendas");
    expect(screen.getByTestId("accept-call")).toBeEnabled();
    expect(screen.getByTestId("reject-call")).toBeEnabled();
    expect(screen.getByTestId("ignore-call")).toBeEnabled();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("Atender, Recusar e Ignorar chamam a ação certa com o id da chamada", () => {
    const ctx = makeCtx({ ringing: [ring({ callId: "ABC" })] });
    renderCards(ctx);
    fireEvent.click(screen.getByTestId("accept-call"));
    fireEvent.click(screen.getByTestId("reject-call"));
    fireEvent.click(screen.getByTestId("ignore-call"));
    expect(ctx.accept).toHaveBeenCalledWith("ABC");
    expect(ctx.reject).toHaveBeenCalledWith("ABC");
    expect(ctx.ignore).toHaveBeenCalledWith("ABC");
    expect(ctx.reject).toHaveBeenCalledTimes(1);
  });

  it("Ignorar NÃO recusa: só silencia (a recusa derrubaria a chamada para o contato e para os outros atendentes)", () => {
    const ctx = makeCtx({ ringing: [ring()] });
    renderCards(ctx);
    fireEvent.click(screen.getByTestId("ignore-call"));
    expect(ctx.reject).not.toHaveBeenCalled();
  });

  it("depois de ignorar o cartão continua na tela, marcado como silenciado, e ainda dá para atender", () => {
    renderCards(makeCtx({ ringing: [ring({ ignored: true })] }));
    const card = screen.getByTestId("incoming-call-card");
    expect(card).toHaveAttribute("data-ignored", "true");
    expect(screen.getByTestId("ignore-call")).toBeDisabled();
    expect(screen.getByTestId("ignore-call")).toHaveTextContent("Toque silenciado");
    expect(screen.getByTestId("accept-call")).toBeEnabled();
  });

  it("quem já está em chamada não atende outra (avisa), mas pode recusar e ignorar", () => {
    renderCards(makeCtx({ ringing: [ring()], active: baseCall({ callId: "OUTRA", phase: "active" }) }));
    expect(screen.getByTestId("accept-call")).toBeDisabled();
    expect(screen.getByTestId("incoming-busy")).toBeInTheDocument();
    expect(screen.getByTestId("reject-call")).toBeEnabled();
    expect(screen.getByTestId("ignore-call")).toBeEnabled();
  });

  it("chamada anterior já encerrada (painel ainda aberto) não bloqueia o Atender nem silencia o toque", () => {
    renderCards(makeCtx({ ringing: [ring({ media: "video" })], active: baseCall({ callId: "ANTIGA", phase: "ended" }) }));
    expect(screen.getByTestId("accept-call")).toBeEnabled();
    expect(screen.queryByTestId("incoming-busy")).not.toBeInTheDocument();
    expect(play).toHaveBeenCalled();
  });

  it("videochamada mostra o tipo e, sem suporte do navegador, avisa que atende só com voz", () => {
    renderCards(makeCtx({ ringing: [ring({ media: "video" })] }));
    expect(screen.getByTestId("incoming-call-card")).toHaveAttribute("data-media", "video");
    expect(screen.getByTestId("incoming-call-card")).toHaveTextContent("Videochamada");
    expect(screen.getByTestId("incoming-video-unsupported")).toHaveTextContent(/só com áudio/i);
  });

  it("chamada de voz não mostra aviso de vídeo", () => {
    renderCards(makeCtx({ ringing: [ring()] }));
    expect(screen.queryByTestId("incoming-video-unsupported")).not.toBeInTheDocument();
  });
});

describe("aparelho do chamador", () => {
  it.each([
    ["android", "Android"],
    ["iphone", "iPhone"],
    ["web", "Web"],
    ["smba", "WhatsApp Business (Android)"],
  ])("chamada de %s mostra %s, sem o til de estimativa (é declarado pelo WhatsApp)", (platform, label) => {
    renderCards(makeCtx({ ringing: [ring({ platform })] }));
    const el = screen.getByTestId("caller-platform");
    expect(el).toHaveTextContent(label);
    expect(el.textContent).not.toContain("~");
  });

  it("junto da conexão, separados por ponto", () => {
    renderCards(makeCtx({ ringing: [ring({ platform: "iphone" })] }));
    expect(screen.getByTestId("incoming-call-card")).toHaveTextContent("Conexão: Vendas · iPhone");
  });

  it("sem plataforma, ou com valor desconhecido, não mostra nada", () => {
    renderCards(makeCtx({ ringing: [ring()] }));
    expect(screen.queryByTestId("caller-platform")).not.toBeInTheDocument();
    renderCards(makeCtx({ ringing: [ring({ callId: "X", platform: "<b>x</b>" })] }));
    expect(screen.queryByText("<b>x</b>")).not.toBeInTheDocument();
  });
});

describe("tempo do toque", () => {
  // O relógio do cartão anda de 1 em 1 s a partir da montagem; os segundos restantes arredondam para cima. Por isso o teste confere
  // faixas e a tendência (cai, chega a zero e não passa), não o milissegundo exato.
  it("os segundos andam com o relógio e param em zero", () => {
    vi.useFakeTimers();
    try {
      renderCards(makeCtx({ ringing: [ring({ ringingSince: Date.now() })] }));
      const secs = () => Number((screen.getByTestId("ring-seconds").textContent ?? "").replace(/\D/g, ""));
      expect(secs()).toBe(45);
      act(() => {
        vi.advanceTimersByTime(30_000);
      });
      expect(secs()).toBeGreaterThanOrEqual(14);
      expect(secs()).toBeLessThanOrEqual(16);
      act(() => {
        vi.advanceTimersByTime(60_000);
      });
      expect(secs()).toBe(0);
    } finally {
      vi.useRealTimers();
    }
  });
});

describe("som do toque", () => {
  it("toca em loop enquanto houver oferta, e para quando acaba", () => {
    const { rerender } = renderCards(makeCtx({ ringing: [ring()] }));
    expect(play).toHaveBeenCalled();
    expect((document.querySelector("audio") as HTMLAudioElement).loop).toBe(true);
    pause.mockClear();
    rerender(withCalls(makeCtx({ ringing: [] }), <CallCardDockProvider><IncomingCallCards /></CallCardDockProvider>));
    expect(pause).toHaveBeenCalled();
  });

  it("Ignorar silencia o som, mas o cartão fica", () => {
    const { rerender } = renderCards(makeCtx({ ringing: [ring()] }));
    expect(play).toHaveBeenCalled();
    pause.mockClear();
    rerender(withCalls(makeCtx({ ringing: [ring({ ignored: true })] }), <CallCardDockProvider><IncomingCallCards /></CallCardDockProvider>));
    expect(pause).toHaveBeenCalled();
    expect(screen.getByTestId("incoming-call-card")).toBeInTheDocument();
  });

  it("com duas ofertas, ignorar só uma mantém o som da outra", () => {
    play.mockClear();
    renderCards(makeCtx({ ringing: [ring({ callId: "A", ignored: true }), ring({ callId: "B" })] }));
    expect(play).toHaveBeenCalled();
  });

  it("quem está em chamada não ouve o toque de outra", () => {
    renderCards(makeCtx({ ringing: [ring()], active: baseCall({ callId: "OUTRA", phase: "active" }) }));
    expect(play).not.toHaveBeenCalled();
  });
});

describe("várias chamadas e posição", () => {
  it("mostra até 2 cartões, as mais antigas, e conta o resto", () => {
    const t0 = Date.now();
    renderCards(
      makeCtx({
        ringing: [
          ring({ callId: "C", ringingSince: t0 + 20, contact: { name: "Carla" } }),
          ring({ callId: "A", ringingSince: t0, contact: { name: "Ana" } }),
          ring({ callId: "B", ringingSince: t0 + 10, contact: { name: "Bia" } }),
        ],
      }),
    );
    expect(screen.getAllByTestId("incoming-call-card")).toHaveLength(2);
    expect(screen.getByText("Ana")).toBeInTheDocument();
    expect(screen.getByText("Bia")).toBeInTheDocument();
    expect(screen.queryByText("Carla")).not.toBeInTheDocument();
    expect(screen.getByTestId("more-calls")).toHaveTextContent("+1");
  });

  it("fora da lista de tickets flutua no canto inferior esquerdo", () => {
    renderCards(makeCtx({ ringing: [ring()] }));
    const box = screen.getByTestId("call-cards-floating");
    expect(box.className).toContain("bottom-4");
    expect(box.className).toContain("left-4");
  });

  it("com a lista de tickets na tela encaixa no rodapé dela, sem flutuar", () => {
    renderCards(
      makeCtx({ ringing: [ring()] }),
      <>
        <CallCardDockSlot />
        <IncomingCallCards />
      </>,
    );
    const dock = screen.getByTestId("call-card-dock");
    expect(within(dock).getByTestId("incoming-call-card")).toBeInTheDocument();
    expect(screen.queryByTestId("call-cards-floating")).not.toBeInTheDocument();
  });
});
