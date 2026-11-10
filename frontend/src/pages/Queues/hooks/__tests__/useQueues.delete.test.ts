import { beforeEach, describe, expect, it, vi } from "vitest";
import { act, renderHook, waitFor } from "@testing-library/react";

const get = vi.fn();
const del = vi.fn();
vi.mock("../../../../services/api", () => ({ default: { get: (...a: unknown[]) => get(...a), delete: (...a: unknown[]) => del(...a) } }));
vi.mock("../../../../services/sse-client", () => ({ subscribeToSocket: () => () => undefined }));
const toastError = vi.fn();
vi.mock("../../../../errors/toastError", () => ({ default: (...a: unknown[]) => toastError(...a) }));
vi.mock("react-toastify", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

import { useQueues } from "../useQueues";

const queues = [
  { id: 1, name: "Origem", color: null, greetingMessage: null },
  { id: 2, name: "Destino", color: null, greetingMessage: null },
];
const dependents = { Tickets: 3, Users: 2, Connections: 1, Setores: 0, Children: 0 };

beforeEach(() => {
  get.mockReset().mockResolvedValue({ data: queues });
  del.mockReset();
  toastError.mockReset();
});

describe("excluir fila que ainda tem dependentes", () => {
  it("409 abre o pedido de destino com os números, sem mostrar erro", async () => {
    del.mockRejectedValueOnce({ response: { status: 409, data: { error: "ERR_QUEUE_HAS_DEPENDENTS", dependents } } });
    const { result } = renderHook(() => useQueues());
    await waitFor(() => expect(result.current.queues).toHaveLength(2));

    await act(async () => result.current.handleDeleteQueue(1));

    expect(result.current.moveRequest?.queue.id).toBe(1);
    expect(result.current.moveRequest?.dependents.Tickets).toBe(3);
    expect(toastError).not.toHaveBeenCalled();
  });

  it("confirmar o destino repete a exclusão com moveTo e fecha o pedido", async () => {
    del.mockRejectedValueOnce({ response: { status: 409, data: { dependents } } }).mockResolvedValueOnce({ data: {} });
    const { result } = renderHook(() => useQueues());
    await waitFor(() => expect(result.current.queues).toHaveLength(2));
    await act(async () => result.current.handleDeleteQueue(1));

    await act(async () => result.current.handleMoveAndDelete(1, 2));

    expect(del).toHaveBeenLastCalledWith("/queue/1", { params: { moveTo: 2 } });
    expect(result.current.moveRequest).toBeNull();
  });

  it("outro erro continua virando aviso comum, sem abrir o pedido", async () => {
    del.mockRejectedValueOnce({ response: { status: 500, data: { error: "x" } } });
    const { result } = renderHook(() => useQueues());
    await waitFor(() => expect(result.current.queues).toHaveLength(2));

    await act(async () => result.current.handleDeleteQueue(1));

    expect(result.current.moveRequest).toBeNull();
    expect(toastError).toHaveBeenCalled();
  });

  it("falha ao mover mantém o pedido aberto para tentar de novo", async () => {
    del.mockRejectedValueOnce({ response: { status: 409, data: { dependents } } }).mockRejectedValueOnce({ response: { status: 422, data: { error: "ERR_QUEUE_MOVE_INVALID" } } });
    const { result } = renderHook(() => useQueues());
    await waitFor(() => expect(result.current.queues).toHaveLength(2));
    await act(async () => result.current.handleDeleteQueue(1));

    await act(async () => result.current.handleMoveAndDelete(1, 2));

    expect(result.current.moveRequest).not.toBeNull();
    expect(toastError).toHaveBeenCalled();
  });
});
