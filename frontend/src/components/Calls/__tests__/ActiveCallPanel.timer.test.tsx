import React from "react";
import { act, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import ActiveCallPanel from "../ActiveCallPanel";
import { baseCall, makeCtx, quality, withCalls } from "./helpers";

// A telemetria de qualidade chega ~1x/s e cria um objeto `active` novo a cada vez. O efeito do
// cronômetro dependia de `active` inteiro: cada telemetria derrubava o setInterval ainda não
// disparado e recomeçava a contagem. Com telemetria um pouco mais rápida que 1 s o relógio nunca
// atualizava, ficava congelado e depois pulava para o horário real.
describe("ActiveCallPanel — cronômetro", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it("continua contando mesmo com telemetria chegando antes de cada tick", () => {
    const start = Date.now();
    const call = (q = quality()) => baseCall({ phase: "active", connectedAt: start, quality: q });
    const { rerender } = render(withCalls(makeCtx({ active: call() }), <ActiveCallPanel />));
    expect(screen.getByTestId("call-status")).toHaveTextContent("00:00");

    for (let i = 0; i < 10; i++) {
      act(() => {
        vi.advanceTimersByTime(900);
      });
      // objeto novo a cada telemetria, como o reducer faz
      rerender(withCalls(makeCtx({ active: call(quality({ rttMs: 40 + i })) }), <ActiveCallPanel />));
    }

    // 9 s se passaram: o relógio tem de ter andado, não ficado em 00:00
    expect(screen.getByTestId("call-status")).not.toHaveTextContent("00:00");
    expect(screen.getByTestId("call-status")).toHaveTextContent(/00:0[5-9]/);
  });

  it("para de contar quando a chamada deixa de estar ativa", () => {
    const start = Date.now();
    const { rerender } = render(withCalls(makeCtx({ active: baseCall({ phase: "active", connectedAt: start }) }), <ActiveCallPanel />));
    rerender(withCalls(makeCtx({ active: baseCall({ phase: "ended", connectedAt: start }) }), <ActiveCallPanel />));
    expect(vi.getTimerCount()).toBe(0);
  });
});
