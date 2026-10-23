import React from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import CallsSection from "../CallsSection";
import SettingsSideNav from "../SettingsSideNav";
import { i18n } from "@/translate/i18n";

const get = vi.fn();
const put = vi.fn();
vi.mock("@/services/api", () => ({ default: { get: (...a: unknown[]) => get(...a), put: (...a: unknown[]) => put(...a) } }));
vi.mock("../../../../services/api", () => ({ default: { get: (...a: unknown[]) => get(...a), put: (...a: unknown[]) => put(...a) } }));
const success = vi.fn();
const error = vi.fn();
vi.mock("@/lib/notify", () => ({ default: { success: (m: string) => success(m), error: (e: unknown) => error(e) } }));
void i18n.changeLanguage("pt");

const cfg = (over = {}) => ({ mode: "off", ackBy: null, ackAt: null, available: true, ...over });

beforeEach(() => {
  get.mockReset();
  put.mockReset();
  success.mockReset();
  error.mockReset();
});

async function ready(c = cfg()) {
  get.mockResolvedValue({ data: c });
  render(<CallsSection />);
  await screen.findByTestId("calls-settings");
}

describe("CallsSection", () => {
  it("começa carregando e depois mostra o modo atual", async () => {
    get.mockResolvedValue({ data: cfg({ mode: "optional", ackBy: 3, ackAt: "2026-10-06T12:00:00Z" }) });
    render(<CallsSection />);
    expect(screen.getByTestId("calls-settings-loading")).toBeInTheDocument();
    await screen.findByTestId("calls-settings");
    expect(screen.getByTestId("mode-optional")).toHaveAttribute("data-state", "checked");
    expect(screen.getByTestId("calls-ack-record")).toHaveTextContent("#3");
  });

  it("modo desconhecido vindo do servidor aparece como desligado", async () => {
    await ready(cfg({ mode: "qualquer-coisa" }));
    expect(screen.getByTestId("mode-off")).toHaveAttribute("data-state", "checked");
  });

  it("sem mudança, Salvar fica desabilitado e o termo não aparece", async () => {
    await ready();
    expect(screen.getByTestId("calls-save")).toBeDisabled();
    expect(screen.queryByTestId("calls-ack")).not.toBeInTheDocument();
  });

  it("escolher um modo ligado mostra o termo e mantém Salvar bloqueado até o aceite", async () => {
    await ready();
    fireEvent.click(screen.getByTestId("mode-auto"));
    expect(screen.getByTestId("calls-ack")).toHaveTextContent("Termo de responsabilidade");
    expect(screen.getByTestId("calls-ack")).toHaveTextContent("LGPD");
    expect(screen.getByTestId("calls-save")).toBeDisabled();
    fireEvent.click(screen.getByTestId("calls-ack-check"));
    expect(screen.getByTestId("calls-save")).toBeEnabled();
  });

  it("sem aceite a API NÃO é chamada", async () => {
    await ready();
    fireEvent.click(screen.getByTestId("mode-auto"));
    fireEvent.click(screen.getByTestId("calls-save"));
    expect(put).not.toHaveBeenCalled();
  });

  it("com aceite envia o modo e ack=true, e reflete o novo estado", async () => {
    await ready();
    put.mockResolvedValue({ data: cfg({ mode: "auto", ackBy: 7, ackAt: "2026-10-06T13:00:00Z" }) });
    fireEvent.click(screen.getByTestId("mode-auto"));
    fireEvent.click(screen.getByTestId("calls-ack-check"));
    fireEvent.click(screen.getByTestId("calls-save"));
    await waitFor(() => expect(put).toHaveBeenCalledWith("/calls/recording-config", { mode: "auto", ack: true }));
    await waitFor(() => expect(success).toHaveBeenCalledWith("Configuração salva."));
    expect(screen.getByTestId("mode-auto")).toHaveAttribute("data-state", "checked");
    expect(screen.getByTestId("calls-ack-record")).toHaveTextContent("#7");
    expect(screen.queryByTestId("calls-ack")).not.toBeInTheDocument();
    expect(screen.getByTestId("calls-save")).toBeDisabled();
  });

  it("trocar a escolha desfaz o aceite já marcado (precisa aceitar de novo)", async () => {
    await ready();
    fireEvent.click(screen.getByTestId("mode-auto"));
    fireEvent.click(screen.getByTestId("calls-ack-check"));
    fireEvent.click(screen.getByTestId("mode-optional"));
    expect(screen.getByTestId("calls-save")).toBeDisabled();
  });

  it("entre modos ligados não pede aceite e não envia ack", async () => {
    await ready(cfg({ mode: "optional", ackBy: 1, ackAt: "2026-10-06T12:00:00Z" }));
    put.mockResolvedValue({ data: cfg({ mode: "auto", ackBy: 1, ackAt: "2026-10-06T12:00:00Z" }) });
    fireEvent.click(screen.getByTestId("mode-auto"));
    expect(screen.queryByTestId("calls-ack")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("calls-save"));
    await waitFor(() => expect(put).toHaveBeenCalledWith("/calls/recording-config", { mode: "auto" }));
  });

  it("desligar não pede aceite", async () => {
    await ready(cfg({ mode: "auto", ackBy: 1, ackAt: "2026-10-06T12:00:00Z" }));
    put.mockResolvedValue({ data: cfg({ mode: "off" }) });
    fireEvent.click(screen.getByTestId("mode-off"));
    expect(screen.queryByTestId("calls-ack")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("calls-save"));
    await waitFor(() => expect(put).toHaveBeenCalledWith("/calls/recording-config", { mode: "off" }));
  });

  it("sem armazenamento (S3): avisa, e só 'desligada' fica disponível", async () => {
    await ready(cfg({ available: false }));
    expect(screen.getByTestId("calls-no-storage")).toHaveTextContent("armazenamento de objetos");
    expect(screen.getByTestId("mode-optional")).toBeDisabled();
    expect(screen.getByTestId("mode-auto")).toBeDisabled();
    expect(screen.getByTestId("mode-off")).toBeEnabled();
  });

  it("erro do servidor ao salvar avisa e mantém o estado anterior", async () => {
    await ready();
    put.mockRejectedValue(new Error("422"));
    fireEvent.click(screen.getByTestId("mode-auto"));
    fireEvent.click(screen.getByTestId("calls-ack-check"));
    fireEvent.click(screen.getByTestId("calls-save"));
    await waitFor(() => expect(error).toHaveBeenCalled());
    expect(success).not.toHaveBeenCalled();
  });
});

describe("SettingsSideNav — seção Chamadas", () => {
  const nav = (canManageCalls?: boolean) =>
    render(
      <MemoryRouter>
        <SettingsSideNav activeSection="general" activePlugins={[]} onSelect={vi.fn()} canManageCalls={canManageCalls} />
      </MemoryRouter>,
    );

  it("aparece só para quem tem calls:manage", () => {
    nav(true);
    expect(screen.getByRole("button", { name: /Chamadas/ })).toBeInTheDocument();
  });
  it("não aparece sem calls:manage", () => {
    nav(false);
    expect(screen.queryByRole("button", { name: /Chamadas/ })).not.toBeInTheDocument();
  });
  it("não aparece quando a permissão nem foi informada", () => {
    nav(undefined);
    expect(screen.queryByRole("button", { name: /Chamadas/ })).not.toBeInTheDocument();
  });
});
