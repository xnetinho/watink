import React from "react";
import { expect, describe, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import Connections from "../index";
import MessageInput from "@/components/MessageInput";
import Calls from "@/pages/Calls";
import { AuthContext } from "@/context/Auth/AuthContext";
import { i18n } from "@/translate/i18n";

void i18n.changeLanguage("pt");

// Mocks de roteamento e navegação
const mockNavigate = vi.fn();
vi.mock("react-router", () => ({
  useNavigate: () => mockNavigate,
  useParams: () => ({ ticketId: "123" }),
}));

// Mocks de API
const mockApiGet = vi.fn();
const mockApiPost = vi.fn();
const mockApiDelete = vi.fn();
vi.mock("@/services/api", () => ({
  default: {
    get: (...args: unknown[]) => mockApiGet(...args),
    post: (...args: unknown[]) => mockApiPost(...args),
    delete: (...args: unknown[]) => mockApiDelete(...args),
  },
}));

vi.mock("@/context/DarkMode", () => ({
  useThemeContext: () => ({ appTheme: "light" }),
}));

const mockSendMessage = vi.fn();
vi.mock("@/components/MessageInput/hooks/useMessageInput", () => ({
  useMessageInput: () => ({
    medias: [],
    inputMessage: "Olá, mensagem do teste de fluxo",
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
    handleSendMessage: mockSendMessage,
    handleStartRecording: vi.fn(),
    handleUploadAudio: vi.fn(),
    handleCancelAudio: vi.fn(),
    handleMentionClick: vi.fn(),
  }),
}));

const renderWithQueryClient = (ui: React.ReactElement) => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
};

describe("Core UI Flows Component Suite (Tarefa 7.6)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("1. Fluxo de Conexão", () => {
    it("renderiza conexões agnósticas e aciona criação de nova conexão", async () => {
      mockApiGet.mockImplementation((url: string) => {
        if (url.includes("/channels/connections") || url.includes("/connections")) {
          return Promise.resolve({
            data: [
              { id: 1, kind: "whatsapp", name: "WhatsApp Oficial", status: "CONNECTED" },
              { id: 2, kind: "telegram", name: "Bot Suporte", status: "DISCONNECTED", lastError: "Token inválido" },
            ],
          });
        }
        return Promise.resolve({ data: [] });
      });

      renderWithQueryClient(<Connections />);

      // Botão de Nova Conexão disponível
      const newConnBtns = screen.getAllByRole("button", { name: /nova conexão/i });
      expect(newConnBtns.length).toBeGreaterThan(0);

      fireEvent.click(newConnBtns[0]);
      await waitFor(() => {
        expect(screen.getByText("Selecione o Canal de Atendimento")).toBeTruthy();
      });
    });
  });

  describe("2. Fluxo de Envio", () => {
    it("permite envio no composer quando o canal está aberto e ativo", () => {
      render(
        <MessageInput
          ticketStatus="open"
          whatsappStatus="CONNECTED"
          replyPolicy={{ canReply: true }}
        />
      );

      // Botão de envio presente
      const sendBtn = screen.getByRole("button", { name: /enviar mensagem/i });
      expect(sendBtn).toBeTruthy();

      fireEvent.click(sendBtn);
      expect(mockSendMessage).toHaveBeenCalled();
    });

    it("bloqueia o envio e avisa o operador quando a conexão do canal cai", () => {
      render(
        <MessageInput
          ticketStatus="open"
          whatsappStatus="DISCONNECTED"
        />
      );

      expect(screen.getByText("CONEXÃO INTERROMPIDA")).toBeTruthy();
      expect(screen.getByText(/reconecte o canal/i)).toBeTruthy();
      expect(screen.queryByRole("button", { name: /enviar mensagem/i })).toBeNull();
    });
  });

  describe("3. Fluxo de Chamada", () => {
    it("lista chamadas ativas/histórico e exibe status e dados", async () => {
      mockApiGet.mockImplementation((url: string) => {
        if (url.includes("/calls")) {
          return Promise.resolve({
            data: {
              calls: [
                {
                  id: 99,
                  callId: "CALL-FLOW-1",
                  whatsappId: 1,
                  contactId: 10,
                  ticketId: 20,
                  direction: "outgoing",
                  status: "ended",
                  peerJid: "5511999990001",
                  callerPn: "5511999990001",
                  startedAt: "2026-10-06T12:00:00Z",
                  answeredAt: "2026-10-06T12:00:05Z",
                  endedAt: null,
                  durationSec: 42,
                  handledByUserId: 1,
                },
              ],
              total: 1,
            },
          });
        }
        return Promise.resolve({ data: [] });
      });

      render(
        <AuthContext.Provider value={{ user: { id: 1, alcance: "tenant", permissions: ["calls:read"] }, loading: false, isAuth: true } as never}>
          <Calls />
        </AuthContext.Provider>
      );

      await waitFor(() => {
        expect(screen.getByText("5511999990001")).toBeTruthy();
      });
    });
  });
});
