import React from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import Calls from "../index";
import { AuthContext } from "@/context/Auth/AuthContext";
import { i18n } from "@/translate/i18n";
import type { CallLogRow } from "../types";

const get = vi.fn();
const del = vi.fn();
vi.mock("@/services/api", () => ({ default: { get: (...a: unknown[]) => get(...a), delete: (...a: unknown[]) => del(...a) } }));
const toastError = vi.fn();
vi.mock("@/errors/toastError", () => ({ default: (e: unknown) => toastError(e) }));
vi.mock("@/lib/notify", () => ({ default: { success: vi.fn(), error: vi.fn() }, notify: { success: vi.fn(), error: vi.fn() } }));
void i18n.changeLanguage("pt");

const row = (over: Partial<CallLogRow> = {}): CallLogRow => ({
  id: 1, callId: "C1", whatsappId: 1, contactId: 1, ticketId: 1, direction: "incoming", status: "ended", peerJid: "5511999990001@s.whatsapp.net",
  callerPn: "5511999990001", startedAt: "2026-10-06T12:00:00Z", answeredAt: null, endedAt: null, durationSec: 75, handledByUserId: 2,
  endReason: "user_ended", rttAvg: 40, rttMax: 60, lossAvg: 0, lossMax: 1, jitterAvg: 5, jitterMax: 8, mosEstimated: 4.3,
  recordingStatus: "", recordingDurationSec: 0, ...over,
});

function renderPage(permissions: string[], alcance = "proprio") {
  render(
    <AuthContext.Provider value={{ user: { id: 1, alcance, permissions }, loading: false, isAuth: true } as never}>
      <Calls />
    </AuthContext.Provider>,
  );
}

beforeEach(() => {
  get.mockReset();
  del.mockReset();
  toastError.mockReset();
});

