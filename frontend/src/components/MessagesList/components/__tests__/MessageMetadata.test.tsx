import React from "react";
import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import MessageMetadata from "../MessageMetadata";
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
