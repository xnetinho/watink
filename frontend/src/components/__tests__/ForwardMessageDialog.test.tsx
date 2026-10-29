import React from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";

const get = vi.fn();
const post = vi.fn();
vi.mock("../../services/api", () => ({ default: { get: (...a: unknown[]) => get(...a), post: (...a: unknown[]) => post(...a) } }));
vi.mock("react-toastify", () => ({ toast: { success: vi.fn(), warning: vi.fn(), error: vi.fn() } }));

import ForwardMessageDialog from "../ForwardMessageDialog";
import { toast } from "react-toastify";

const tickets = [
  { id: 2, contact: { name: "Ana" }, channel: "telegram", connection: { kind: "telegram", id: 1, name: "Jarvis", source: "channel" } },
  { id: 3, contact: { name: "Bruno" }, channel: "telegram", connection: { kind: "telegram", id: 1, name: "Jarvis", source: "channel" } },
  { id: 9, contact: { name: "Atual" }, channel: "telegram" },
];

const open = () => render(<ForwardMessageDialog open onClose={vi.fn()} messageId="m1" channel="telegram" currentTicketId={9} />);

describe("ForwardMessageDialog", () => {
  beforeEach(() => {
    get.mockReset().mockResolvedValue({ data: { tickets } });
    post.mockReset();
    vi.clearAllMocks();
  });

  it("lista só o canal da mensagem, sem a conversa atual", async () => {
    open();
    await screen.findByText("Ana");
    expect(get).toHaveBeenCalledWith("/tickets", { params: expect.objectContaining({ channel: "telegram", isGroup: "all" }) });
    expect(screen.queryByText("Atual")).toBeNull();
    expect(screen.getByText(/Só aparecem conversas do mesmo canal \(Telegram\)/)).toBeInTheDocument();
  });

  it("envia para vários destinos, no modo escolhido", async () => {
    post.mockResolvedValue({ data: { results: [{ ticketId: 2, ok: true }, { ticketId: 3, ok: true }] } });
    open();
    await screen.findByText("Ana");
    fireEvent.click(screen.getByLabelText("Ana"));
    fireEvent.click(screen.getByLabelText("Bruno"));
    fireEvent.click(screen.getByText("Reenviar como nova"));
    fireEvent.click(screen.getByText("Enviar"));
    await waitFor(() => expect(post).toHaveBeenCalledWith("/message/m1/forward", { ticketIds: [2, 3], mode: "copy" }));
    await waitFor(() => expect(toast.success).toHaveBeenCalled());
  });

  it("Enviar fica desligado sem destino", async () => {
    open();
    await screen.findByText("Ana");
    expect(screen.getByText("Enviar").closest("button")).toBeDisabled();
  });

  it("falha parcial explica o motivo traduzido e mantém só o que falhou", async () => {
    post.mockResolvedValue({ data: { results: [{ ticketId: 2, ok: true }, { ticketId: 3, ok: false, code: "forward_not_supported" }] } });
    open();
    await screen.findByText("Ana");
    fireEvent.click(screen.getByLabelText("Ana"));
    fireEvent.click(screen.getByLabelText("Bruno"));
    fireEvent.click(screen.getByText("Enviar"));
    await waitFor(() => expect(toast.error).toHaveBeenCalledWith(expect.stringContaining("Reenviar como nova")));
    expect(screen.getByText("1 selecionada(s)")).toBeInTheDocument();
  });
});
