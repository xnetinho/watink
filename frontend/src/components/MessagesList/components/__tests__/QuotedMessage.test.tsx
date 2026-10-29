import React from "react";
import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import QuotedMessage from "../QuotedMessage";
import { getQuotePreview } from "../../utils/messageHelpers";
import type { Message } from "../../types";

const reply = (quoted: Record<string, unknown>): Message =>
  ({ id: "r", body: "ok", fromMe: true, createdAt: "2026-10-29T10:00:00Z", quotedMsg: { id: "q", fromMe: false, ...quoted } }) as unknown as Message;

describe("citação: o que a caixa mostra", () => {
  it("voz recebida: rótulo de voz, microfone e duração (antes a caixa ficava vazia)", () => {
    render(<QuotedMessage message={reply({ mediaType: "audio", body: "", dataJson: JSON.stringify({ rich: { voice: true, seconds: 7 } }) })} />);
    expect(screen.getByTestId("quote-content").textContent).toContain("Mensagem de voz");
    expect(screen.getByTestId("quote-content").textContent).toContain("0:07");
  });

  it("arquivo de áudio: rótulo de áudio, sem 'voz'", () => {
    expect(getQuotePreview({ mediaType: "audio", body: "", dataJson: JSON.stringify({ mimetype: "audio/mpeg" }) })).toEqual({ label: "Áudio", audio: { voice: false, seconds: undefined } });
  });

  it("voz enviada pelo Watink (voice) também é voz", () => {
    expect(getQuotePreview({ mediaType: "audio", dataJson: JSON.stringify({ voice: true }) }).audio?.voice).toBe(true);
  });

  it("foto sem legenda: rótulo e miniatura; com legenda: a legenda", () => {
    const thumb = JSON.stringify({ thumbnail: "QUJD" });
    expect(getQuotePreview({ mediaType: "image", body: "", dataJson: thumb })).toEqual({ label: "📷 Foto", thumbnail: "QUJD" });
    expect(getQuotePreview({ mediaType: "image", body: "olha", dataJson: thumb }).label).toBe("olha");
    render(<QuotedMessage message={reply({ mediaType: "image", body: "", dataJson: thumb })} />);
    expect(screen.getByTestId("quote-thumb").getAttribute("src")).toBe("data:image/jpeg;base64,QUJD");
  });

  it("texto e tipo desconhecido não quebram", () => {
    expect(getQuotePreview({ mediaType: "chat", body: "oi" })).toEqual({ label: "oi", thumbnail: undefined });
    expect(getQuotePreview({ mediaType: "zzz", body: "" }).label).toBe("");
  });
});
