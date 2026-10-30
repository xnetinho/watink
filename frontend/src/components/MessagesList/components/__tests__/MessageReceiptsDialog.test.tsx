import React from "react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, waitFor, within } from "@testing-library/react";

const get = vi.fn();
vi.mock("../../../../services/api", () => ({ default: { get: (...a: unknown[]) => get(...a) } }));

import MessageReceiptsDialog from "../MessageReceiptsDialog";

const row = (over = {}) => ({ senderJid: "", name: "", delivered: null, read: null, played: null, ...over });

describe("Dados da mensagem", () => {
  beforeEach(() => get.mockReset());

  it("conversa individual: mostra entrega e leitura com data e hora, sem nome de participante", async () => {
    get.mockResolvedValue({ data: { receipts: [row({ delivered: "2026-10-30T14:00:00Z", read: "2026-10-30T14:05:00Z" })] } });
    render(<MessageReceiptsDialog messageId="M1" onClose={() => {}} />);
    await waitFor(() => expect(screen.getByTestId("receipt-read")).toHaveTextContent(/\d{2}\/\d{2}\/2026 \d{2}:\d{2}:\d{2}/));
    expect(screen.getByTestId("receipt-delivered")).toHaveTextContent(/30\/10\/2026/);
    expect(screen.getAllByTestId("receipt-row")).toHaveLength(1);
    expect(screen.queryByTestId("receipt-played")).toBeNull();
    expect(get).toHaveBeenCalledWith("/message/M1/receipts");
  });

  it("ainda não lida: a leitura aparece como traço, não como data inventada", async () => {
    get.mockResolvedValue({ data: { receipts: [row({ delivered: "2026-10-30T14:00:00Z" })] } });
    render(<MessageReceiptsDialog messageId="M1" onClose={() => {}} />);
    await waitFor(() => expect(screen.getByTestId("receipt-read")).toHaveTextContent("—"));
  });

  it("áudio: mostra também 'Reproduzida'", async () => {
    get.mockResolvedValue({ data: { receipts: [row({ delivered: "2026-10-30T14:00:00Z", played: "2026-10-30T14:09:00Z" })] } });
    render(<MessageReceiptsDialog messageId="M1" isAudio onClose={() => {}} />);
    await waitFor(() => expect(screen.getByTestId("receipt-played")).toHaveTextContent(/30\/10\/2026/));
  });

  it("grupo: uma seção por participante, com o nome quando existe e o número quando não", async () => {
    get.mockResolvedValue({
      data: {
        receipts: [
          row({ senderJid: "5511988887777@s.whatsapp.net", name: "Maria", delivered: "2026-10-30T14:00:00Z", read: "2026-10-30T14:10:00Z" }),
          row({ senderJid: "5511977776666@s.whatsapp.net", delivered: "2026-10-30T14:01:00Z" }),
        ],
      },
    });
    render(<MessageReceiptsDialog messageId="G1" onClose={() => {}} />);
    await waitFor(() => expect(screen.getAllByTestId("receipt-row")).toHaveLength(2));
    const [maria, outro] = screen.getAllByTestId("receipt-row");
    expect(within(maria).getByText("Maria")).toBeTruthy();
    expect(within(outro).getByText("5511977776666")).toBeTruthy();
    expect(within(outro).getByTestId("receipt-read")).toHaveTextContent("—");
  });

  it("sem nenhum recibo: explica", async () => {
    get.mockResolvedValue({ data: { receipts: [] } });
    render(<MessageReceiptsDialog messageId="M1" onClose={() => {}} />);
    await waitFor(() => expect(screen.getByText(/Ainda não há confirmação/)).toBeTruthy());
  });
});
