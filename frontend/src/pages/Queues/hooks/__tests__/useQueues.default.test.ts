import { beforeEach, describe, expect, it, vi } from "vitest";
import { act, renderHook, waitFor } from "@testing-library/react";

const get = vi.fn();
const post = vi.fn();
vi.mock("../../../../services/api", () => ({
  default: { get: (...a: unknown[]) => get(...a), post: (...a: unknown[]) => post(...a) },
}));
vi.mock("../../../../services/sse-client", () => ({ subscribeToSocket: () => () => undefined }));
const toastError = vi.fn();
vi.mock("../../../../errors/toastError", () => ({ default: (...a: unknown[]) => toastError(...a) }));
vi.mock("react-toastify", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

import { useQueues } from "../useQueues";

const defaultQueue = { id: 1, name: "Padrão", color: null, greetingMessage: null, isDefault: true };
const otherQueue = { id: 2, name: "Comercial", color: null, greetingMessage: null, isDefault: false };

beforeEach(() => {
  get.mockReset().mockResolvedValue({ data: [defaultQueue, otherQueue] });
  post.mockReset();
  toastError.mockReset();
});

const queueFetches = () => get.mock.calls.filter(([url]) => url === "/queue").length;

describe("fila padrão da empresa", () => {
  it("define a fila escolhida e recarrega a lista", async () => {
    post.mockResolvedValueOnce({ data: {} });
    const { result } = renderHook(() => useQueues());
    await waitFor(() => expect(result.current.queues).toHaveLength(2));

    await act(async () => result.current.handleSetDefault(otherQueue));

    expect(post).toHaveBeenCalledWith("/queue/default", { queueId: 2 });
    expect(queueFetches()).toBe(2);
    expect(toastError).not.toHaveBeenCalled();
  });

  it("clicar na fila que já é padrão remove o padrão", async () => {
    post.mockResolvedValueOnce({ data: {} });
    const { result } = renderHook(() => useQueues());
    await waitFor(() => expect(result.current.queues).toHaveLength(2));

    await act(async () => result.current.handleSetDefault(defaultQueue));

    expect(post).toHaveBeenCalledWith("/queue/default", { queueId: null });
  });

  it("recusa do servidor (fila de outra empresa) vira aviso e não recarrega", async () => {
    post.mockRejectedValueOnce({
      response: { status: 422, data: { error: "ERR_DEFAULT_QUEUE_INVALID" } },
    });
    const { result } = renderHook(() => useQueues());
    await waitFor(() => expect(result.current.queues).toHaveLength(2));

    await act(async () => result.current.handleSetDefault(otherQueue));

    expect(toastError).toHaveBeenCalled();
    expect(queueFetches()).toBe(1);
  });
});
