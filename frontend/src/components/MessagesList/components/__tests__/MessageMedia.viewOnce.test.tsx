import React from "react";
import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import MessageMedia from "../MessageMedia";
import ViewOnceBadge from "../ViewOnceBadge";
import { hasMediaBubble } from "../MessageBubble";
import type { Message } from "../../types";
import "../../../Calls/__tests__/helpers";

vi.mock("@/services/api", () => ({ default: { get: vi.fn(), post: vi.fn() } }));
vi.mock("@/errors/toastError", () => ({ default: vi.fn() }));

const msg = (over: Partial<Message> = {}): Message =>
  ({ id: "VO1", body: "", fromMe: false, mediaType: "view_once", createdAt: new Date().toISOString(), ...over }) as Message;

// O WhatsApp não entrega a visualização única a aparelhos vinculados: chega só um aviso, sem conteúdo.
// O chat precisa mostrar que o cliente mandou algo e orientar a ver no celular, SEM oferecer um download
// que nunca funcionaria.
describe("MessageMedia — visualização única", () => {
  it("tem bolha própria, mesmo sem mediaUrl", () => {
    expect(hasMediaBubble(msg())).toBe(true);
  });

  it("mostra o aviso e orienta a ver no celular", () => {
    render(<MessageMedia message={msg()} />);
    const box = screen.getByTestId("view-once-notice");
    expect(box).toHaveTextContent("Visualização única");
    expect(box).toHaveTextContent(/celular/i);
    expect(box).toHaveTextContent(/responda/i);
  });

  it("NÃO oferece download nem miniatura desfocada", () => {
    render(<MessageMedia message={msg()} />);
    expect(screen.queryByRole("button", { name: /baixar|download/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("img")).not.toBeInTheDocument();
  });

  it("uma imagem comum continua sem o aviso", () => {
    render(<MessageMedia message={msg({ mediaType: "image", mediaUrl: "/x.jpg" })} />);
    expect(screen.queryByTestId("view-once-notice")).not.toBeInTheDocument();
  });
});

// Depois que o operador responde no celular, o servidor troca o aviso pela foto recuperada: o chat passa a desenhá-la como
// mídia comum (sem aviso, sem botão especial, com o arquivo guardado).
describe("visualização única recuperada", () => {
  it("vira imagem comum: o aviso some e a foto aparece", () => {
    const m = msg({ mediaType: "image", mediaUrl: "/public/media/abc.jpg", dataJson: JSON.stringify({ recoveredViewOnce: true, mediaStatus: "downloaded", mimetype: "image/jpeg" }) });
    expect(hasMediaBubble(m)).toBe(true);
    render(<MessageMedia message={m} />);
    expect(screen.queryByTestId("view-once-notice")).toBeNull();
  });
});

describe("selo de visualização única no rodapé", () => {
  it("aparece só na mídia recuperada", () => {
    const rec = msg({ mediaType: "image", mediaUrl: "/public/media/a.jpg", dataJson: JSON.stringify({ recoveredViewOnce: true }) });
    const { unmount } = render(<ViewOnceBadge message={rec} />);
    expect(screen.getByTestId("view-once-badge")).toHaveTextContent("Visualização única");
    unmount();
    for (const m of [msg({ mediaType: "image", dataJson: "{}" }), msg({ mediaType: "image" }), { ...rec, isDeleted: true }]) {
      const r = render(<ViewOnceBadge message={m as Message} />);
      expect(r.container.querySelector('[data-testid="view-once-badge"]')).toBeNull();
      r.unmount();
    }
  });
});
