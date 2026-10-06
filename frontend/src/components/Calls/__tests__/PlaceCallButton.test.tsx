import React from "react";
import { describe, expect, it } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import PlaceCallButton from "../PlaceCallButton";
import { WhatsAppsContext } from "@/context/WhatsApp/WhatsAppsContext";
import { baseCall, makeCtx, withCalls } from "./helpers";
import type { CallsContextValue } from "@/context/Calls/CallsContext";

const wa = (over: Record<string, unknown> = {}) => ({ id: 1, name: "Vendas", status: "CONNECTED", proxyMode: "none", ...over });

function renderBtn(
  ticket: { id: number; whatsappId?: number; isGroup?: boolean; isCommunity?: boolean; isSubGroup?: boolean },
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
    renderBtn({ id: 5, whatsappId: 1 }, [wa()], { canPlace: false });
    expect(screen.queryByTestId("place-call")).not.toBeInTheDocument();
  });

  it.each([{ isGroup: true }, { isCommunity: true }, { isSubGroup: true }])("em %j não é oferecido", (t) => {
    renderBtn({ id: 5, whatsappId: 1, ...t }, [wa()]);
    expect(screen.queryByTestId("place-call")).not.toBeInTheDocument();
  });

  it("ticket individual com conexão conectada: habilitado, e liga com o id do ticket", () => {
    const ctx = renderBtn({ id: 5, whatsappId: 1 }, [wa()]);
    const b = screen.getByTestId("place-call");
    expect(b).toBeEnabled();
    expect(b).toHaveAttribute("data-block", "");
    fireEvent.click(b);
    expect(ctx.place).toHaveBeenCalledWith(5);
  });

  it("conexão desconectada: desabilitado, com o motivo", () => {
    renderBtn({ id: 5, whatsappId: 1 }, [wa({ status: "DISCONNECTED" })]);
    const b = screen.getByTestId("place-call");
    expect(b).toBeDisabled();
    expect(b).toHaveAttribute("data-block", "disconnected");
  });

  it("conexão com proxy: desabilitado, com o motivo", () => {
    renderBtn({ id: 5, whatsappId: 1 }, [wa({ proxyMode: "single" })]);
    expect(screen.getByTestId("place-call")).toBeDisabled();
    expect(screen.getByTestId("place-call")).toHaveAttribute("data-block", "proxy");
  });

  it("conexão desconhecida (ainda carregando) não deixa ligar", () => {
    renderBtn({ id: 5, whatsappId: 99 }, [wa()]);
    expect(screen.getByTestId("place-call")).toBeDisabled();
  });

  it("operador em chamada não liga outra", () => {
    renderBtn({ id: 5, whatsappId: 1 }, [wa()], { active: baseCall({ phase: "active" }) });
    expect(screen.getByTestId("place-call")).toHaveAttribute("data-block", "busy");
    expect(screen.getByTestId("place-call")).toBeDisabled();
  });

  it("chamada já encerrada (painel ainda aberto) não bloqueia nova ligação", () => {
    renderBtn({ id: 5, whatsappId: 1 }, [wa()], { active: baseCall({ phase: "ended" }) });
    expect(screen.getByTestId("place-call")).toBeEnabled();
  });
});
