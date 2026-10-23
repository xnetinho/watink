import { expect, describe, it } from "vitest";
import { messagesReducer } from "../messagesReducer";
import { Message } from "../../types";

const msg = (id: number, createdAt: Message["createdAt"]): Message => ({
  id,
  body: `msg-${id}`,
  fromMe: false,
  createdAt,
});

describe("messagesReducer — sortByDate", () => {
  it("orders a unix-seconds timestamp correctly against ISO-dated messages, instead of collapsing it to 1970", () => {
    // 1722259200 = 2024-07-29T12:00:00Z in seconds -- earlier than both ISO
    // messages below, but interpreted as milliseconds (the pre-fix bug) it
    // becomes 1970-01-20, which would still sort first but for the wrong
    // reason and would break as soon as any ISO message predates 1970+20d.
    const unixSeconds = msg(1, 1722259200 as unknown as string);
    const isoMid = msg(2, "2026-01-01T00:00:00.000Z");
    const isoLate = msg(3, "2026-06-01T00:00:00.000Z");

    const state = messagesReducer([], {
      type: "LOAD_MESSAGES",
      payload: [isoLate, isoMid, unixSeconds],
    });

    expect(state.map((m) => m.id)).toEqual([1, 2, 3]);
  });

  it("keeps a numeric createdAt from causing a 1970-style ordering bug against 2026 dates", () => {
    // Same value as above, but this time the ISO message predates it if
    // treated as milliseconds — proves the fix isn't accidentally correct
    // only because 1970 happens to sort first.
    const isoBefore1970Equivalent = msg(1, "1970-01-01T00:00:00.000Z");
    const unixSeconds = msg(2, 1722259200 as unknown as string);

    const state = messagesReducer([], {
      type: "LOAD_MESSAGES",
      payload: [unixSeconds, isoBefore1970Equivalent],
    });

    expect(state.map((m) => m.id)).toEqual([1, 2]);
  });

  it("still sorts correctly when all messages use ISO date strings", () => {
    const a = msg(1, "2026-01-01T00:00:00.000Z");
    const b = msg(2, "2026-03-01T00:00:00.000Z");
    const c = msg(3, "2026-05-01T00:00:00.000Z");

    const state = messagesReducer([], {
      type: "LOAD_MESSAGES",
      payload: [c, a, b],
    });

    expect(state.map((m) => m.id)).toEqual([1, 2, 3]);
  });

  it("breaks a same-second timestamp tie deterministically by id (issue #414)", () => {
    // WhatsApp timestamps carry only second precision -- two messages sent
    // within the same second arrive with an identical createdAt. Without an
    // explicit tiebreak the resulting order depends on whatever order the
    // backend/network happened to deliver them in, which corrupts the
    // reply/reply-back flow shown in the chat.
    const same = "2026-07-30T12:00:00.000Z";
    const a = msg(2, same);
    const b = msg(10, same);
    const c = msg(1, same);

    const state = messagesReducer([], {
      type: "LOAD_MESSAGES",
      payload: [b, a, c],
    });

    // Tiebreak is string comparison of id (matches the backend's `id DESC`
    // tiebreak, reversed the same way createdAt is) -- "1" < "10" < "2" lexically.
    expect(state.map((m) => m.id)).toEqual([1, 10, 2]);
  });

  it("sorts correctly when timestamps are already in milliseconds", () => {
    const a = msg(1, 1700000000000 as unknown as string);
    const b = msg(2, 1750000000000 as unknown as string);

    const state = messagesReducer([], {
      type: "LOAD_MESSAGES",
      payload: [b, a],
    });

    expect(state.map((m) => m.id)).toEqual([1, 2]);
  });
});

describe("messagesReducer — resposta enviada", () => {
  it("ADD_MESSAGE mantém o quotedMsg que veio no evento, para a citação aparecer na hora", () => {
    const reply = {
      ...msg(10, "2026-10-07T11:00:00.000Z"),
      fromMe: true,
      body: "5 dias",
      quotedMsg: { ...msg(9, "2026-10-07T10:59:00.000Z"), body: "qual o prazo?" },
    } as Message;

    const state = messagesReducer([msg(9, "2026-10-07T10:59:00.000Z")], { type: "ADD_MESSAGE", payload: reply });

    const added = state.find((m) => m.id === 10);
    expect(added?.quotedMsg?.body).toBe("qual o prazo?");
  });
});

// Regressão: a citação aparecia na hora e SUMIA logo depois. Ack, mídia, reação e revogação reemitem a
// mensagem lida do banco (sem quotedMsg, que só a listagem e o envio anexam), e o UPDATE_MESSAGE trocava o
// objeto inteiro. A citação só voltava ao recarregar a página.
describe("messagesReducer — update não apaga a citação", () => {
  const quoted = { ...msg(9, "2026-10-07T10:59:00.000Z"), body: "qual o prazo?" } as Message;
  const reply = { ...msg(10, "2026-10-07T11:00:00.000Z"), fromMe: true, body: "5 dias", ack: 0, quotedMsg: quoted } as Message;

  it("o ack (sem quotedMsg) atualiza o ack e mantém a citação", () => {
    const ack = { ...msg(10, "2026-10-07T11:00:00.000Z"), fromMe: true, body: "5 dias", ack: 2 } as Message;
    const state = messagesReducer([reply], { type: "UPDATE_MESSAGE", payload: ack });
    expect(state[0].ack).toBe(2);
    expect(state[0].quotedMsg?.body).toBe("qual o prazo?");
  });

  it("um update que TRAZ quotedMsg continua valendo", () => {
    const other = { ...quoted, body: "editada" } as Message;
    const state = messagesReducer([reply], { type: "UPDATE_MESSAGE", payload: { ...reply, quotedMsg: other } as Message });
    expect(state[0].quotedMsg?.body).toBe("editada");
  });

  it("ADD_MESSAGE repetido (re-entrega) também não apaga a citação", () => {
    const again = { ...msg(10, "2026-10-07T11:00:00.000Z"), fromMe: true, body: "5 dias" } as Message;
    const state = messagesReducer([reply], { type: "ADD_MESSAGE", payload: again });
    expect(state[0].quotedMsg?.body).toBe("qual o prazo?");
  });

  it("mensagem sem citação continua sem citação", () => {
    const plain = msg(11, "2026-10-07T11:01:00.000Z");
    const state = messagesReducer([plain], { type: "UPDATE_MESSAGE", payload: { ...plain, ack: 1 } as Message });
    expect(state[0].quotedMsg).toBeUndefined();
  });
});
