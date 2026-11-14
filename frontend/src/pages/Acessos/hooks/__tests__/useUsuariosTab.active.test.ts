import { beforeEach, describe, expect, it, vi } from "vitest";
import { act, renderHook, waitFor } from "@testing-library/react";

const get = vi.fn();
const put = vi.fn();
vi.mock("../../../../services/api", () => ({
  default: { get: (...a: unknown[]) => get(...a), put: (...a: unknown[]) => put(...a) },
}));
const toastError = vi.fn();
vi.mock("../../../../errors/toastError", () => ({ default: (...a: unknown[]) => toastError(...a) }));
vi.mock("react-toastify", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

import { useUsuariosTab } from "../useUsuariosTab";

const activeUser = {
  id: 7,
  name: "Ana",
  email: "a@x",
  connectionId: null,
  tenantId: "t",
  cargoId: null,
  alcance: "proprio",
  active: true,
  configs: "{}",
  createdAt: "",
  updatedAt: "",
};

const userFetches = () => get.mock.calls.filter(([url]) => url === "/users").length;

beforeEach(() => {
  get.mockReset().mockImplementation((url: string) =>
    Promise.resolve({ data: url === "/users" ? { users: [activeUser] } : [] })
  );
  put.mockReset();
  toastError.mockReset();
});

describe("desativar usuário", () => {
  it("envia active=false no próprio usuário e recarrega a lista", async () => {
    put.mockResolvedValueOnce({ data: {} });
    const { result } = renderHook(() => useUsuariosTab());
    await waitFor(() => expect(result.current.users).toHaveLength(1));

    await act(async () => result.current.toggleUserActive(activeUser as never));

    expect(put).toHaveBeenCalledWith("/users/7", { active: false });
    expect(userFetches()).toBe(2);
    expect(toastError).not.toHaveBeenCalled();
  });

  it("reativa quando o usuário já está inativo", async () => {
    const inactive = { ...activeUser, active: false };
    get.mockImplementation((url: string) =>
      Promise.resolve({ data: url === "/users" ? { users: [inactive] } : [] })
    );
    put.mockResolvedValueOnce({ data: {} });
    const { result } = renderHook(() => useUsuariosTab());
    await waitFor(() => expect(result.current.users).toHaveLength(1));

    await act(async () => result.current.toggleUserActive(inactive as never));

    expect(put).toHaveBeenCalledWith("/users/7", { active: true });
  });

  it("409 (dono, último admin, próprio usuário) mostra o erro e mantém a linha", async () => {
    put.mockRejectedValueOnce({
      response: { status: 409, data: { error: "ERR_CANNOT_DEACTIVATE_SELF" } },
    });
    const { result } = renderHook(() => useUsuariosTab());
    await waitFor(() => expect(result.current.users).toHaveLength(1));

    await act(async () => result.current.toggleUserActive(activeUser as never));

    expect(toastError).toHaveBeenCalled();
    expect(userFetches()).toBe(1);
  });
});
