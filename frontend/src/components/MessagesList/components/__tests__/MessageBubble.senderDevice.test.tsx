import React from "react";
import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";

vi.mock("../../../../services/api", () => ({ default: { post: vi.fn(), put: vi.fn() } }));

import MessageBubble from "../MessageBubble";
import { ReplyMessageContext } from "../../../../context/ReplyingMessage/ReplyingMessageContext";
import { i18n } from "../../../../translate/i18n";
import type { Message } from "../../types";

void i18n.changeLanguage("pt");
const base = { id: "3AF2FA95D11B72616BF0", body: "Oi", fromMe: false, ack: 0, ticketId: 7, createdAt: "2026-11-07T16:50:00Z" } as Message;

const renderBubble = (m: Message, channel?: string) =>
  render(
    <ReplyMessageContext.Provider value={{ replyingMessage: null, setReplyingMessage: vi.fn() } as never}>
      <MessageBubble message={m} index={0} messagesList={[m]} channel={channel} appTheme="saas" mentionsMap={{}} colorCache={new Map()} onOpenOptions={vi.fn()} />
    </ReplyMessageContext.Provider>,
  );

describe("balão: aparelho do contato no rodapé", () => {
  it("mensagem recebida do WhatsApp mostra ~iPhone no balão", () => {
    renderBubble(base, "whatsapp");
    expect(screen.getByTestId("sender-device")).toHaveTextContent("~iPhone");
  });

  it("mensagem nossa e mensagem de outro canal não mostram o aparelho", () => {
    renderBubble({ ...base, fromMe: true }, "whatsapp");
    expect(screen.queryByTestId("sender-device")).toBeNull();
    renderBubble(base, "telegram");
    expect(screen.queryByTestId("sender-device")).toBeNull();
  });

  it("mensagem editada mostra só o selo 'Editada' no canto (o aparelho cede o lugar, sem sobrepor)", () => {
    renderBubble({ ...base, editCount: 1 }, "whatsapp");
    expect(screen.getByTestId("edited-badge")).toBeInTheDocument();
    expect(screen.queryByTestId("sender-device")).toBeNull();
  });
});
