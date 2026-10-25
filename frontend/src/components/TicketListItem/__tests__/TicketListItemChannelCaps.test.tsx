import React from "react";
import { expect, describe, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import TicketListItem from "../index";
import MessageInput from "../../MessageInput";
import type { Ticket } from "../../../types/Ticket";

// Mocks
vi.mock("react-router", () => ({
  useNavigate: () => vi.fn(),
  useParams: () => ({ ticketId: undefined }),
}));

vi.mock("../../../helpers/urlUtils", () => ({
  getBackendUrl: (url: string | null) => url,
}));

vi.mock("../../MarkdownWrapper", () => ({
  default: ({ children }: { children: React.ReactNode }) => <span>{children}</span>,
}));

vi.mock("../../../context/DarkMode", () => ({
  useThemeContext: () => ({ appTheme: "light" }),
}));

vi.mock("../../MessageInput/hooks/useMessageInput", () => ({
  useMessageInput: () => ({
    medias: [],
    inputMessage: "",
    showEmoji: false,
    loading: false,
    recording: false,
    quickAnswers: [],
    typeBar: false,
    anchorElMenu: false,
    mentions: [],
    mentionOpen: false,
    signMessage: false,
    replyingMessage: null,
    inputRef: { current: null },
    setShowEmoji: vi.fn(),
    setAnchorElMenu: vi.fn(),
    setReplyingMessage: vi.fn(),
    setSignMessage: vi.fn(),
    handleChangeInput: vi.fn(),
    handleQuickAnswersClick: vi.fn(),
    handleAddEmoji: vi.fn(),
    handleChangeMedias: vi.fn(),
    handleInputPaste: vi.fn(),
    handleMediaCaptionChange: vi.fn(),
    handleUploadMedia: vi.fn(),
    handleRemoveMedia: vi.fn(),
    handleClearMedias: vi.fn(),
    handleSendMessage: vi.fn(),
    handleStartRecording: vi.fn(),
    handleUploadAudio: vi.fn(),
    handleCancelAudio: vi.fn(),
    handleMentionClick: vi.fn(),
  }),
}));

describe("Agnostic Channel and Capabilities UI (Tarefa 7.3)", () => {
  it("TicketListItem exibe ChannelBadge quando o canal é Telegram", () => {
    const telegramTicket: Ticket = {
      id: 10,
      status: "open",
      updatedAt: new Date().toISOString(),
      unreadMessages: 0,
      lastMessage: "Olá pelo Telegram",
      isGroup: false,
      createdAt: new Date().toISOString(),
      contactId: 100,
      contact: { id: 100, name: "Telegram User", number: "12345" },
      channel: "telegram",
      capabilities: {
        maxTextBytes: 4096,
        initiateConversation: false,
      },
    };

    render(<TicketListItem ticket={telegramTicket} />);
    expect(screen.getByText("Telegram User")).toBeTruthy();
    // ChannelBadge renderiza o canal Telegram
    expect(screen.getByText("Telegram")).toBeTruthy();
  });

  it("TicketListItem não exibe badge secundária de canal para canal WhatsApp nativo", () => {
    const whatsappTicket: Ticket = {
      id: 20,
      status: "open",
      updatedAt: new Date().toISOString(),
      unreadMessages: 0,
      lastMessage: "Olá pelo WhatsApp",
      isGroup: false,
      createdAt: new Date().toISOString(),
      contactId: 200,
      contact: { id: 200, name: "WhatsApp User", number: "5511999990000" },
      channel: "whatsapp",
      capabilities: {
        maxTextBytes: 65535,
        initiateConversation: true,
      },
    };

    render(<TicketListItem ticket={whatsappTicket} />);
    expect(screen.getByText("WhatsApp User")).toBeTruthy();
    expect(screen.queryByText("telegram")).toBeNull();
  });

  it("MessageInput bloqueia o composer quando replyPolicy.canReply é falso", () => {
    render(
      <MessageInput
        ticketStatus="open"
        replyPolicy={{
          canReply: false,
          reason: "window_expired",
          message: "Janela de 24h fechada. Aguarde o cliente.",
        }}
      />
    );

    expect(screen.getByText("Resposta Bloqueada pelo Canal")).toBeTruthy();
    expect(screen.getByText("Janela de 24h fechada. Aguarde o cliente.")).toBeTruthy();
  });

  it("MessageInput permite digitação quando replyPolicy.canReply é verdadeiro", () => {
    render(
      <MessageInput
        ticketStatus="open"
        replyPolicy={{
          canReply: true,
        }}
      />
    );

    expect(screen.queryByText("Resposta Bloqueada pelo Canal")).toBeNull();
  });
});
