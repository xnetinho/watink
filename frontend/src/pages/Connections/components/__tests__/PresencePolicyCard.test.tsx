import React from "react";
import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import PresencePolicyCard from "../PresencePolicyCard";
import { PRESENCE_POLICY_OPTIONS, clampIdleMinutes, optionFor, toPresencePolicy } from "../../presencePolicy";
import { i18n } from "../../../../translate/i18n";

const tr = (key: string, lng = "pt") => String(i18n.t(key, { lng }) ?? "");

const setup = (over: Partial<React.ComponentProps<typeof PresencePolicyCard>> = {}) => {
  const onChange = vi.fn().mockResolvedValue(undefined);
  render(<PresencePolicyCard policy="never" idleMinutes={5} saving={false} onChange={onChange} {...over} />);
  return onChange;
};

describe("política de presença: regras", () => {
  it("as 3 opções existem, cada uma com nome e resumo nos 3 idiomas (nada fixo no código)", () => {
    expect(PRESENCE_POLICY_OPTIONS.map((o) => o.value)).toEqual(["never", "on_demand", "always"]);
    for (const o of PRESENCE_POLICY_OPTIONS) {
      for (const lng of ["pt", "en", "es"]) {
        expect(tr(o.labelKey, lng), `${o.labelKey} em ${lng}`).not.toBe(o.labelKey);
        expect(tr(o.summaryKey, lng).length, `${o.summaryKey} em ${lng}`).toBeGreaterThan(40);
      }
    }
  });
  it("valor desconhecido vira 'never' (o mais seguro: não silencia o celular)", () => {
    for (const bad of [undefined, null, "", "ALWAYS", "sempre", 3]) expect(toPresencePolicy(bad)).toBe("never");
    expect(toPresencePolicy("always")).toBe("always");
  });
  it("tempo ocioso: limita a 1..240 e trata lixo", () => {
    expect([0, -5, 1, 30, 240, 999, NaN].map(clampIdleMinutes)).toEqual([1, 1, 1, 30, 240, 240, 5]);
  });
  it("os resumos avisam do efeito no celular nas políticas que o têm", () => {
    expect(tr(optionFor("never").summaryKey)).toMatch(/continua recebendo notificações/i);
    expect(tr(optionFor("on_demand").summaryKey)).toMatch(/notificações/i);
    expect(tr(optionFor("always").summaryKey)).toMatch(/não recebe notificações/i);
  });
});

describe("PresencePolicyCard", () => {
  it("mostra o resumo da política atual", () => {
    setup({ policy: "always" });
    expect(screen.getByTestId("presence-policy-summary")).toHaveTextContent(/sempre disponível/i);
  });
  it("o campo fechado mostra só o NOME da opção, não o resumo (que fica embaixo)", () => {
    setup({ policy: "on_demand" });
    const combo = screen.getByRole("combobox", { name: /política de presença/i });
    expect(combo).toHaveTextContent("Disponível durante o atendimento");
    expect(combo.textContent).not.toMatch(/volta ao normal/i);
  });
  it("política desconhecida da API mostra 'never'", () => {
    setup({ policy: "xyz" });
    expect(screen.getByTestId("presence-policy-summary")).toHaveTextContent(/continua recebendo notificações/i);
  });
  it("o campo de minutos só aparece em 'durante o atendimento'", () => {
    const { rerender } = render(<PresencePolicyCard policy="never" idleMinutes={5} saving={false} onChange={vi.fn()} />);
    expect(screen.queryByLabelText(/minutos sem operador/i)).toBeNull();
    rerender(<PresencePolicyCard policy="on_demand" idleMinutes={5} saving={false} onChange={vi.fn()} />);
    expect(screen.getByLabelText(/minutos sem operador/i)).toBeInTheDocument();
  });
  it("editar os minutos salva ao sair do campo, já limitado ao máximo", async () => {
    const onChange = setup({ policy: "on_demand", idleMinutes: 5 });
    const input = screen.getByLabelText(/minutos sem operador/i);
    fireEvent.change(input, { target: { value: "9999" } });
    fireEvent.blur(input);
    await waitFor(() => expect(onChange).toHaveBeenCalledWith("on_demand", 240));
  });
  it("não salva se os minutos não mudaram", () => {
    const onChange = setup({ policy: "on_demand", idleMinutes: 5 });
    fireEvent.blur(screen.getByLabelText(/minutos sem operador/i));
    expect(onChange).not.toHaveBeenCalled();
  });
  it("enquanto salva, o seletor fica desabilitado", () => {
    setup({ saving: true });
    expect(screen.getByRole("combobox", { name: /política de presença/i })).toBeDisabled();
  });
});
