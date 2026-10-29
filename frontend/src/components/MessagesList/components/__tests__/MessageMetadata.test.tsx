import React from "react";
import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import MessageMetadata, { MessageAck } from "../MessageMetadata";
import type { Message } from "../../types";

const base = { id: "m1", body: "x", fromMe: true, ack: 0, createdAt: "2026-10-06T16:42:04.000Z" } as unknown as Message;

describe("MessageMetadata", () => {
  it("mensagem enviada aguardando entrega mostra o relógio", () => {
    const { container } = render(<MessageMetadata message={base} />);
    expect(container.querySelector("svg.lucide-clock")).not.toBeNull();
  });

  it("registro de chamada não mostra relógio nem ticks (é mensagem de sistema)", () => {
    const { container } = render(<MessageMetadata message={{ ...base, mediaType: "call" }} />);
    expect(container.querySelector("svg")).toBeNull();
  });
});

// O token --action-primary é HSL cru ("217 89% 76%"). `text-[var(--action-primary)]` gera `color: var(...)` = cor
// inválida, e o tique "lido" ficava cinza. A classe precisa envolver com hsl().
describe("tique de lido", () => {
  const readMsg = (ack: number) => ({ id: "m", body: "x", fromMe: true, ack, createdAt: new Date().toISOString() }) as Message;
  it("ack 3 e 4 usam a cor primária dentro de hsl(var(...))", () => {
    for (const ack of [3, 4]) {
      const { container, unmount } = render(<MessageAck message={readMsg(ack)} />);
      expect(container.querySelector("svg")?.getAttribute("class")).toContain("text-[hsl(var(--action-primary))]");
      unmount();
    }
  });
  it("ack 2 (entregue) NÃO é azul", () => {
    const { container } = render(<MessageAck message={readMsg(2)} />);
    expect(container.querySelector("svg")?.getAttribute("class")).not.toContain("action-primary");
  });
});
