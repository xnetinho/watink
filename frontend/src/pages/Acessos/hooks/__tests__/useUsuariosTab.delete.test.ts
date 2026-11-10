import { beforeEach, describe, expect, it, vi } from "vitest";
import { act, renderHook, waitFor } from "@testing-library/react";

const get = vi.fn();
const del = vi.fn();
vi.mock("../../../../services/api", () => ({ default: { get: (...a: unknown[]) => get(...a), delete: (...a: unknown[]) => del(...a) } }));
const toastError = vi.fn();
vi.mock("../../../../errors/toastError", () => ({ default: (...a: unknown[]) => toastError(...a) }));
vi.mock("react-toastify", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

import { useUsuariosTab } from "../useUsuariosTab";

const user = { id: 7, name: "Ana", email: "a@x", connectionId: null, tenantId: "t", cargoId: null, alcance: "proprio", configs: "{}", createdAt: "", updatedAt: "" };

beforeEach(() => {
  get.mockReset().mockImplementation((url: string) => Promise.resolve({ data: url === "/users" ? { users: [user] } : [] }));
  del.mockReset();
  toastError.mockReset();
});

describe("excluir usuário que é responsável por contatos", () => {
  it("409 abre o pedido de destino da carteira com a contagem, sem mostrar erro", async () => {
    del.mockRejectedValueOnce({ response: { status: 409, data: { error: "ERR_USER_HAS_WALLET", contacts: 5 } } });
    const { result } = renderHook(() => useUsuariosTab());
    await waitFor(() => expect(result.current.users).toHaveLength(1));

    await act(async () => result.current.deleteUser(user as never));

    expect(result.current.walletRequest?.contacts).toBe(5);
    expect(toastError).not.toHaveBeenCalled();
  });

  it("confirmar repete a exclusão com walletTo e fecha o pedido", async () => {
    del.mockRejectedValueOnce({ response: { status: 409, data: { contacts: 5 } } }).mockResolvedValueOnce({ data: {} });
    const { result } = renderHook(() => useUsuariosTab());
    await waitFor(() => expect(result.current.users).toHaveLength(1));
    await act(async () => result.current.deleteUser(user as never));

    await act(async () => result.current.deleteUserWithWallet(user as never, 9));

    expect(del).toHaveBeenLastCalledWith("/users/7", { params: { walletTo: 9 } });
    expect(result.current.walletRequest).toBeNull();
  });

  it("409 sem contagem de carteira (dono do tenant, último admin) continua sendo aviso comum", async () => {
    del.mockRejectedValueOnce({ response: { status: 409, data: { error: "não é possível excluir o dono do tenant" } } });
    const { result } = renderHook(() => useUsuariosTab());
    await waitFor(() => expect(result.current.users).toHaveLength(1));

    await act(async () => result.current.deleteUser(user as never));

    expect(result.current.walletRequest).toBeNull();
    expect(toastError).toHaveBeenCalled();
  });
});
