import React from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import WAVersionSection from "../WAVersionSection";
import SettingsSideNav from "../SettingsSideNav";
import { MemoryRouter } from "react-router";
import { i18n } from "@/translate/i18n";

const get = vi.fn();
const put = vi.fn();
vi.mock("../../../../services/api", () => ({ default: { get: (...a: unknown[]) => get(...a), put: (...a: unknown[]) => put(...a) } }));
const toastSuccess = vi.fn();
const toastWarning = vi.fn();
vi.mock("react-toastify", () => ({ toast: { success: (m: string) => toastSuccess(m), warning: (m: string) => toastWarning(m), error: vi.fn() } }));
vi.mock("../../../../errors/toastError", () => ({ default: vi.fn() }));
void i18n.changeLanguage("pt");

// O RadioGroup do Radix mede o controle com ResizeObserver, que o jsdom não tem.
vi.stubGlobal(
  "ResizeObserver",
  class {
    observe() {}
    unobserve() {}
    disconnect() {}
  },
);

const engine = (over = {}) => ({
  mode: "auto",
  embedded: "2.3000.1052556849",
  official: { version: "2.3000.1052855845" },
  registry: { version: "2.3000.1052828872" },
  effective: "2.3000.1052855845",
  source: "oficial",
  ...over,
});
const resp = (over = {}) => ({ data: { mode: "auto", manualVersion: null, engine: engine(), ...over } });

beforeEach(() => {
  get.mockReset();
  put.mockReset();
  toastSuccess.mockReset();
  toastWarning.mockReset();
});

describe("WAVersionSection", () => {
  it("mostra as quatro versões lado a lado", async () => {
    get.mockResolvedValue(resp());
    render(<WAVersionSection />);
    expect(await screen.findByTestId("wa-version-effective")).toHaveTextContent("2.3000.1052855845");
    expect(screen.getByTestId("wa-version-official")).toHaveTextContent("2.3000.1052855845");
    expect(screen.getByTestId("wa-version-registry")).toHaveTextContent("2.3000.1052828872");
    expect(screen.getByTestId("wa-version-embedded")).toHaveTextContent("2.3000.1052556849");
  });

  it("fonte indisponível mostra o motivo no lugar do número", async () => {
    get.mockResolvedValue(resp({ engine: engine({ registry: { error: "sem rede" } }) }));
    render(<WAVersionSection />);
    expect(await screen.findByTestId("wa-version-registry")).toHaveTextContent("sem rede");
  });

  it("motor fora do ar avisa, mas mostra a escolha salva", async () => {
    get.mockResolvedValue({ data: { mode: "manual", manualVersion: "2.3000.5", engineError: "engine_unavailable" } });
    render(<WAVersionSection />);
    expect(await screen.findByTestId("wa-version-engine-error")).toBeInTheDocument();
    expect(screen.getByRole("textbox")).toHaveValue("2.3000.5");
  });

  it("manual: valida o formato e só então envia", async () => {
    get.mockResolvedValue(resp());
    put.mockResolvedValue(resp({ mode: "manual", manualVersion: "2.3000.1052747707", engineApplied: true }));
    render(<WAVersionSection />);
    await screen.findByTestId("wa-version-effective");
    fireEvent.click(screen.getByLabelText(/Manual/));
    const input = await screen.findByRole("textbox");
    fireEvent.change(input, { target: { value: "lixo" } });
    expect(screen.getByTestId("wa-version-invalid")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Salvar" })).toBeDisabled();

    fireEvent.change(input, { target: { value: " 2.3000.1052747707 " } });
    expect(screen.queryByTestId("wa-version-invalid")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Salvar" }));
    await waitFor(() => expect(put).toHaveBeenCalledWith("/system/wa-version", { mode: "manual", manualVersion: "2.3000.1052747707" }));
    await waitFor(() => expect(toastSuccess).toHaveBeenCalled());
  });

  it("automático: não envia número e avisa quando o motor não aplicou", async () => {
    get.mockResolvedValue(resp({ mode: "manual", manualVersion: "2.3000.5" }));
    put.mockResolvedValue(resp({ engineApplied: false }));
    render(<WAVersionSection />);
    await screen.findByTestId("wa-version-effective");
    fireEvent.click(screen.getByLabelText(/Automático/));
    fireEvent.click(screen.getByRole("button", { name: "Salvar" }));
    await waitFor(() => expect(put).toHaveBeenCalledWith("/system/wa-version", { mode: "auto", manualVersion: "" }));
    await waitFor(() => expect(toastWarning).toHaveBeenCalled());
    expect(toastSuccess).not.toHaveBeenCalled();
  });
});

describe("SettingsSideNav · versão do WhatsApp", () => {
  const nav = (isSuperAdmin: boolean) =>
    render(
      <MemoryRouter>
        <SettingsSideNav activeSection="general" activePlugins={[]} onSelect={() => undefined} isSuperAdmin={isSuperAdmin} />
      </MemoryRouter>,
    );

  it("só o superadmin vê o item", () => {
    nav(true);
    expect(screen.getByRole("button", { name: "Versão do WhatsApp" })).toBeInTheDocument();
  });

  it("quem não é superadmin não vê", () => {
    nav(false);
    expect(screen.queryByRole("button", { name: "Versão do WhatsApp" })).toBeNull();
  });
});
