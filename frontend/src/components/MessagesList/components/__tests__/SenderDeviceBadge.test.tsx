import React from "react";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import SenderDeviceBadge, { senderDeviceOf } from "../SenderDeviceBadge";
import { hasFooterBadge } from "../MessageBubble";
import type { Message } from "../../types";
import { i18n } from "../../../../translate/i18n";

void i18n.changeLanguage("pt");
const msg = (over: Partial<Message> = {}) => ({ id: "ACF893AD9DC7F9C71FA2C47EA45818AF", body: "Oi", fromMe: false, createdAt: "2026-11-07T16:50:00Z", ...over }) as Message;

describe("selo do aparelho do contato", () => {
  it("mostra o aparelho com o til de estimativa, em cada idioma e para cada tipo", () => {
    render(<SenderDeviceBadge message={msg()} channel="whatsapp" />);
    expect(screen.getByTestId("sender-device")).toHaveTextContent("~Android");
  });

  it.each([
    ["3AF2FA95D11B72616BF0", "~iPhone"],
    ["3EB0908DBE993FFAB74472", "~Web"],
  ])("%s mostra %s", (id, text) => {
    render(<SenderDeviceBadge message={msg({ id })} />);
    expect(screen.getByTestId("sender-device")).toHaveTextContent(text);
  });

  it("não aparece em mensagem nossa, apagada, registro de chamada, outro canal ou id desconhecido", () => {
    for (const [m, ch] of [
      [msg({ fromMe: true }), "whatsapp"],
      [msg({ isDeleted: true }), "whatsapp"],
      [msg({ mediaType: "call" }), "whatsapp"],
      [msg(), "telegram"],
      [msg({ id: "xyz" }), "whatsapp"],
    ] as [Message, string][]) {
      const { container, unmount } = render(<SenderDeviceBadge message={m} channel={ch} />);
      expect(container.firstChild).toBeNull();
      unmount();
    }
  });

  it("o balão ganha a linha do rodapé só quando há selo (e só quando o aparelho é conhecido)", () => {
    expect(hasFooterBadge(msg(), "whatsapp")).toBe(true);
    expect(hasFooterBadge(msg({ id: "xyz" }), "whatsapp")).toBe(false);
    expect(senderDeviceOf(msg(), undefined)).toBe("android");
  });
});
