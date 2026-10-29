import { describe, expect, it } from "vitest";
import { dragHasFiles, filesFromDrop } from "../dropFiles";

const file = (name: string, type = "image/png", size = 10) => new File([new Uint8Array(size)], name, { type });
const item = (f: File | null, kind = "file", dir = false) =>
  ({ kind, getAsFile: () => f, webkitGetAsEntry: () => ({ isDirectory: dir }) }) as unknown as DataTransferItem;

describe("dragHasFiles", () => {
  it("só arraste com arquivos conta; texto selecionado e link não", () => {
    expect(dragHasFiles({ types: ["Files"] })).toBe(true);
    expect(dragHasFiles({ types: ["text/plain", "Files"] })).toBe(true);
    expect(dragHasFiles({ types: ["text/plain"] })).toBe(false);
    expect(dragHasFiles({ types: ["text/uri-list"] })).toBe(false);
    expect(dragHasFiles(null)).toBe(false);
    expect(dragHasFiles(undefined)).toBe(false);
  });
});

describe("filesFromDrop", () => {
  it("devolve os arquivos soltos, na ordem", () => {
    const a = file("a.png"), b = file("b.pdf", "application/pdf");
    const out = filesFromDrop({ items: [item(a), item(b)] as unknown as DataTransferItemList, files: [] as unknown as FileList });
    expect(out.map((f) => f.name)).toEqual(["a.png", "b.pdf"]);
  });

  it("ignora texto/links e pastas (o envio lê um arquivo por vez)", () => {
    const a = file("a.png");
    const out = filesFromDrop({
      items: [item(null, "string"), item(file("pasta"), "file", true), item(a)] as unknown as DataTransferItemList,
      files: [] as unknown as FileList,
    });
    expect(out.map((f) => f.name)).toEqual(["a.png"]);
  });

  it("sem items (alguns navegadores) cai em files", () => {
    const out = filesFromDrop({ items: undefined as unknown as DataTransferItemList, files: [file("x.mp3", "audio/mpeg")] as unknown as FileList });
    expect(out.map((f) => f.name)).toEqual(["x.mp3"]);
  });

  it("entrada nula não quebra", () => {
    expect(filesFromDrop(null)).toEqual([]);
  });
});
