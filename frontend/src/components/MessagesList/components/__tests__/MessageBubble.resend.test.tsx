import React from "react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { fireEvent, render, waitFor } from "@testing-library/react";

const post = vi.fn();
const toastError = vi.fn();
vi.mock("../../../../services/api", () => ({ default: { post: (...a: unknown[]) => post(...a), put: vi.fn().mockResolvedValue({}) } }));
vi.mock("react-toastify", () => ({ toast: { error: (...a: unknown[]) => toastError(...a) } }));

import MessageBubble from "../MessageBubble";
import { ReplyMessageContext } from "../../../../context/ReplyingMessage/ReplyingMessageContext";
import type { Message } from "../../types";

const failed = { id: "MSG1", body: "", fromMe: true, ack: 5, mediaType: "audio", mediaUrl: "/public/a.ogg", ticketId: 7, createdAt: "2026-10-29T10:00:00Z", dataJson: "{}" } as unknown as Message;

const renderBubble = () =>
  render(
    <ReplyMessageContext.Provider value={{ replyingMessage: null, setReplyingMessage: vi.fn() } as never}>
      <MessageBubble message={failed} index={0} messagesList={[failed]} appTheme="saas" mentionsMap={{}} colorCache={new Map()} onOpenOptions={vi.fn()} />
    </ReplyMessageContext.Provider>,
  );

describe("reenviar mensagem que falhou", () => {
  beforeEach(() => {
    post.mockReset();
    toastError.mockReset();
  });

  it("chama a rota de reenvio pelo id da mensagem: o servidor reaproveita a mídia (o corpo de um áudio é vazio)", async () => {
    post.mockResolvedValue({});
    const { container } = renderBubble();
    fireEvent.click(container.querySelector("span[title]") as HTMLElement);
    await waitFor(() => expect(post).toHaveBeenCalledWith("/message/MSG1/resend"));
  });

  it("se o reenvio falha, avisa o operador (antes o erro era engolido em silêncio)", async () => {
    post.mockRejectedValue(new Error("x"));
    const { container } = renderBubble();
    fireEvent.click(container.querySelector("span[title]") as HTMLElement);
    await waitFor(() => expect(toastError).toHaveBeenCalledWith("Não foi possível reenviar a mensagem."));
  });
});
