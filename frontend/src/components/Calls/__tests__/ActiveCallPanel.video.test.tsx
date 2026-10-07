import React from "react";
import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import ActiveCallPanel from "../ActiveCallPanel";
import { baseCall, makeCtx, withCalls } from "./helpers";
import { VideoSink } from "@/lib/calls/videoSink";

const sinkOf = (supported: boolean) => {
  const s = new VideoSink(supported ? (class {} as never) : undefined, supported ? (class {} as never) : undefined);
  s.attach = vi.fn();
  return s;
};

describe("ActiveCallPanel — vídeo", () => {
  it("videochamada ativa mostra o vídeo do contato (canvas) e registra o canvas no sink", () => {
    const sink = sinkOf(true);
    render(withCalls(makeCtx({ active: baseCall({ media: "video", phase: "active", connectedAt: Date.now() }), videoSink: sink }), <ActiveCallPanel />));
    expect(screen.getByTestId("call-video")).toBeInTheDocument();
    expect(sink.attach).toHaveBeenCalledWith(expect.any(HTMLCanvasElement));
  });

  it("chamada de voz não tem área de vídeo", () => {
    render(withCalls(makeCtx({ active: baseCall({ media: "audio", phase: "active", connectedAt: Date.now() }), videoSink: sinkOf(true) }), <ActiveCallPanel />));
    expect(screen.queryByTestId("call-video")).not.toBeInTheDocument();
  });

  it("navegador sem WebCodecs: segue só com áudio e explica por que não há imagem", () => {
    render(withCalls(makeCtx({ active: baseCall({ media: "video", phase: "active", connectedAt: Date.now() }), videoSink: sinkOf(false) }), <ActiveCallPanel />));
    expect(screen.queryByTestId("call-video")).not.toBeInTheDocument();
    expect(screen.getByTestId("video-unsupported")).toHaveTextContent(/Chrome|Edge|Brave/);
  });

  it("conectando, a área de vídeo já existe (o primeiro quadro-chave chega logo depois)", () => {
    render(withCalls(makeCtx({ active: baseCall({ media: "video", phase: "connecting" }), videoSink: sinkOf(true) }), <ActiveCallPanel />));
    expect(screen.getByTestId("call-video")).toBeInTheDocument();
  });

  it("chamada encerrada não mantém o vídeo na tela", () => {
    render(withCalls(makeCtx({ active: baseCall({ media: "video", phase: "ended" }), videoSink: sinkOf(true) }), <ActiveCallPanel />));
    expect(screen.queryByTestId("call-video")).not.toBeInTheDocument();
  });
});