describe("página Chamadas", () => {
  it("lista as chamadas com direção, situação, duração e qualidade", async () => {
    get.mockResolvedValue({ data: { calls: [row()], total: 1 } });
    renderPage(["calls:read"]);
    await waitFor(() => expect(screen.getByText("5511999990001")).toBeInTheDocument());
    expect(screen.getByText("Recebida")).toBeInTheDocument();
    expect(screen.getByText("Atendida")).toBeInTheDocument();
    expect(screen.getByText("01:15")).toBeInTheDocument();
    expect(screen.getByTestId("history-quality")).toHaveAttribute("data-level", "3");
    expect(screen.getByText("4,3")).toBeInTheDocument();
  });

  it("lista vazia mostra o estado vazio", async () => {
    get.mockResolvedValue({ data: { calls: [], total: 0 } });
    renderPage(["calls:read"]);
    await waitFor(() => expect(screen.getByText("Nenhuma chamada registrada.")).toBeInTheDocument());
  });

  it("falha ao carregar mostra o erro e permite tentar de novo", async () => {
    get.mockRejectedValueOnce(new Error("500")).mockResolvedValueOnce({ data: { calls: [row()], total: 1 } });
    renderPage(["calls:read"]);
    const retry = await screen.findByRole("button", { name: /tentar|novamente|retry/i });
    fireEvent.click(retry);
    await waitFor(() => expect(screen.getByText("5511999990001")).toBeInTheDocument());
    expect(get).toHaveBeenCalledTimes(2);
  });

  it("chamada perdida não mostra duração, e sem medição não mostra qualidade", async () => {
    get.mockResolvedValue({
      data: { calls: [row({ status: "missed", durationSec: 0, rttMax: null, lossMax: null, jitterMax: null, mosEstimated: null })], total: 1 },
    });
    renderPage(["calls:read"]);
    await waitFor(() => expect(screen.getByText("Perdida")).toBeInTheDocument());
    expect(screen.queryByText("00:00")).not.toBeInTheDocument();
    expect(screen.queryByTestId("history-quality")).not.toBeInTheDocument();
  });

  it("qualidade do resumo usa o pior fator (perda alta = nível ruim)", async () => {
    get.mockResolvedValue({ data: { calls: [row({ lossMax: 9 })], total: 1 } });
    renderPage(["calls:read"]);
    await waitFor(() => expect(screen.getByTestId("history-quality")).toHaveAttribute("data-level", "1"));
  });

  it("gravação pronta oferece Ouvir (sem pedir a URL antes) e mostra o player ao clicar", async () => {
    get.mockImplementation((url: string) =>
      url === "/calls" ? Promise.resolve({ data: { calls: [row({ recordingStatus: "ready" })], total: 1 } }) : Promise.resolve({ data: { url: "https://s3.local/a.mp3?sig=1" } }),
    );
    renderPage(["calls:read"]);
    const listen = await screen.findByTestId("history-listen");
    expect(get).toHaveBeenCalledTimes(1);
    fireEvent.click(listen);
    await waitFor(() => expect(screen.getByTestId("history-player")).toHaveAttribute("src", "https://s3.local/a.mp3?sig=1"));
    expect(get).toHaveBeenCalledWith("/calls/C1/recording");
  });

  it("excluir só aparece com calls:delete", async () => {
    get.mockResolvedValue({ data: { calls: [row({ recordingStatus: "ready" })], total: 1 } });
    renderPage(["calls:read"]);
    await screen.findByTestId("history-listen");
    expect(screen.queryByTestId("history-delete")).not.toBeInTheDocument();
  });

  it("excluir pede confirmação, chama a API e marca a gravação como excluída", async () => {
    get.mockResolvedValue({ data: { calls: [row({ recordingStatus: "ready" })], total: 1 } });
    del.mockResolvedValue({});
    renderPage(["calls:read", "calls:delete"]);
    fireEvent.click(await screen.findByTestId("history-delete"));
    expect(del).not.toHaveBeenCalled();
    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByText("Excluir gravação?")).toBeInTheDocument();
    fireEvent.click(within(dialog).getByRole("button", { name: /confirm|sim|ok|excluir/i }));
    await waitFor(() => expect(del).toHaveBeenCalledWith("/calls/C1/recording"));
    await waitFor(() => expect(screen.getByText("Gravação excluída")).toBeInTheDocument());
    expect(screen.queryByTestId("history-delete")).not.toBeInTheDocument();
  });

  it("cancelar a exclusão não chama a API", async () => {
    get.mockResolvedValue({ data: { calls: [row({ recordingStatus: "ready" })], total: 1 } });
    renderPage(["calls:read", "calls:delete"]);
    fireEvent.click(await screen.findByTestId("history-delete"));
    const dialog = await screen.findByRole("dialog");
    fireEvent.click(within(dialog).getByRole("button", { name: /cancel/i }));
    expect(del).not.toHaveBeenCalled();
  });

  it("gravação falhada e excluída têm texto próprio e nenhum botão", async () => {
    get.mockResolvedValue({
      data: { calls: [row({ id: 1, callId: "A", recordingStatus: "failed" }), row({ id: 2, callId: "B", recordingStatus: "deleted" })], total: 2 },
    });
    renderPage(["calls:read", "calls:delete"]);
    await waitFor(() => expect(screen.getByText("A gravação falhou")).toBeInTheDocument());
    expect(screen.getByText("Gravação excluída")).toBeInTheDocument();
    expect(screen.queryByTestId("history-listen")).not.toBeInTheDocument();
  });

  it("filtrar envia o filtro e volta para a primeira página; sem filtro não envia nada extra", async () => {
    get.mockResolvedValue({ data: { calls: [row()], total: 1 } });
    renderPage(["calls:read"]);
    await waitFor(() => expect(get).toHaveBeenCalledTimes(1));
    expect(get.mock.calls[0][1]).toEqual({ params: { page: 1, pageSize: 20 } });
  });

  it("paginação aparece só com mais de uma página e navega", async () => {
    get.mockResolvedValue({ data: { calls: [row()], total: 45 } });
    renderPage(["calls:read"]);
    const pager = await screen.findByTestId("calls-pager");
    expect(pager).toHaveTextContent("1 / 3");
    fireEvent.click(within(pager).getByText("›"));
    await waitFor(() => expect(get).toHaveBeenLastCalledWith("/calls", { params: { page: 2, pageSize: 20 } }));
    expect(within(pager).getByText("‹")).toBeEnabled();
  });

  it("uma página só não mostra paginação", async () => {
    get.mockResolvedValue({ data: { calls: [row()], total: 3 } });
    renderPage(["calls:read"]);
    await screen.findByText("5511999990001");
    expect(screen.queryByTestId("calls-pager")).not.toBeInTheDocument();
  });
});
