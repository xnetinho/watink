import React from "react";
import { beforeAll, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import IncomingCallModal from "../IncomingCallModal";
import { baseCall, makeCtx, withCalls } from "./helpers";

beforeAll(() => {
  // jsdom não implementa reprodução de mídia
  window.HTMLMediaElement.prototype.play = vi.fn(() => Promise.resolve());
  window.HTMLMediaElement.prototype.pause = vi.fn();
});

describe("IncomingCallModal", () => {
  it("sem toque não mostra o modal", () => {
    render(withCalls(makeCtx(), <IncomingCallModal />));
    expect(screen.queryByTestId("incoming-call")).not.toBeInTheDocument();
  });

  it("com toque mostra o nome do contato, a conexão e os botões", () => {
    render(
      withCalls(
        makeCtx({ ringing: [baseCall()] }),
        <IncomingCallModal connectionName={(id) => (id === 1 ? "Vendas" : undefined)} />,
      ),
    );
    expect(screen.getByTestId("incoming-call")).toBeInTheDocument();
    expect(screen.getByText("Maria Souza")).toBeInTheDocument();
    expect(screen.getByText(/Vendas/)).toBeInTheDocument();
    expect(screen.getByTestId("accept-call")).toBeEnabled();
    expect(screen.getByTestId("reject-call")).toBeEnabled();
  });

  it("sem nome usa o número; sem nenhum usa o texto padrão", () => {
    const { rerender } = render(
      withCalls(makeCtx({ ringing: [baseCall({ contact: { number: "5511888" } })] }), <IncomingCallModal />),
    );
    expect(screen.getByText("5511888")).toBeInTheDocument();
    rerender(withCalls(makeCtx({ ringing: [baseCall({ contact: {} })] }), <IncomingCallModal />));
    expect(screen.getByText("Contato sem nome")).toBeInTheDocument();
  });

  it("Atender e Recusar chamam as ações com o id da chamada", () => {
    const ctx = makeCtx({ ringing: [baseCall({ callId: "ABC" })] });
    render(withCalls(ctx, <IncomingCallModal />));
    fireEvent.click(screen.getByTestId("accept-call"));
    expect(ctx.accept).toHaveBeenCalledWith("ABC");
    fireEvent.click(screen.getByTestId("reject-call"));
    expect(ctx.reject).toHaveBeenCalledWith("ABC");
  });

  it("quem já está em chamada não pode atender outra, mas pode recusar", () => {
    const ctx = makeCtx({ ringing: [baseCall()], active: baseCall({ callId: "OUTRA", phase: "active" }) });
    render(withCalls(ctx, <IncomingCallModal />));
    expect(screen.getByTestId("accept-call")).toBeDisabled();
    expect(screen.getByTestId("reject-call")).toBeEnabled();
  });

  it("toca o som em loop enquanto houver toque e para quando acaba", () => {
    const play = window.HTMLMediaElement.prototype.play as ReturnType<typeof vi.fn>;
    play.mockClear();
    const { rerender } = render(withCalls(makeCtx({ ringing: [baseCall()] }), <IncomingCallModal />));
    expect(play).toHaveBeenCalled();
    const audio = document.querySelector("audio") as HTMLAudioElement;
    expect(audio.loop).toBe(true);
    const pause = window.HTMLMediaElement.prototype.pause as ReturnType<typeof vi.fn>;
    pause.mockClear();
    rerender(withCalls(makeCtx({ ringing: [] }), <IncomingCallModal />));
    expect(pause).toHaveBeenCalled();
  });

  it("várias ofertas: mostra a primeira e indica quantas esperam", () => {
    const ctx = makeCtx({ ringing: [baseCall({ callId: "A" }), baseCall({ callId: "B", contact: { name: "Outro" } })] });
    render(withCalls(ctx, <IncomingCallModal />));
    expect(screen.getByText("Maria Souza")).toBeInTheDocument();
    expect(screen.getByText("+1")).toBeInTheDocument();
  });
});
