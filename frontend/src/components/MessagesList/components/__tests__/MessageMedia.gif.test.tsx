import React from "react";
import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import MessageMedia, { isGif } from "../MessageMedia";
import GifTrimmedBadge, { hasGifTrimmedBadge } from "../GifTrimmedBadge";
import { hasFooterBadge } from "../MessageBubble";
import type { Message } from "../../types";
import "../../../Calls/__tests__/helpers";

vi.mock("@/services/api", () => ({ default: { get: vi.fn(), post: vi.fn() } }));
vi.mock("@/errors/toastError", () => ({ default: vi.fn() }));

const msg = (over: Partial<Message> = {}): Message =>
  ({ id: "G1", body: "", fromMe: true, mediaType: "video", mediaUrl: "/public/media/a.mp4", createdAt: new Date().toISOString(), ...over }) as Message;

describe("MessageMedia — GIF", () => {
  it("GIF enviado por nós toca sozinho, em loop, sem som e sem controles", () => {
    render(<MessageMedia message={msg({ dataJson: JSON.stringify({ gif: true }) })} />);
    const v = screen.getByTestId("gif-video") as HTMLVideoElement;
    expect(v.autoplay).toBe(true);
    expect(v.loop).toBe(true);
    expect(v.muted).toBe(true);
    expect(v.controls).toBe(false);
  });

  it("GIF recebido (rich.gif, do gifPlayback) também toca em loop", () => {
    render(<MessageMedia message={msg({ fromMe: false, dataJson: JSON.stringify({ rich: { gif: true } }) })} />);
    expect(screen.getByTestId("gif-video")).toBeInTheDocument();
  });

  it("vídeo comum continua com controles e sem loop", () => {
    const { container } = render(<MessageMedia message={msg({ dataJson: "{}" })} />);
    expect(screen.queryByTestId("gif-video")).not.toBeInTheDocument();
    const v = container.querySelector("video") as HTMLVideoElement;
    expect(v.controls).toBe(true);
    expect(v.loop).toBe(false);
  });

  it("isGif lê as duas marcas e ignora dado quebrado ou ausente", () => {
    expect(isGif(JSON.stringify({ gif: true }))).toBe(true);
    expect(isGif(JSON.stringify({ rich: { gif: true } }))).toBe(true);
    expect(isGif(JSON.stringify({ gif: false, rich: { round: true } }))).toBe(false);
    expect(isGif("não é json")).toBe(false);
    expect(isGif(undefined)).toBe(false);
  });
});

describe("GifTrimmedBadge", () => {
  const badge = (over: Partial<Message>) => render(<GifTrimmedBadge message={msg(over)} />);

  it("GIF nosso que foi cortado mostra o selo", () => {
    badge({ dataJson: JSON.stringify({ gif: true, gifTrimmed: true }) });
    expect(screen.getByTestId("gif-trimmed-badge")).toHaveTextContent("GIF cortado em 15 s");
    expect(hasGifTrimmedBadge(msg({ dataJson: JSON.stringify({ gifTrimmed: true }) }))).toBe(true);
  });

  it("GIF que coube inteiro, mensagem recebida ou apagada não mostram", () => {
    badge({ dataJson: JSON.stringify({ gif: true }) });
    expect(screen.queryByTestId("gif-trimmed-badge")).not.toBeInTheDocument();
    expect(hasGifTrimmedBadge(msg({ fromMe: false, dataJson: JSON.stringify({ gifTrimmed: true }) }))).toBe(false);
    expect(hasGifTrimmedBadge(msg({ isDeleted: true, dataJson: JSON.stringify({ gifTrimmed: true }) }))).toBe(false);
  });

  it("o selo reserva o rodapé da bolha (largura mínima e linha extra)", () => {
    expect(hasFooterBadge(msg({ dataJson: JSON.stringify({ gifTrimmed: true }) }))).toBe(true);
    expect(hasFooterBadge(msg({ dataJson: "{}" }))).toBe(false);
  });
});
