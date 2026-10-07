import React from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import CallMessage from "../CallMessage";
import "./helpers";

const get = vi.fn();
vi.mock("@/services/api", () => ({ default: { get: (...a: unknown[]) => get(...a) } }));
const toastError = vi.fn();
vi.mock("@/errors/toastError", () => ({ default: (e: unknown) => toastError(e) }));

const data = (over: Record<string, unknown> = {}) => ({
  callId: "C1",
  direction: "incoming",
  status: "ended",
  durationSec: 75,
  handledByName: "Ana",
  ...over,
});

beforeEach(() => {
  get.mockReset();
  toastError.mockReset();
});

describe("CallMessage", () => {
  it("chamada recebida e atendida: título, duração, operador e situação", () => {
    render(<CallMessage dataJson={data()} />);
    const m = screen.getByTestId("call-message");
    expect(m).toHaveAttribute("data-status", "ended");
    expect(m).toHaveTextContent("Chamada de voz recebida");
    expect(m).toHaveTextContent("Duração: 01:15");
    expect(m).toHaveTextContent("Atendida por Ana");
    expect(m).toHaveTextContent("Atendida");
  });

  it("chamada realizada", () => {
    render(<CallMessage dataJson={data({ direction: "outgoing" })} />);
    expect(screen.getByTestId("call-message")).toHaveTextContent("Chamada de voz realizada");
  });

  it("perdida: sem duração nem operador, com o motivo", () => {
    render(<CallMessage dataJson={data({ status: "missed", durationSec: 0, handledByName: undefined, endReason: "no_operator" })} />);
    const m = screen.getByTestId("call-message");
    expect(m).toHaveTextContent("Chamada de voz perdida");
    expect(m).toHaveTextContent("Perdida");
    expect(m).toHaveTextContent("Sem operador disponível");
    expect(m).not.toHaveTextContent("Duração");
    expect(m).not.toHaveTextContent("Atendida por");
  });

  it("recusada e interrompida têm título próprio", () => {
    const { rerender } = render(<CallMessage dataJson={data({ status: "rejected" })} />);
    expect(screen.getByTestId("call-message")).toHaveTextContent("Chamada de voz recusada");
    rerender(<CallMessage dataJson={data({ status: "interrupted", endReason: "interrupted" })} />);
    expect(screen.getByTestId("call-message")).toHaveTextContent("Chamada de voz interrompida");
  });

  it("dataJson inválido cai no texto do corpo, sem quebrar", () => {
    render(<CallMessage dataJson="lixo" body="Chamada de voz perdida" />);
    expect(screen.getByText("Chamada de voz perdida")).toBeInTheDocument();
    expect(screen.queryByTestId("call-message")).not.toBeInTheDocument();
  });

  it("sem gravação não há player nem botão", () => {
    render(<CallMessage dataJson={data()} />);
    expect(screen.queryByTestId("call-recording-listen")).not.toBeInTheDocument();
    expect(screen.queryByTestId("call-recording-player")).not.toBeInTheDocument();
  });

  it("com gravação pronta oferece 'Ouvir' e NÃO pede a URL antes do clique", () => {
    render(<CallMessage dataJson={data({ recordingStatus: "ready" })} />);
    expect(screen.getByTestId("call-recording-listen")).toBeInTheDocument();
    expect(get).not.toHaveBeenCalled();
  });

  it("clicar em ouvir busca a URL assinada e mostra o player", async () => {
    get.mockResolvedValue({ data: { url: "https://s3.local/x.mp3?sig=1" } });
    render(<CallMessage dataJson={data({ recordingStatus: "ready" })} />);
    fireEvent.click(screen.getByTestId("call-recording-listen"));
    await waitFor(() => expect(screen.getByTestId("call-recording-player")).toBeInTheDocument());
    expect(get).toHaveBeenCalledWith("/calls/C1/recording");
    expect(screen.getByTestId("call-recording-player")).toHaveAttribute("src", "https://s3.local/x.mp3?sig=1");
    expect(screen.queryByTestId("call-recording-listen")).not.toBeInTheDocument();
  });

  it("falha ao buscar a URL avisa o usuário e mantém o botão para tentar de novo", async () => {
    get.mockRejectedValue(new Error("403"));
    render(<CallMessage dataJson={data({ recordingStatus: "ready" })} />);
    fireEvent.click(screen.getByTestId("call-recording-listen"));
    await waitFor(() => expect(toastError).toHaveBeenCalled());
    expect(screen.getByTestId("call-recording-listen")).toBeInTheDocument();
    expect(screen.queryByTestId("call-recording-player")).not.toBeInTheDocument();
  });

  it("gravação que falhou e gravação excluída têm aviso próprio e nenhum player", () => {
    const { rerender } = render(<CallMessage dataJson={data({ recordingStatus: "failed" })} />);
    expect(screen.getByTestId("call-recording-failed")).toHaveTextContent("A gravação falhou");
    rerender(<CallMessage dataJson={data({ recordingStatus: "deleted" })} />);
    expect(screen.getByTestId("call-recording-deleted")).toHaveTextContent("Gravação excluída");
    expect(screen.queryByTestId("call-recording-listen")).not.toBeInTheDocument();
  });

  it("videochamada mostra o título e o ícone de vídeo", () => {
    render(<CallMessage dataJson={{ callId: "V1", direction: "incoming", media: "video", status: "ended", durationSec: 30 }} />);
    expect(screen.getByTestId("call-message")).toHaveTextContent("Videochamada recebida");
    expect(screen.getByTestId("call-message").querySelector("svg.lucide-video")).not.toBeNull();
  });

  it("chamada de voz não tem o ícone de vídeo e mantém o título", () => {
    render(<CallMessage dataJson={{ callId: "V2", direction: "incoming", media: "audio", status: "ended", durationSec: 30 }} />);
    expect(screen.getByTestId("call-message")).toHaveTextContent("Chamada de voz recebida");
    expect(screen.getByTestId("call-message").querySelector("svg.lucide-video")).toBeNull();
  });

  it("videochamada perdida usa o ícone de perdida, não o de vídeo", () => {
    render(<CallMessage dataJson={{ callId: "V3", direction: "incoming", media: "video", status: "missed" }} />);
    expect(screen.getByTestId("call-message")).toHaveTextContent("Videochamada perdida");
    expect(screen.getByTestId("call-message").querySelector("svg.lucide-video")).toBeNull();
  });
});

