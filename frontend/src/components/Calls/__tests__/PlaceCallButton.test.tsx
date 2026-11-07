import React from "react";
import { describe, expect, it } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import PlaceCallButton from "../PlaceCallButton";
import { WhatsAppsContext } from "@/context/WhatsApp/WhatsAppsContext";
import { baseCall, makeCtx, withCalls } from "./helpers";
import type { CallsContextValue } from "@/context/Calls/CallsContext";

const wa = (over: Record<string, unknown> = {}) => ({ id: 1, name: "Vendas", status: "CONNECTED", proxyMode: "none", ...over });

function renderBtn(
  ticket: { id: number; connectionId?: number; isGroup?: boolean; isCommunity?: boolean; isSubGroup?: boolean },
  whatsApps: ReturnType<typeof wa>[],
  ctx: Partial<CallsContextValue> = {},
) {
  const c = makeCtx(ctx);
  render(
    withCalls(
      c,
      <WhatsAppsContext.Provider value={{ whatsApps, loading: false, reloadWhatsApps: async () => undefined }}>
        <PlaceCallButton ticket={ticket} />
      </WhatsAppsContext.Provider>,
    ),
  );
  return c;
}

describe("PlaceCallButton", () => {
  it("sem calls:place não aparece", () => {
    renderBtn({ id: 5, connectionId: 1 }, [wa()], { canPlace: false });
    expect(screen.queryByTestId("place-call")).not.toBeInTheDocument();
  });

  it.each([{ isGroup: true }, { isCommunity: true }, { isSubGroup: true }])("em %j não é oferecido", (t) => {
    renderBtn({ id: 5, connectionId: 1, ...t }, [wa()]);
    expect(screen.queryByTestId("place-call")).not.toBeInTheDocument();
  });

  it("ticket individual com conexão conectada: habilitado, e liga com o id do ticket", () => {
    const ctx = renderBtn({ id: 5, connectionId: 1 }, [wa()]);
    const b = screen.getByTestId("place-call");
    expect(b).toBeEnabled();
    expect(b).toHaveAttribute("data-block", "");
    fireEvent.click(b);
    expect(ctx.place).toHaveBeenCalledWith(5);
    expect(ctx.place).not.toHaveBeenCalledWith(5, true);
  });

  it("conexão desconectada: desabilitado, com o motivo", () => {
    renderBtn({ id: 5, connectionId: 1 }, [wa({ status: "DISCONNECTED" })]);
    const b = screen.getByTestId("place-call");
    expect(b).toBeDisabled();
    expect(b).toHaveAttribute("data-block", "disconnected");
  });

  it("conexão com proxy: desabilitado, com o motivo", () => {
    renderBtn({ id: 5, connectionId: 1 }, [wa({ proxyMode: "single" })]);
    expect(screen.getByTestId("place-call")).toBeDisabled();
    expect(screen.getByTestId("place-call")).toHaveAttribute("data-block", "proxy");
  });

  it("conexão desconhecida (ainda carregando) não deixa ligar", () => {
    renderBtn({ id: 5, connectionId: 99 }, [wa()]);
    expect(screen.getByTestId("place-call")).toBeDisabled();
  });

  it("operador em chamada não liga outra", () => {
    renderBtn({ id: 5, connectionId: 1 }, [wa()], { active: baseCall({ phase: "active" }) });
    expect(screen.getByTestId("place-call")).toHaveAttribute("data-block", "busy");
    expect(screen.getByTestId("place-call")).toBeDisabled();
  });

  it("chamada já encerrada (painel ainda aberto) não bloqueia nova ligação", () => {
    renderBtn({ id: 5, connectionId: 1 }, [wa()], { active: baseCall({ phase: "ended" }) });
    expect(screen.getByTestId("place-call")).toBeEnabled();
  });

  describe("videochamada", () => {
    it("habilitado e liga com vídeo; o telefone continua sem vídeo", () => {
      const ctx = renderBtn({ id: 5, connectionId: 1 }, [wa()]);
      const v = screen.getByTestId("place-video-call");
      expect(v).toBeEnabled();
      expect(v).toHaveAttribute("data-block", "");
      fireEvent.click(v);
      expect(ctx.place).toHaveBeenCalledTimes(1);
      expect(ctx.place).toHaveBeenCalledWith(5, true);
    });

    it("fica ao lado do telefone, depois dele", () => {
      renderBtn({ id: 5, connectionId: 1 }, [wa()]);
      const phone = screen.getByTestId("place-call");
      const video = screen.getByTestId("place-video-call");
      expect(phone.compareDocumentPosition(video) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    });

    it("navegador sem suporte a enviar vídeo: só o vídeo fica desabilitado", () => {
      renderBtn({ id: 5, connectionId: 1 }, [wa()], { canSendVideo: false });
      expect(screen.getByTestId("place-video-call")).toBeDisabled();
      expect(screen.getByTestId("place-video-call")).toHaveAttribute("data-block", "video-unsupported");
      expect(screen.getByTestId("place-call")).toBeEnabled();
    });

    it("bloqueado como o telefone (proxy, desconectado, em chamada)", () => {
      renderBtn({ id: 5, connectionId: 1 }, [wa({ proxyMode: "single" })]);
      expect(screen.getByTestId("place-video-call")).toBeDisabled();
      expect(screen.getByTestId("place-video-call")).toHaveAttribute("data-block", "proxy");
    });

    it("operador em chamada não liga outra em vídeo", () => {
      renderBtn({ id: 5, connectionId: 1 }, [wa()], { active: baseCall({ phase: "active" }) });
      expect(screen.getByTestId("place-video-call")).toBeDisabled();
    });

    it("sem calls:place e em grupo não aparece", () => {
      renderBtn({ id: 5, connectionId: 1 }, [wa()], { canPlace: false });
      expect(screen.queryByTestId("place-video-call")).not.toBeInTheDocument();
    });
  });
});
