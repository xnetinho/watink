import React from "react";
import { describe, expect, it } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import ActiveCallPanel, { phaseLabel } from "../ActiveCallPanel";
import { baseCall, makeCtx, quality, withCalls } from "./helpers";

const rec = (ui: React.ReactElement) => ui;

describe("phaseLabel", () => {
  it("cada estado tem o seu texto", () => {
    expect(phaseLabel(baseCall({ phase: "calling" }), 0)).toBe("Chamando…");
    expect(phaseLabel(baseCall({ phase: "connecting" }), 0)).toBe("Conectando…");
    expect(phaseLabel(baseCall({ phase: "ended" }), 0)).toBe("Chamada encerrada");
  });
  it("em chamada mostra o cronômetro desde a conexão real", () => {
    const c = baseCall({ phase: "active", connectedAt: 1_000_000 });
    expect(phaseLabel(c, 1_075_000)).toBe("01:15");
    expect(phaseLabel(baseCall({ phase: "active", connectedAt: null }), 5_000_000)).toBe("00:00");
  });
});

describe("ActiveCallPanel", () => {
  it("sem chamada ativa não renderiza nada", () => {
    const { container } = render(withCalls(makeCtx(), <ActiveCallPanel />));
    expect(container.querySelector('[data-testid="active-call"]')).toBeNull();
  });

  it("'Chamando…' enquanto o contato não atende, com encerrar disponível e silenciar bloqueado", () => {
    render(withCalls(makeCtx({ active: baseCall({ phase: "calling", direction: "outgoing" }) }), <ActiveCallPanel />));
    expect(screen.getByTestId("call-status")).toHaveTextContent("Chamando…");
    expect(screen.getByTestId("active-call")).toHaveAttribute("data-phase", "calling");
    expect(screen.getByTestId("end-call")).toBeEnabled();
    expect(screen.getByTestId("mute-call")).toBeDisabled();
  });

  it("conectando: texto próprio", () => {
    render(withCalls(makeCtx({ active: baseCall({ phase: "connecting" }) }), <ActiveCallPanel />));
    expect(screen.getByTestId("call-status")).toHaveTextContent("Conectando…");
    expect(screen.getByTestId("mute-call")).toBeEnabled();
  });

  it("ativa: mostra nome, cronômetro e permite silenciar e encerrar", () => {
    const ctx = makeCtx({ active: baseCall({ phase: "active", connectedAt: Date.now() - 65_000 }) });
    render(withCalls(ctx, <ActiveCallPanel />));
    expect(screen.getByText("Maria Souza")).toBeInTheDocument();
    expect(screen.getByTestId("call-status").textContent).toMatch(/^01:0[5-6]$/);
    fireEvent.click(screen.getByTestId("mute-call"));
    expect(ctx.setMuted).toHaveBeenCalledWith(true);
    fireEvent.click(screen.getByTestId("end-call"));
    expect(ctx.end).toHaveBeenCalled();
  });

  it("silenciado: o estado fica visível e o botão oferece reativar", () => {
    const ctx = makeCtx({ active: baseCall({ phase: "active", muted: true, connectedAt: Date.now() }) });
    render(withCalls(ctx, <ActiveCallPanel />));
    expect(screen.getByTestId("muted-label")).toHaveTextContent("Microfone silenciado");
    expect(screen.getByTestId("mute-call")).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByTestId("mute-call")).toHaveAttribute("aria-label", "Ativar microfone");
    fireEvent.click(screen.getByTestId("mute-call"));
    expect(ctx.setMuted).toHaveBeenCalledWith(false);
  });

  it("encerrada: mostra o fim, esconde os controles e permite fechar", () => {
    const ctx = makeCtx({ active: baseCall({ phase: "ended", endReason: "user_ended" }) });
    render(withCalls(ctx, <ActiveCallPanel />));
    expect(screen.getByTestId("call-status")).toHaveTextContent("Chamada encerrada");
    expect(screen.queryByTestId("end-call")).not.toBeInTheDocument();
    expect(screen.queryByTestId("mute-call")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Fechar" }));
    expect(ctx.dismiss).toHaveBeenCalled();
  });

  it("microfone negado mostra o motivo e orienta a liberar a permissão", () => {
    render(withCalls(makeCtx({ active: baseCall({ phase: "connecting", failure: "mic_denied" }) }), <ActiveCallPanel />));
    expect(screen.getByTestId("call-failure")).toHaveTextContent(/negou o acesso ao microfone/);
  });

  it.each([
    ["mic_unavailable", /Nenhum microfone/],
    ["unsupported", /não suporta/],
    ["socket", /conexão de áudio caiu/],
  ])("falha %s tem texto próprio", (failure, re) => {
    render(withCalls(makeCtx({ active: baseCall({ phase: "active", failure, connectedAt: Date.now() }) }), <ActiveCallPanel />));
    expect(screen.getByTestId("call-failure")).toHaveTextContent(re);
  });

  it("indicador 'Gravando' aparece só quando está gravando", () => {
    const { rerender } = render(withCalls(makeCtx({ active: baseCall({ phase: "active", connectedAt: Date.now() }) }), <ActiveCallPanel />));
    expect(screen.queryByTestId("recording-indicator")).not.toBeInTheDocument();
    rerender(withCalls(makeCtx({ active: baseCall({ phase: "active", recording: true, connectedAt: Date.now() }) }), <ActiveCallPanel />));
    expect(screen.getByTestId("recording-indicator")).toHaveTextContent("Gravando");
  });

  it("botão de gravar: só no modo opcional, com armazenamento e chamada ativa", () => {
    const active = baseCall({ phase: "active", connectedAt: Date.now() });
    const mk = (props: { recordingAvailable?: boolean; recordingMode?: string }, a = active) =>
      withCalls(makeCtx({ active: a }), <ActiveCallPanel {...props} />);

    const { rerender } = render(mk({ recordingAvailable: true, recordingMode: "optional" }));
    expect(screen.getByTestId("record-call")).toBeInTheDocument();

    rerender(mk({ recordingAvailable: true, recordingMode: "off" }));
    expect(screen.queryByTestId("record-call")).not.toBeInTheDocument();
    rerender(mk({ recordingAvailable: true, recordingMode: "auto" }));
    expect(screen.queryByTestId("record-call")).not.toBeInTheDocument();
    rerender(mk({ recordingAvailable: false, recordingMode: "optional" }));
    expect(screen.queryByTestId("record-call")).not.toBeInTheDocument();
    rerender(mk({ recordingAvailable: true, recordingMode: "optional" }, baseCall({ phase: "connecting" })));
    expect(screen.queryByTestId("record-call")).not.toBeInTheDocument();
  });

  it("gravar chama start; com gravação em curso o mesmo botão para", () => {
    const ctx = makeCtx({ active: baseCall({ phase: "active", connectedAt: Date.now() }) });
    const { rerender } = render(withCalls(ctx, <ActiveCallPanel recordingAvailable recordingMode="optional" />));
    fireEvent.click(screen.getByTestId("record-call"));
    expect(ctx.startRecording).toHaveBeenCalled();

    const ctx2 = makeCtx({ active: baseCall({ phase: "active", recording: true, connectedAt: Date.now() }) });
    rerender(withCalls(ctx2, <ActiveCallPanel recordingAvailable recordingMode="optional" />));
    fireEvent.click(screen.getByTestId("record-call"));
    expect(ctx2.stopRecording).toHaveBeenCalled();
  });

  it("o painel de qualidade só aparece com a chamada ativa", () => {
    const q = quality();
    const { rerender } = render(withCalls(makeCtx({ active: baseCall({ phase: "connecting", quality: q }) }), <ActiveCallPanel />));
    expect(screen.queryByTestId("call-quality")).not.toBeInTheDocument();
    rerender(withCalls(makeCtx({ active: baseCall({ phase: "active", connectedAt: Date.now(), quality: q }) }), <ActiveCallPanel />));
    expect(screen.getByTestId("call-quality")).toBeInTheDocument();
  });
});

void rec;
