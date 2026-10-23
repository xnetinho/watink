import React from "react";
import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import MessageBubble from "../MessageBubble";
import { ReplyMessageContext } from "../../../../context/ReplyingMessage/ReplyingMessageContext";
import type { Message } from "../../types";
import "../../../Calls/__tests__/helpers";

vi.mock("@/services/api", () => ({ default: { get: vi.fn(), post: vi.fn() } }));
vi.mock("../../../../services/api", () => ({ default: { get: vi.fn(), post: vi.fn() } }));
vi.mock("@/errors/toastError", () => ({ default: vi.fn() }));

const msg = (over: Partial<Message> = {}): Message =>
  ({ id: "M1", body: "oi, tudo bem?", fromMe: false, ack: 0, createdAt: "2026-10-06T16:00:00.000Z", ticketId: 1, ...over }) as unknown as Message;

function setup(message: Message) {
  const setReplyingMessage = vi.fn();
  render(
    <ReplyMessageContext.Provider value={{ replyingMessage: null, setReplyingMessage } as never}>
      <MessageBubble
        message={message}
        index={0}
        messagesList={[message]}
        appTheme="default"
        mentionsMap={{}}
        colorCache={new Map()}
        onOpenOptions={vi.fn()}
      />
    </ReplyMessageContext.Provider>
  );
  return setReplyingMessage;
}

describe("MessageBubble — duplo clique responde", () => {
  it("duplo clique na mensagem recebida seleciona para responder", () => {
    const set = setup(msg());
    fireEvent.doubleClick(screen.getByText("oi, tudo bem?"));
    expect(set).toHaveBeenCalledTimes(1);
    expect(set.mock.calls[0][0]).toMatchObject({ id: "M1" });
  });

  it("vale também para a mensagem que eu enviei", () => {
    const set = setup(msg({ id: "M2", fromMe: true, body: "minha mensagem" }));
    fireEvent.doubleClick(screen.getByText("minha mensagem"));
    expect(set.mock.calls[0][0]).toMatchObject({ id: "M2" });
  });

  it("clique simples não responde", () => {
    const set = setup(msg());
    fireEvent.click(screen.getByText("oi, tudo bem?"));
    expect(set).not.toHaveBeenCalled();
  });

  it("mensagem apagada não pode ser respondida", () => {
    const set = setup(msg({ isDeleted: true }));
    fireEvent.doubleClick(screen.getByText("oi, tudo bem?"));
    expect(set).not.toHaveBeenCalled();
  });

  it("registro de chamada não pode ser respondido", () => {
    const set = setup(
      msg({ mediaType: "call", body: "Chamada de voz recebida", dataJson: JSON.stringify({ callId: "C1", direction: "incoming", status: "ended", durationSec: 5 }) })
    );
    fireEvent.doubleClick(screen.getByTestId("call-message"));
    expect(set).not.toHaveBeenCalled();
  });

  it("duplo clique no botão de opções não responde (tem ação própria)", () => {
    const set = setup(msg());
    fireEvent.doubleClick(screen.getByRole("button", { name: "Opções da mensagem" }));
    expect(set).not.toHaveBeenCalled();
  });
});
