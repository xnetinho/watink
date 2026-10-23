import { describe, it, expect, vi, beforeEach } from "vitest";

const get = vi.fn();
const post = vi.fn();
vi.mock("../../../services/api", () => ({
  default: { get: (...a: unknown[]) => get(...a), post: (...a: unknown[]) => post(...a) },
}));

import { findOrCreateContact } from "../index";

describe("findOrCreateContact", () => {
  beforeEach(() => {
    get.mockReset();
    post.mockReset();
  });

  it("usa o contato que já existe na agenda, sem criar outro", async () => {
    get.mockResolvedValue({ data: { contacts: [{ id: 7, number: "558382341576" }] } });
    expect(await findOrCreateContact("Fulano", "558382341576")).toBe(7);
    expect(post).not.toHaveBeenCalled();
  });

  it("busca em /contacts/ (rota real, plural)", async () => {
    get.mockResolvedValue({ data: { contacts: [{ id: 7, number: "558382341576" }] } });
    await findOrCreateContact("Fulano", "558382341576");
    expect(get).toHaveBeenCalledWith("/contacts/", { params: { searchParam: "558382341576" } });
  });

  it("não confunde com número parecido (a busca do backend é por trecho)", async () => {
    get.mockResolvedValue({ data: { contacts: [{ id: 9, number: "5583823415769" }] } });
    post.mockResolvedValue({ data: { id: 12 } });
    expect(await findOrCreateContact("Fulano", "558382341576")).toBe(12);
    expect(post).toHaveBeenCalledWith("/contacts", { name: "Fulano", number: "558382341576", email: "" });
  });

  it("cria quando não existe, usando o número como nome se o vCard não tiver nome", async () => {
    get.mockResolvedValue({ data: { contacts: [] } });
    post.mockResolvedValue({ data: { id: 3 } });
    expect(await findOrCreateContact("", "5511900000009")).toBe(3);
    expect(post).toHaveBeenCalledWith("/contacts", { name: "5511900000009", number: "5511900000009", email: "" });
  });

  it("409 na criação (corrida com outro atendente): busca de novo e usa o existente", async () => {
    get
      .mockResolvedValueOnce({ data: { contacts: [] } })
      .mockResolvedValueOnce({ data: { contacts: [{ id: 21, number: "5511900000009" }] } });
    post.mockImplementation(() => Promise.reject({ response: { status: 409 } }));
    expect(await findOrCreateContact("Fulano", "5511900000009")).toBe(21);
  });

  it("erro que não é 409 sobe para o chamador mostrar", async () => {
    get.mockResolvedValue({ data: { contacts: [] } });
    const boom = { response: { status: 500 } };
    post.mockImplementation(() => Promise.reject(boom));
    await expect(findOrCreateContact("Fulano", "5511900000009")).rejects.toBe(boom);
  });
});
