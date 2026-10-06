import React from "react";
import { describe, expect, it } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import PauseCallsButton from "../PauseCallsButton";
import { makeCtx, withCalls } from "./helpers";

describe("PauseCallsButton", () => {
  it("só aparece para quem pode receber chamadas", () => {
    const { container } = render(withCalls(makeCtx({ canReceive: false }), <PauseCallsButton />));
    expect(container.querySelector('[data-testid="pause-calls"]')).toBeNull();
  });

  it("aparece para quem recebe e oferece pausar", () => {
    render(withCalls(makeCtx({ canReceive: true, paused: false }), <PauseCallsButton />));
    const b = screen.getByTestId("pause-calls");
    expect(b).toHaveAttribute("aria-pressed", "false");
    expect(b).toHaveAttribute("aria-label", "Pausar chamadas");
  });

  it("clicar pausa; pausado oferece retomar", () => {
    const ctx = makeCtx({ canReceive: true, paused: false });
    const { rerender } = render(withCalls(ctx, <PauseCallsButton />));
    fireEvent.click(screen.getByTestId("pause-calls"));
    expect(ctx.setPaused).toHaveBeenCalledWith(true);

    const paused = makeCtx({ canReceive: true, paused: true });
    rerender(withCalls(paused, <PauseCallsButton />));
    expect(screen.getByTestId("pause-calls")).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByTestId("pause-calls")).toHaveAttribute("aria-label", "Retomar chamadas");
    fireEvent.click(screen.getByTestId("pause-calls"));
    expect(paused.setPaused).toHaveBeenCalledWith(false);
  });
});
