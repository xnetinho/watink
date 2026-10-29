import React from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";

const get = vi.fn();
const post = vi.fn();
vi.mock("../../services/api", () => ({ default: { get: (...a: unknown[]) => get(...a), post: (...a: unknown[]) => post(...a) } }));
vi.mock("react-toastify", () => ({ toast: { success: vi.fn(), warning: vi.fn(), error: vi.fn() } }));

import ForwardMessageDialog from "../ForwardMessageDialog";
import { toast } from "react-toastify";

const contacts = [
  { id: 2, name: "Ana", number: "5511999990001", connectionSource: "whatsapp", connectionId: 1 },
  { id: 3, name: "Bruno", number: "5511999990002", connectionSource: "whatsapp", connectionId: 2 },
  { id: 9, name: "Atual", number: "5511999990009", connectionSource: "whatsapp", connectionId: 1 },
];
const connections = [
  { id: 1, name: "Comercial", status: "CONNECTED" },
  { id: 2, name: "Suporte", status: "CONNECTED" },
];

const open = () => render(<ForwardMessageDialog open onClose={vi.fn()} messageId="m1" channel="whatsapp" currentContactId={9} />);

describe("ForwardMessageDialog (destino = contato)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    get.mockImplementation((url: string) => {
      if (url === "/contacts") return Promise.resolve({ data: { contacts } });
      if (url === "/whatsapp") return Promise.resolve({ data: connections });
      return Promise.resolve({ data: [] });
    });
    post.mockReset();
  });

  it("lista contatos do mesmo canal, sem o da conversa atual, cada um com a conexão a que pertence", async () => {
    open();
    await screen.findByText("Ana");
    expect(get).toHaveBeenCalledWith("/contacts", { params: expect.objectContaining({ channel: "whatsapp" }) });
    expect(screen.queryByText("Atual")).toBeNull();
    await waitFor(() => expect(screen.getByText("Suporte")).toBeInTheDocument());
    expect(screen.getByText("Comercial")).toBeInTheDocument();
  });

  it("envia os contatos escolhidos, no modo escolhido", async () => {
    post.mockResolvedValue({ data: { results: [{ contactId: 2, ok: true }, { contactId: 3, ok: true }] } });
    open();
    await screen.findByText("Ana");
    fireEvent.click(screen.getByLabelText("Ana"));
    fireEvent.click(screen.getByLabelText("Bruno"));
    fireEvent.click(screen.getByText("Reenviar como nova"));
    fireEvent.click(screen.getByText("Enviar"));
    await waitFor(() => expect(post).toHaveBeenCalledWith("/message/m1/forward", { contactIds: [2, 3], mode: "copy" }));
    await waitFor(() => expect(toast.success).toHaveBeenCalled());
  });

  it("Enviar fica desligado sem destino", async () => {
    open();
    await screen.findByText("Ana");
    expect(screen.getByText("Enviar").closest("button")).toBeDisabled();
  });

  it("falha parcial explica o motivo traduzido e mantém só o contato que falhou", async () => {
    post.mockResolvedValue({ data: { results: [{ contactId: 2, ok: true }, { contactId: 3, ok: false, code: "forward_not_supported" }] } });
    open();
    await screen.findByText("Ana");
    fireEvent.click(screen.getByLabelText("Ana"));
    fireEvent.click(screen.getByLabelText("Bruno"));
    fireEvent.click(screen.getByText("Enviar"));
    await waitFor(() => expect(toast.error).toHaveBeenCalledWith(expect.stringContaining("Reenviar como nova")));
    expect(screen.getByText("1 selecionada(s)")).toBeInTheDocument();
  });
});
