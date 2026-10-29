import React from "react";
import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";

vi.mock("../../services/api", () => ({
  default: {
    get: vi.fn((url: string) => {
      if (url === "/whatsapp") return Promise.resolve({ data: [{ id: 1, name: "Comercial", status: "CONNECTED" }, { id: 2, name: "Suporte", status: "CONNECTED" }] });
      if (url === "/channels/connections") return Promise.resolve({ data: [{ id: 1, kind: "telegram", name: "Jarvis", status: "CONNECTED" }] });
      return Promise.resolve({ data: {} });
    }),
    post: vi.fn(),
    put: vi.fn(),
  },
}));

import ContactModal from "../ContactModal";

// jsdom não implementa estes métodos de ponteiro/rolagem que o Radix Select usa.
Object.assign(HTMLElement.prototype, { hasPointerCapture: () => false, setPointerCapture: () => undefined, releasePointerCapture: () => undefined, scrollIntoView: () => undefined });

describe("ContactModal: conexão do contato", () => {
  it("só oferece conexões de canais em que a empresa pode cadastrar contato (Telegram fica de fora)", async () => {
    render(<ContactModal open onClose={() => undefined} />);
    fireEvent.pointerDown(await screen.findByRole("combobox"), { button: 0, pointerType: "mouse" });
    await waitFor(() => expect(screen.getAllByRole("option").length).toBeGreaterThan(0));
    const names = screen.getAllByRole("option").map((o) => o.textContent);
    expect(names.join("|")).toContain("Comercial");
    expect(names.join("|")).toContain("Suporte");
    expect(names.join("|")).not.toContain("Jarvis");
  });
});
