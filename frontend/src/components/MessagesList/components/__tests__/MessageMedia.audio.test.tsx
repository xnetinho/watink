import React from "react";
import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";

vi.mock("../../../../helpers/urlUtils", () => ({ getBackendUrl: (u?: string) => (u ? `http://srv${u}` : u) }));
vi.mock("../../../../lib/audio/waveform", async (orig) => ({ ...(await orig<typeof import("../../../../lib/audio/waveform")>()), loadWaveform: vi.fn().mockResolvedValue(null) }));

import MessageMedia from "../MessageMedia";
import { senderOf } from "../MessageBubble";
import type { Message } from "../../types";

const audio = (over: Partial<Message> = {}): Message => ({
  id: "a1",
  body: "",
  fromMe: false,
  mediaType: "audio",
  mediaUrl: "/public/media/a.ogg",
  createdAt: "2026-10-29T10:00:00Z",
  dataJson: JSON.stringify({ mimetype: "audio/ogg; codecs=opus" }),
  ...over,
});

describe("áudio no chat: remetente e tipo", () => {
  it("mensagem de voz do Watink (voice) mostra o microfone", () => {
    render(<MessageMedia message={audio({ dataJson: JSON.stringify({ voice: true, mimetype: "audio/ogg" }) })} sender={{ name: "Ana Lima" }} />);
    expect(screen.getByTestId("audio-voice-badge")).toBeInTheDocument();
  });

  it("voz recebida do celular (rich.voice) mostra o microfone; arquivo de áudio não", () => {
    const { rerender } = render(<MessageMedia message={audio({ dataJson: JSON.stringify({ rich: { voice: true } }) })} sender={{ name: "Ana" }} />);
    expect(screen.getByTestId("audio-voice-badge")).toBeInTheDocument();
    rerender(<MessageMedia message={audio()} sender={{ name: "Ana" }} />);
    expect(screen.queryByTestId("audio-voice-badge")).toBeNull();
  });

  it("o avatar leva foto ou iniciais do remetente", () => {
    const { rerender, container } = render(<MessageMedia message={audio()} sender={{ name: "Diomedes Neto", picUrl: "http://srv/f.jpg" }} />);
    expect(container.querySelector('[data-testid="audio-avatar"] img')?.getAttribute("src")).toBe("http://srv/f.jpg");
    rerender(<MessageMedia message={audio()} sender={{ name: "Diomedes Neto" }} />);
    expect(screen.getByTestId("audio-avatar").textContent).toContain("DN");
  });
});

describe("senderOf (quem aparece no avatar do áudio)", () => {
  const peer = { name: "Diomedes Neto", picUrl: "/pic.jpg" };

  it("1:1 recebido → o contato, com a foto dele", () => {
    expect(senderOf(audio(), false, peer, { name: "Comercial" })).toEqual({ name: "Diomedes Neto", picUrl: "http://srv/pic.jpg" });
  });

  it("1:1 enviado por nós → a conexão (iniciais; o Watink não guarda a foto de quem envia)", () => {
    expect(senderOf(audio({ fromMe: true }), false, peer, { name: "Comercial" })).toEqual({ name: "Comercial", picUrl: null });
  });

  it("grupo → quem falou, com a foto individual do participante", () => {
    const m = audio({ dataJson: JSON.stringify({ pushName: "Maria", senderPicUrl: "/m.jpg" }) });
    expect(senderOf(m, true, peer, { name: "Comercial" })).toEqual({ name: "Maria", picUrl: "http://srv/m.jpg" });
  });

  it("grupo sem nome nem foto → cai no número do participante", () => {
    const m = audio({ participant: "5583988880001@s.whatsapp.net", dataJson: "{}" });
    expect(senderOf(m, true, peer).name).toBe("5583988880001");
  });

  it("sem dados do outro lado, não quebra", () => {
    expect(senderOf(audio(), false, undefined, undefined)).toEqual({ name: undefined, picUrl: null });
  });
});
