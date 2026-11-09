import { beforeEach, describe, expect, it, vi } from "vitest";

const post = vi.fn();
vi.mock("../../../services/api", () => ({ default: { post: (...a: unknown[]) => post(...a) } }));
vi.mock("../../../errors/toastError", () => ({ default: vi.fn() }));

import { enqueueMessage, retryMessage, subscribeOutbox } from "../outbox";
import type { Message, MessagesAction } from "../types";

const mk = (id: string, body = "oi") => ({ id, clientMessageId: id, body, fromMe: true, ack: 0, createdAt: "2026-11-09T10:00:00Z" }) as Message;
const flush = () => new Promise((r) => setTimeout(r, 0));
const build = (body: string) => async () => ({ body });

describe("outbox", () => {
  let actions: MessagesAction[];
  beforeEach(() => {
    post.mockReset();
    actions = [];
  });

  it("mostra a mensagem antes de o POST responder", async () => {
    post.mockReturnValue(new Promise(() => undefined));
    subscribeOutbox("1", (a) => actions.push(a));
    enqueueMessage("1", mk("local-a"), build("oi"));
    expect(actions[0]).toMatchObject({ type: "ADD_MESSAGE", payload: { id: "local-a", ack: 0 } });
  });

  it("envia em fila: a segunda só sai depois da primeira terminar", async () => {
    let release: (v: unknown) => void = () => undefined;
    post.mockImplementationOnce(() => new Promise((r) => (release = r))).mockResolvedValue({ data: { id: "B" } });
    subscribeOutbox("2", () => undefined);
    enqueueMessage("2", mk("local-1", "primeira"), build("primeira"));
    enqueueMessage("2", mk("local-2", "segunda"), build("segunda"));
    await flush();
    expect(post).toHaveBeenCalledTimes(1);
    release({ data: { id: "A" } });
    await flush();
    await flush();
    expect(post).toHaveBeenCalledTimes(2);
    expect(post.mock.calls.map((c) => (c[1] as { body: string }).body)).toEqual(["primeira", "segunda"]);
    expect(post.mock.calls[0][1]).toMatchObject({ clientMessageId: "local-1" });
  });

  it("falha vira ack 5 e o retry reenvia com o mesmo id", async () => {
    post.mockRejectedValueOnce(new Error("rede")).mockResolvedValueOnce({ data: { id: "R" } });
    subscribeOutbox("3", (a) => actions.push(a));
    enqueueMessage("3", mk("local-f"), build("oi"));
    await flush();
    await flush();
    expect(actions).toContainEqual({ type: "UPDATE_MESSAGE", payload: expect.objectContaining({ id: "local-f", ack: 5 }) });
    retryMessage("local-f");
    await flush();
    await flush();
    expect(post).toHaveBeenCalledTimes(2);
    expect(post.mock.calls[1][1]).toMatchObject({ clientMessageId: "local-f" });
    expect(actions).toContainEqual({ type: "CONFIRM_LOCAL", clientId: "local-f", id: "R", ack: undefined });
  });

  it("uma falha não trava a fila: a seguinte ainda é enviada", async () => {
    post.mockRejectedValueOnce(new Error("x")).mockResolvedValueOnce({ data: { id: "S" } });
    subscribeOutbox("4", () => undefined);
    enqueueMessage("4", mk("local-x"), build("a"));
    enqueueMessage("4", mk("local-y"), build("b"));
    await flush();
    await flush();
    await flush();
    expect(post).toHaveBeenCalledTimes(2);
  });

  it("quem abre o ticket depois recebe o que ainda está pendente ou falho", async () => {
    post.mockRejectedValue(new Error("x"));
    enqueueMessage("5", mk("local-p"), build("a"));
    await flush();
    await flush();
    subscribeOutbox("5", (a) => actions.push(a));
    expect(actions).toContainEqual({ type: "ADD_MESSAGE", payload: expect.objectContaining({ id: "local-p", ack: 5 }) });
  });
});
