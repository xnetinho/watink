import React from "react";
import { describe, expect, it } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import CallQualityPanel from "../CallQualityPanel";
import { quality } from "./helpers";
import "./helpers";

describe("CallQualityPanel", () => {
  it("sem telemetria não renderiza", () => {
    const { container } = render(<CallQualityPanel quality={null} />);
    expect(container).toBeEmptyDOMElement();
  });

  it.each([
    [3, "Boa"],
    [2, "Regular"],
    [1, "Ruim"],
  ] as const)("nível %s mostra '%s' e o indicador de sinal correspondente", (level, label) => {
    render(<CallQualityPanel quality={quality({ level })} />);
    expect(screen.getByText(label)).toBeInTheDocument();
    expect(screen.getByRole("img", { name: label })).toHaveAttribute("data-level", String(level));
  });

  it("nível bom e regular não disparam alerta; ruim dispara", () => {
    const { rerender } = render(<CallQualityPanel quality={quality({ level: 3 })} />);
    expect(screen.queryByTestId("call-quality-alert")).not.toBeInTheDocument();
    rerender(<CallQualityPanel quality={quality({ level: 2 })} />);
    expect(screen.queryByTestId("call-quality-alert")).not.toBeInTheDocument();
    rerender(<CallQualityPanel quality={quality({ level: 1, alerts: ["loss"] })} />);
    expect(screen.getByTestId("call-quality-alert")).toHaveTextContent("A qualidade da chamada está baixa.");
  });

  it("contato parou de enviar áudio gera o aviso próprio mesmo com nível bom", () => {
    render(<CallQualityPanel quality={quality({ level: 3, noPeerAudio: true, silentMs: 6000 })} />);
    expect(screen.getByTestId("call-quality-alert")).toHaveTextContent("O contato parou de enviar áudio.");
  });

  it("o painel de detalhes começa fechado e abre ao clicar", () => {
    render(<CallQualityPanel quality={quality()} />);
    expect(screen.queryByTestId("call-quality-details")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { expanded: false }));
    expect(screen.getByTestId("call-quality-details")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { expanded: true }));
    expect(screen.queryByTestId("call-quality-details")).not.toBeInTheDocument();
  });

  it("detalhes: latência, perda, jitter, taxa e níveis, atualizados pela telemetria", () => {
    const { rerender } = render(<CallQualityPanel quality={quality({ rttMs: 42, lossPct: 1.5, jitterMs: 12 })} />);
    fireEvent.click(screen.getByRole("button"));
    const d = screen.getByTestId("call-quality-details");
    expect(d).toHaveTextContent("42 ms");
    expect(d).toHaveTextContent("1,5%");
    expect(d).toHaveTextContent("12 ms");
    expect(d).toHaveTextContent("12,0 kbps");
    expect(d).toHaveTextContent("12,8 kbps");
    rerender(<CallQualityPanel quality={quality({ rttMs: 300, lossPct: 8, jitterMs: 70, level: 1 })} />);
    expect(screen.getByTestId("call-quality-details")).toHaveTextContent("300 ms");
    expect(screen.getByTestId("call-quality-details")).toHaveTextContent("8,0%");
  });

  it("o índice é rotulado como ESTIMADO e há a nota de que a perda só vale para o áudio recebido", () => {
    render(<CallQualityPanel quality={quality({ mosEstimated: 4.3 })} />);
    fireEvent.click(screen.getByRole("button"));
    const d = screen.getByTestId("call-quality-details");
    expect(d).toHaveTextContent("Índice de qualidade (estimado)");
    expect(d).toHaveTextContent("4,3");
    expect(d).toHaveTextContent("não é a qualidade percebida pelo contato");
    expect(d).toHaveTextContent("A perda no envio ao contato não é medida");
  });

  it("RTT não medido aparece como 'não medido', nunca como zero", () => {
    render(<CallQualityPanel quality={quality({ rttMs: null })} />);
    fireEvent.click(screen.getByRole("button"));
    expect(screen.getByTestId("call-quality-details")).toHaveTextContent("não medido");
  });

  it("soma o atraso do buffer do navegador à latência exibida", () => {
    render(<CallQualityPanel quality={quality({ rttMs: 40 })} extraDelayMs={60} />);
    fireEvent.click(screen.getByRole("button"));
    expect(screen.getByTestId("call-quality-details")).toHaveTextContent("100 ms");
  });

  it("índice ausente mostra traço", () => {
    render(<CallQualityPanel quality={quality({ mosEstimated: null })} />);
    fireEvent.click(screen.getByRole("button"));
    expect(screen.getByTestId("call-quality-details")).toHaveTextContent("—");
  });
});
