import React from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import CallMessage from "../CallMessage";
import { WhatsAppsContext } from "@/context/WhatsApp/WhatsAppsContext";
import { baseCall, makeCtx, withCalls } from "./helpers";
import type { CallsContextValue } from "@/context/Calls/CallsContext";

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


// "Ligar de volta" no registro de chamada (pedido do dono): só na chamada RECEBIDA que ninguém atendeu, com as mesmas regras de bloqueio do botão
// de telefone, e volta em vídeo quando a chamada era em vídeo e o navegador sabe enviar câmera.
describe("CallMessage: Ligar de volta", () => {
  const wa = (over: Record<string, unknown> = {}) => ({ id: 1, name: "Vendas", status: "CONNECTED", proxyMode: "none", ...over });
  const renderBack = (d: Record<string, unknown>, ctx: Partial<CallsContextValue> = {}, whatsApps: ReturnType<typeof wa>[] = [wa()], ticketId: number | undefined = 7) => {
    const c = makeCtx(ctx);
    render(
      withCalls(
        c,
        <WhatsAppsContext.Provider value={{ whatsApps, loading: false, reloadWhatsApps: async () => undefined }}>
          <CallMessage dataJson={data(d)} ticketId={ticketId} connectionId={1} />
        </WhatsAppsContext.Provider>,
      ),
    );
    return c;
  };

  it.each(["missed", "rejected", "failed", "interrupted"])("chamada recebida %s oferece o botão e liga com o id do ticket", (status) => {
    const c = renderBack({ status, handledByName: undefined });
    fireEvent.click(screen.getByTestId("call-back"));
    expect(c.place).toHaveBeenCalledWith(7, false);
  });

  it("chamada atendida, ou feita por nós, não oferece", () => {
    renderBack({ status: "ended" });
    expect(screen.queryByTestId("call-back")).not.toBeInTheDocument();
  });

  it("chamada nossa que ninguém atendeu também não (o chamador somos nós)", () => {
    renderBack({ status: "missed", direction: "outgoing" });
    expect(screen.queryByTestId("call-back")).not.toBeInTheDocument();
  });

  it("sem calls:place, ou sem o ticket, não aparece", () => {
    renderBack({ status: "missed" }, { canPlace: false });
    expect(screen.queryByTestId("call-back")).not.toBeInTheDocument();
  });

  it("videochamada perdida volta em vídeo; sem suporte a câmera no navegador volta em voz", () => {
    const c = renderBack({ status: "missed", media: "video" });
    expect(screen.getByTestId("call-back")).toHaveAttribute("data-video", "true");
    fireEvent.click(screen.getByTestId("call-back"));
    expect(c.place).toHaveBeenCalledWith(7, true);
  });

  it("videochamada perdida, mas navegador sem câmera, volta em voz", () => {
    const c = renderBack({ status: "missed", media: "video" }, { canSendVideo: false });
    fireEvent.click(screen.getByTestId("call-back"));
    expect(c.place).toHaveBeenCalledWith(7, false);
  });

  it("conexão desconectada, com proxy, ou operador em outra chamada: desabilitado, com o motivo", () => {
    renderBack({ status: "missed" }, {}, [wa({ status: "DISCONNECTED" })]);
    expect(screen.getByTestId("call-back")).toBeDisabled();
    expect(screen.getByTestId("call-back")).toHaveAttribute("data-block", "disconnected");
  });

  it("com proxy fica desabilitado", () => {
    renderBack({ status: "missed" }, {}, [wa({ proxyMode: "single" })]);
    expect(screen.getByTestId("call-back")).toHaveAttribute("data-block", "proxy");
  });

  it("operador já em chamada fica desabilitado; chamada anterior encerrada não conta como ocupado", () => {
    renderBack({ status: "missed" }, { active: baseCall({ callId: "OUTRA", phase: "active" }) });
    expect(screen.getByTestId("call-back")).toHaveAttribute("data-block", "busy");
  });

  it("chamada anterior já encerrada não bloqueia o Ligar de volta", () => {
    renderBack({ status: "missed" }, { active: baseCall({ callId: "ANTIGA", phase: "ended" }) });
    expect(screen.getByTestId("call-back")).toBeEnabled();
  });
});

describe("CallMessage: aparelho do chamador", () => {
  it("chamada recebida mostra o aparelho declarado, sem til", () => {
    render(<CallMessage dataJson={data({ platform: "android" })} />);
    const m = screen.getByTestId("call-message");
    expect(m).toHaveTextContent("Android");
    expect(m.textContent).not.toContain("~");
  });
  it("chamada que fizemos não mostra aparelho (o chamador somos nós)", () => {
    render(<CallMessage dataJson={data({ direction: "outgoing", platform: "android" })} />);
    expect(screen.getByTestId("call-message")).not.toHaveTextContent("Android");
  });
  it("sem plataforma, ou desconhecida, não mostra", () => {
    render(<CallMessage dataJson={data({ platform: "windows" })} />);
    expect(screen.getByTestId("call-message")).not.toHaveTextContent("windows");
  });
});
