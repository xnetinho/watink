import React from "react";
import { describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";

const get = vi.fn();
vi.mock("../../../../services/api", () => ({ default: { get: (...a: unknown[]) => get(...a) } }));

import MessageEditsDialog from "../MessageEditsDialog";

describe("histórico de edições: falha", () => {
  it("falha ao carregar avisa o operador em vez de ficar em branco", async () => {
    get.mockRejectedValue(new Error("500"));
    render(<MessageEditsDialog messageId="M1" onClose={() => undefined} />);
    await waitFor(() => expect(screen.getByText("Não foi possível carregar o histórico.")).toBeInTheDocument());
  });
});
