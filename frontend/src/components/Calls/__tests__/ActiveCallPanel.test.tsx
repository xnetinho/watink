import React from "react";
import { describe, expect, it, vi } from "vitest";
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
    expect(screen.getByTestId("call-status")).toHaveTextContent("O contato desligou");
    expect(screen.queryByTestId("end-call")).not.toBeInTheDocument();
    expect(screen.queryByTestId("mute-call")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Fechar" }));
    expect(ctx.dismiss).toHaveBeenCalled();
  });

  it.each([
    ["user_ended", false, "O contato desligou"],
    ["user_ended", true, "Você encerrou a chamada"],
    ["declined", false, "O contato recusou a chamada"],
    ["timeout", false, "O contato não atendeu"],
    ["busy", false, "O contato está ocupado"],
    ["accepted_elsewhere", false, "Atendida em outro aparelho"],
    ["motivo_desconhecido", false, "Chamada encerrada"],
  ])("encerrada por %s (eu encerrei: %s) diz quem encerrou", (endReason, endedByMe, texto) => {
    render(withCalls(makeCtx({ active: baseCall({ phase: "ended", endReason, endedByMe }) }), <ActiveCallPanel />));
    expect(screen.getByTestId("call-status")).toHaveTextContent(texto);
  });

  it("microfone negado mostra o motivo e orienta a liberar a permissão", () => {
    render(withCalls(makeCtx({ active: baseCall({ phase: "connecting", failure: "mic_denied" }) }), <ActiveCallPanel />));
    expect(screen.getByTestId("call-failure")).toHaveTextContent(/negou o acesso ao microfone/);
  });

  it.each([
    ["mic_unavailable", /Nenhum microfone/],
    ["unsupported", /não suporta/],
    ["socket", /conexão de áudio caiu/],
    ["audio_unavailable", /canal de áudio do servidor está indisponível/],
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

describe("ActiveCallPanel — aviso de risco permanente", () => {
  it.each(["calling", "connecting", "active", "ended"] as const)("aparece com a chamada em '%s'", (phase) => {
    render(withCalls(makeCtx({ active: baseCall({ phase, connectedAt: phase === "active" ? Date.now() : null }) }), <ActiveCallPanel />));
    expect(screen.getByTestId("risk-notice")).toHaveTextContent("risco de bloqueio do número");
    expect(screen.getByTestId("risk-notice")).toHaveTextContent("Esse risco é da empresa");
  });
});

describe("ActiveCallPanel — câmera do operador", () => {
  it("numa chamada de voz não há botão de câmera", () => {
    render(withCalls(makeCtx({ active: baseCall({ phase: "active", media: "audio", connectedAt: 1 }) }), <ActiveCallPanel />));
    expect(screen.queryByTestId("toggle-camera")).toBeNull();
    expect(screen.queryByTestId("camera-unsupported")).toBeNull();
  });

  it("na videochamada ativa o botão liga a câmera e, ligada, desliga", () => {
    const setCamera = vi.fn(async () => undefined);
    const { rerender } = render(
      withCalls(makeCtx({ setCamera, active: baseCall({ phase: "active", media: "video", connectedAt: 1 }) }), <ActiveCallPanel />),
    );
    const btn = screen.getByTestId("toggle-camera");
    expect(btn).toBeEnabled();
    expect(btn).toHaveAttribute("aria-label", "Ligar câmera");
    fireEvent.click(btn);
    expect(setCamera).toHaveBeenLastCalledWith(true);

    rerender(withCalls(makeCtx({ setCamera, active: baseCall({ phase: "active", media: "video", connectedAt: 1, camera: true }) }), <ActiveCallPanel />));
    const on = screen.getByTestId("toggle-camera");
    expect(on).toHaveAttribute("aria-label", "Desligar câmera");
    expect(on).toHaveAttribute("aria-pressed", "true");
    fireEvent.click(on);
    expect(setCamera).toHaveBeenLastCalledWith(false);
  });

  it("enquanto a chamada ainda toca o botão fica desabilitado", () => {
    render(withCalls(makeCtx({ active: baseCall({ phase: "calling", media: "video" }) }), <ActiveCallPanel />));
    expect(screen.getByTestId("toggle-camera")).toBeDisabled();
  });

  it("navegador sem WebCodecs de envio explica o motivo e não deixa ligar", () => {
    render(withCalls(makeCtx({ canSendVideo: false, active: baseCall({ phase: "active", media: "video", connectedAt: 1 }) }), <ActiveCallPanel />));
    expect(screen.getByTestId("camera-unsupported")).toHaveTextContent(/Chrome, o Edge ou o Brave/);
    expect(screen.getByTestId("toggle-camera")).toBeDisabled();
  });

  it("câmera bloqueada mostra a causa e como resolver", () => {
    render(withCalls(makeCtx({ active: baseCall({ phase: "active", media: "video", connectedAt: 1, cameraFailure: "denied" }) }), <ActiveCallPanel />));
    expect(screen.getByTestId("camera-failure")).toHaveTextContent(/bloqueada pelo navegador/);
  });

  it("câmera ausente e erro do codificador têm mensagens próprias", () => {
    const { rerender } = render(
      withCalls(makeCtx({ active: baseCall({ phase: "active", media: "video", connectedAt: 1, cameraFailure: "unavailable" }) }), <ActiveCallPanel />),
    );
    expect(screen.getByTestId("camera-failure")).toHaveTextContent(/não existe ou está em uso/);
    rerender(withCalls(makeCtx({ active: baseCall({ phase: "active", media: "video", connectedAt: 1, cameraFailure: "encoder" }) }), <ActiveCallPanel />));
    expect(screen.getByTestId("camera-failure")).toHaveTextContent(/codificar o vídeo/);
  });

  it("chamada encerrada não mostra controle de câmera", () => {
    render(withCalls(makeCtx({ active: baseCall({ phase: "ended", media: "video", endReason: "user_ended" }) }), <ActiveCallPanel />));
    expect(screen.queryByTestId("toggle-camera")).toBeNull();
  });
});
