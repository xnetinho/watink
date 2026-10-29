import React from "react";
import { expect, describe, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import Connections from "../index";

const mockNavigate = vi.fn();
vi.mock("react-router", () => ({
  useNavigate: () => mockNavigate,
}));

const mockDeleteChannel = vi.fn();
const mockReloadChannels = vi.fn();

vi.mock("../hooks/useConnections", () => ({
  useConnections: () => ({
    whatsApps: [],
    reloadWhatsApps: vi.fn(),
    channelConnections: [
      {
        id: 101,
        kind: "telegram",
        name: "Bot Telegram Suporte",
        status: "DISCONNECTED",
        lastError: "Unauthorized: token was revoked",
        config: JSON.stringify({ mode: "polling" }),
      },
      {
        id: 102,
        kind: "instagram",
        name: "Insta Direct Vendas",
        status: "CONNECTED",
        config: JSON.stringify({ mode: "webhook" }),
      },
    ],
    reloadChannelConnections: mockReloadChannels,
    handleDeleteChannelConnection: mockDeleteChannel,
    whatsAppModalOpen: false,
    webchatModalOpen: false,
    confirmModalOpen: false,
    selectedWhatsApp: null,
    pendingSessionIds: new Set(),
    handleStartSession: vi.fn(),
    handleRequestNewQrCode: vi.fn(),
    handleDisconnectSession: vi.fn(),
    handleDeleteWhatsApp: vi.fn(),
    handleEditWhatsApp: vi.fn(),
    handleOpenNewWhatsApp: vi.fn(),
    handleOpenNewWebchat: vi.fn(),
    handleCloseWhatsAppModal: vi.fn(),
    handleCloseWebchatModal: vi.fn(),
    handleOpenConfirm: vi.fn(),
    handleCloseConfirm: vi.fn(),
    navigate: mockNavigate,
  }),
}));

describe("Unified Channel Connections UI (Tarefa 7.4)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("exibe conexões unificadas com lastError destacado", () => {
    render(<Connections />);

    expect(screen.getByText("Bot Telegram Suporte")).toBeTruthy();
    expect(screen.getByText("Unauthorized: token was revoked")).toBeTruthy();
    expect(screen.getByText("Insta Direct Vendas")).toBeTruthy();
  });

  it("aciona exclusão de conexão ao clicar no botão Excluir", () => {
    render(<Connections />);

    const deleteButtons = screen.getAllByRole("button", { name: /excluir/i });
    expect(deleteButtons.length).toBeGreaterThan(0);

    fireEvent.click(deleteButtons[0]);
    expect(mockDeleteChannel).toHaveBeenCalledWith(101);
  });

  it("abre modal de edição ao clicar no botão Editar", async () => {
    render(<Connections />);

    const editButtons = screen.getAllByRole("button", { name: /editar/i });
    expect(editButtons.length).toBeGreaterThan(0);

    fireEvent.click(editButtons[0]);
    await waitFor(() => {
      expect(screen.getByRole("heading", { name: /Editar Telegram/i })).toBeTruthy();
    });
  });
});
