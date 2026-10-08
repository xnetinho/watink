import { describe, expect, it } from "vitest";
import { callsReducer, initialCallsState } from "../callReducer";
import type { CallEventPayload, CallQuality } from "../types";

const incoming = (id = "C1"): CallEventPayload => ({
  callId: id,
  whatsappId: 1,
  direction: "incoming",
  contact: { name: "Maria", number: "5511999990001" },
  ticketId: 10,
});

const quality: CallQuality = { rttMs: 40, lossPct: 0, jitterMs: 5, level: 3, mosEstimated: 4.3, alerts: [], noPeerAudio: false, silentMs: 0 };

describe("callsReducer", () => {
  it("uma oferta passa a tocar", () => {
    const s = callsReducer(initialCallsState(), { type: "incoming", payload: incoming() });
    expect(Object.keys(s.ringing)).toEqual(["C1"]);
    expect(s.ringing.C1.phase).toBe("ringing");
    expect(s.ringing.C1.contact.name).toBe("Maria");
  });

  it("a mesma oferta repetida não duplica", () => {
    let s = callsReducer(initialCallsState(), { type: "incoming", payload: incoming() });
    s = callsReducer(s, { type: "incoming", payload: incoming() });
    expect(Object.keys(s.ringing)).toHaveLength(1);
  });

  it("o toque some quando OUTRO operador atende", () => {
    let s = callsReducer(initialCallsState(), { type: "incoming", payload: incoming() });
    s = callsReducer(s, { type: "answered", callId: "C1", byMe: false });
    expect(s.ringing).toEqual({});
    expect(s.active).toBeNull();
  });

  it("o toque some quando o chamador desiste (ended)", () => {
    let s = callsReducer(initialCallsState(), { type: "incoming", payload: incoming() });
    s = callsReducer(s, { type: "ended", callId: "C1", endReason: "timeout" });
    expect(s.ringing).toEqual({});
  });

  it("atender troca o toque pela chamada em conexão, sem sumir o contato", () => {
    let s = callsReducer(initialCallsState(), { type: "incoming", payload: incoming() });
    s = callsReducer(s, { type: "accepted", payload: incoming() });
    expect(s.ringing).toEqual({});
    expect(s.active?.phase).toBe("connecting");
    expect(s.active?.contact.name).toBe("Maria");
    expect(s.active?.connectedAt).toBeNull();
  });

  it("o cronômetro só parte quando a mídia conecta de verdade", () => {
    let s = callsReducer(initialCallsState(), { type: "accepted", payload: incoming() });
    expect(s.active?.connectedAt).toBeNull();
    s = callsReducer(s, { type: "state", callId: "C1", state: "connecting" });
    expect(s.active?.phase).toBe("connecting");
    expect(s.active?.connectedAt).toBeNull();
    s = callsReducer(s, { type: "state", callId: "C1", state: "active" });
    expect(s.active?.phase).toBe("active");
    expect(s.active?.connectedAt).toBeGreaterThan(0);
    const t0 = s.active!.connectedAt;
    s = callsReducer(s, { type: "state", callId: "C1", state: "active" });
    expect(s.active?.connectedAt).toBe(t0); // não reinicia
  });

  it("chamada originada começa em 'Chamando…' e vira ativa quando o contato atende", () => {
    let s = callsReducer(initialCallsState(), { type: "originate", payload: { ...incoming(), direction: "outgoing" } });
    expect(s.active?.phase).toBe("calling");
    s = callsReducer(s, { type: "state", callId: "C1", state: "active" });
    expect(s.active?.phase).toBe("active");
  });

  it("eventos de outra chamada não mexem na ativa", () => {
    let s = callsReducer(initialCallsState(), { type: "accepted", payload: incoming("A") });
    s = callsReducer(s, { type: "state", callId: "B", state: "active" });
    s = callsReducer(s, { type: "quality", callId: "B", quality });
    s = callsReducer(s, { type: "ended", callId: "B" });
    expect(s.active?.callId).toBe("A");
    expect(s.active?.phase).toBe("connecting");
    expect(s.active?.quality).toBeNull();
  });

  it("qualidade, silêncio e gravação atualizam a chamada ativa", () => {
    let s = callsReducer(initialCallsState(), { type: "accepted", payload: incoming() });
    s = callsReducer(s, { type: "quality", callId: "C1", quality });
    s = callsReducer(s, { type: "mute", muted: true });
    s = callsReducer(s, { type: "recording", callId: "C1", recording: true });
    expect(s.active?.quality?.level).toBe(3);
    expect(s.active?.muted).toBe(true);
    expect(s.active?.recording).toBe(true);
  });

  it("encerrar guarda o motivo e dispensar limpa; dispensar uma ativa não faz nada", () => {
    let s = callsReducer(initialCallsState(), { type: "accepted", payload: incoming() });
    s = callsReducer(s, { type: "dismiss" });
    expect(s.active).not.toBeNull();
    s = callsReducer(s, { type: "ended", callId: "C1", endReason: "user_ended" });
    expect(s.active?.phase).toBe("ended");
    expect(s.active?.endReason).toBe("user_ended");
    s = callsReducer(s, { type: "dismiss" });
    expect(s.active).toBeNull();
  });

  it("microfone negado fica registrado na chamada", () => {
    let s = callsReducer(initialCallsState(), { type: "accepted", payload: incoming() });
    s = callsReducer(s, { type: "failure", callId: "C1", reason: "mic_denied" });
    expect(s.active?.failure).toBe("mic_denied");
  });

  // O WebSocket de áudio fecha ANTES de o call.ended chegar pelo SSE: o contato desligou, o engine
  // fechou o canal e o navegador registrou "socket". Depois que a chamada acaba, a queda do áudio
  // é consequência do fim, não uma falha a mostrar ao operador.
  it("queda de áudio seguida do fim da chamada não deixa o aviso de falha", () => {
    let s = callsReducer(initialCallsState(), { type: "accepted", payload: incoming() });
    s = callsReducer(s, { type: "failure", callId: "C1", reason: "socket" });
    s = callsReducer(s, { type: "ended", callId: "C1", endReason: "user_ended" });
    expect(s.active?.phase).toBe("ended");
    expect(s.active?.failure).toBeNull();
  });

  it("falha de microfone continua visível depois do fim da chamada", () => {
    let s = callsReducer(initialCallsState(), { type: "accepted", payload: incoming() });
    s = callsReducer(s, { type: "failure", callId: "C1", reason: "mic_denied" });
    s = callsReducer(s, { type: "ended", callId: "C1", endReason: "user_ended" });
    expect(s.active?.failure).toBe("mic_denied");
  });

  it("falha de áudio que chega depois do fim é ignorada", () => {
    let s = callsReducer(initialCallsState(), { type: "accepted", payload: incoming() });
    s = callsReducer(s, { type: "ended", callId: "C1", endReason: "declined" });
    s = callsReducer(s, { type: "failure", callId: "C1", reason: "socket" });
    expect(s.active?.failure).toBeNull();
  });

  it("encerrar pelo painel marca que fui eu, e o fim preserva a marca", () => {
    let s = callsReducer(initialCallsState(), { type: "accepted", payload: incoming() });
    s = callsReducer(s, { type: "endRequested", callId: "C1" });
    s = callsReducer(s, { type: "ended", callId: "C1", endReason: "user_ended" });
    expect(s.active?.endedByMe).toBe(true);
  });

  // Chamada RECEBIDA: o operador clica em Atender, o POST /accept demora, e o engine conecta a mídia
  // e emite call.state "active" ANTES de a resposta do POST virar "accepted". Nesse intervalo a
  // chamada ainda está só em "ringing" (active = null) e o evento era descartado: a fase ficava em
  // "connecting" para sempre e o cronômetro nunca ligava.
  it("call.state active que chega ANTES de accepted não se perde (cronômetro liga)", () => {
    let s = callsReducer(initialCallsState(), { type: "incoming", payload: incoming() });
    s = callsReducer(s, { type: "state", callId: "C1", state: "active" });
    s = callsReducer(s, { type: "accepted", payload: incoming() });
    expect(s.active?.phase).toBe("active");
    expect(s.active?.connectedAt).not.toBeNull();
  });

  it("call.state active de outra chamada que não é a minha não vira active", () => {
    let s = callsReducer(initialCallsState(), { type: "incoming", payload: incoming("OUTRA") });
    s = callsReducer(s, { type: "state", callId: "OUTRA", state: "active" });
    s = callsReducer(s, { type: "accepted", payload: incoming("C1") });
    expect(s.active?.phase).toBe("connecting");
    expect(s.active?.connectedAt).toBeNull();
  });

  it("sem o evento antecipado, accepted segue em connecting até o active chegar", () => {
    let s = callsReducer(initialCallsState(), { type: "incoming", payload: incoming() });
    s = callsReducer(s, { type: "accepted", payload: incoming() });
    expect(s.active?.phase).toBe("connecting");
    s = callsReducer(s, { type: "state", callId: "C1", state: "active" });
    expect(s.active?.phase).toBe("active");
    expect(s.active?.connectedAt).not.toBeNull();
  });

  it("videochamada: a mídia vai do toque para a chamada ativa", () => {
    let s = callsReducer(initialCallsState(), { type: "incoming", payload: { ...incoming(), media: "video" } });
    expect(s.ringing.C1.media).toBe("video");
    s = callsReducer(s, { type: "accepted", payload: incoming() });
    expect(s.active?.media).toBe("video");
  });

  it("sem media no evento, a chamada é de voz (engine antigo)", () => {
    let s = callsReducer(initialCallsState(), { type: "incoming", payload: incoming() });
    expect(s.ringing.C1.media).toBe("audio");
    s = callsReducer(s, { type: "accepted", payload: incoming() });
    expect(s.active?.media).toBe("audio");
  });

  it("chamada de saída guarda a mídia pedida", () => {
    const s = callsReducer(initialCallsState(), { type: "originate", payload: { ...incoming(), direction: "outgoing", media: "video" } });
    expect(s.active?.media).toBe("video");
  });

  it("pausado: toques que chegam não aparecem e os que estavam na tela somem", () => {
    let s = callsReducer(initialCallsState(), { type: "incoming", payload: incoming("A") });
    s = callsReducer(s, { type: "pause", paused: true });
    expect(s.ringing).toEqual({});
    s = callsReducer(s, { type: "incoming", payload: incoming("B") });
    expect(s.ringing).toEqual({});
    s = callsReducer(s, { type: "pause", paused: false });
    s = callsReducer(s, { type: "incoming", payload: incoming("C") });
    expect(Object.keys(s.ringing)).toEqual(["C"]);
  });

  it("pausar não derruba a chamada que já está em andamento", () => {
    let s = callsReducer(initialCallsState(), { type: "accepted", payload: incoming() });
    s = callsReducer(s, { type: "pause", paused: true });
    expect(s.active?.phase).toBe("connecting");
  });

  it("várias ofertas tocam ao mesmo tempo e cada uma some sozinha", () => {
    let s = callsReducer(initialCallsState(), { type: "incoming", payload: incoming("A") });
    s = callsReducer(s, { type: "incoming", payload: incoming("B") });
    expect(Object.keys(s.ringing).sort()).toEqual(["A", "B"]);
    s = callsReducer(s, { type: "ended", callId: "A" });
    expect(Object.keys(s.ringing)).toEqual(["B"]);
  });

  it("a câmera liga limpando a falha e, ao falhar, desliga guardando o motivo", () => {
    let s = callsReducer(initialCallsState(), { type: "accepted", payload: incoming() });
    expect(s.active?.camera).toBe(false);
    s = callsReducer(s, { type: "camera", on: false, failure: "denied" });
    expect(s.active?.camera).toBe(false);
    expect(s.active?.cameraFailure).toBe("denied");
    s = callsReducer(s, { type: "camera", on: true });
    expect(s.active?.camera).toBe(true);
    expect(s.active?.cameraFailure).toBeNull();
    s = callsReducer(s, { type: "camera", on: false });
    expect(s.active?.camera).toBe(false);
    expect(s.active?.cameraFailure).toBeNull();
  });

  it("sem chamada ativa o evento de câmera não faz nada", () => {
    const s = callsReducer(initialCallsState(), { type: "camera", on: true });
    expect(s.active).toBeNull();
  });
});
