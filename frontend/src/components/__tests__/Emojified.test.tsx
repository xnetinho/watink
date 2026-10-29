import React from "react";
import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { Emojified } from "../Emojified";
import EmojiPicker from "../EmojiPicker";
import MarkdownWrapper from "../MarkdownWrapper";
import { i18n } from "../../translate/i18n";

describe("Emojified: o emoji vira imagem (conjunto Apple), o texto continua texto", () => {
  it("😢 vira <img> com o emoji no alt (copiar e leitor de tela continuam funcionando)", () => {
    const { container } = render(<Emojified>{"Sinto muito 😢"}</Emojified>);
    const img = screen.getByTestId("emoji-img");
    expect(img).toHaveAttribute("src", "/emoji/apple/1f622.webp");
    expect(img).toHaveAttribute("alt", "😢");
    expect(container.textContent).toContain("Sinto muito");
  });
  it("sem emoji, nenhuma imagem", () => {
    render(<Emojified>{"só texto"}</Emojified>);
    expect(screen.queryByTestId("emoji-img")).toBeNull();
  });
  it("vazio ou nulo não renderiza nada", () => {
    const { container } = render(<Emojified>{null}</Emojified>);
    expect(container).toBeEmptyDOMElement();
  });
});

describe("MarkdownWrapper: emoji dentro de mensagem formatada", () => {
  it("emoji no texto, em negrito e em itálico viram imagem sem quebrar a formatação", () => {
    const { container } = render(<MarkdownWrapper>{"oi 😢 *negrito 🙏* fim"}</MarkdownWrapper>);
    expect(screen.getAllByTestId("emoji-img").length).toBe(2);
    expect(container.querySelector("strong")).not.toBeNull();
    expect(container.textContent).toContain("negrito");
  });
  it("texto comum não vira imagem", () => {
    render(<MarkdownWrapper>{"olá, tudo bem?"}</MarkdownWrapper>);
    expect(screen.queryByTestId("emoji-img")).toBeNull();
  });
});

describe("EmojiPicker: o que se escolhe é o que aparece na bolha", () => {
  it("mostra emojis como imagem do MESMO conjunto e devolve o emoji ao clicar", () => {
    const onSelect = vi.fn();
    render(<EmojiPicker onSelect={onSelect} />);
    const grid = screen.getByTestId("emoji-grid");
    expect(grid.querySelectorAll("img").length).toBeGreaterThan(50);
    expect(grid.querySelector("img")!.getAttribute("src")).toMatch(/^\/emoji\/apple\/.+\.webp$/);
    fireEvent.click(screen.getByLabelText("grinning"));
    expect(onSelect).toHaveBeenCalledWith({ native: "😀" });
  });
  it("busca varre todas as categorias", () => {
    render(<EmojiPicker onSelect={() => undefined} />);
    fireEvent.change(screen.getByLabelText(i18n.t("messagesInput.emojiPicker.search") as string), { target: { value: "cry" } });
    expect(screen.getByLabelText("cry")).toBeInTheDocument();
  });
  it("busca sem resultado mostra a mensagem traduzida", () => {
    render(<EmojiPicker onSelect={() => undefined} />);
    fireEvent.change(screen.getByLabelText(i18n.t("messagesInput.emojiPicker.search") as string), { target: { value: "zzzzqqq" } });
    expect(screen.getByText("Nenhum emoji encontrado")).toBeInTheDocument();
  });
  it("trocar de categoria mostra outro conjunto", () => {
    render(<EmojiPicker onSelect={() => undefined} />);
    const before = screen.getByTestId("emoji-grid").textContent;
    fireEvent.click(screen.getByRole("tab", { name: "Bandeiras" }));
    expect(screen.getByRole("tab", { name: "Bandeiras" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("emoji-grid").querySelectorAll("img").length).toBeGreaterThan(50);
    expect(screen.getByTestId("emoji-grid").textContent).not.toBe(before);
  });
  it("nunca monta os ~1.900 emojis de uma vez (uma categoria por vez)", () => {
    render(<EmojiPicker onSelect={() => undefined} />);
    expect(screen.getByTestId("emoji-grid").querySelectorAll("img").length).toBeLessThan(450);
  });
});
