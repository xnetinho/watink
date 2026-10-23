import React from "react";
import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import MessageMedia from "../MessageMedia";
import type { Message } from "../../types";
import "../../../Calls/__tests__/helpers";

vi.mock("@/services/api", () => ({ default: { get: vi.fn() } }));
vi.mock("@/errors/toastError", () => ({ default: vi.fn() }));

const callMessage = (over: Partial<Message> = {}): Message => ({
  id: "call:C1",
  body: "Chamada de voz perdida",
  fromMe: false,
  mediaType: "call",
  createdAt: new Date().toISOString(),
  dataJson: JSON.stringify({ callId: "C1", direction: "incoming", status: "missed", durationSec: 0, endReason: "no_operator" }),
  ...over,
});

describe("MessageMedia — mensagem de chamada", () => {
  it("o tipo 'call' é renderizado como CallMessage, mesmo sem mediaUrl", () => {
    render(<MessageMedia message={callMessage()} />);
    expect(screen.getByTestId("call-message")).toHaveAttribute("data-status", "missed");
    expect(screen.getByTestId("call-message")).toHaveTextContent("Chamada de voz perdida");
  });

  it("não é tratado como mídia baixável sob demanda (sem botão de download)", () => {
    render(<MessageMedia message={callMessage()} />);
    expect(screen.queryByRole("button", { name: /baixar|download/i })).not.toBeInTheDocument();
  });

  it("mostra o operador e a duração de uma chamada atendida", () => {
    render(
      <MessageMedia
        message={callMessage({
          body: "Chamada de voz recebida",
          dataJson: { callId: "C2", direction: "incoming", status: "ended", durationSec: 130, handledByName: "Bia" },
        })}
      />,
    );
    const m = screen.getByTestId("call-message");
    expect(m).toHaveTextContent("Atendida por Bia");
    expect(m).toHaveTextContent("Duração: 02:10");
  });
});
