import { describe, expect, it, vi } from "vitest";
import { act, renderHook } from "@testing-library/react";

vi.mock("../../../../services/api", () => ({ default: { post: vi.fn() } }));
vi.mock("../../../../errors/toastError", () => ({ default: vi.fn() }));

import { useMediaUpload } from "../useMediaUpload";

const f = (name: string, type: string) => new File(["x"], name, { type });

describe("arquivos soltos (drag and drop)", () => {
  it("entram como os escolhidos pelo clipe, e o texto digitado vira a legenda do primeiro", () => {
    const clear = vi.fn();
    const { result } = renderHook(() => useMediaUpload(() => undefined));
    act(() => result.current.handleDropFiles([f("a.png", "image/png"), f("b.png", "image/png")], "olha isso", clear));
    expect(result.current.medias.map((m) => [m.file.name, m.caption])).toEqual([
      ["a.png", "olha isso"],
      ["b.png", ""],
    ]);
    expect(clear).toHaveBeenCalledTimes(1);
  });

  it("áudio solto começa como arquivo (não como voz), igual ao anexo pelo clipe", () => {
    const { result } = renderHook(() => useMediaUpload(() => undefined));
    act(() => result.current.handleDropFiles([f("nota.mp3", "audio/mpeg")], "", () => undefined));
    expect(result.current.medias[0].audioKind).toBe("file");
  });

  it("soltar mais arquivos depois acrescenta, e a legenda digitada não é reaproveitada", () => {
    const clear = vi.fn();
    const { result } = renderHook(() => useMediaUpload(() => undefined));
    act(() => result.current.handleDropFiles([f("a.png", "image/png")], "", clear));
    act(() => result.current.handleDropFiles([f("b.png", "image/png")], "texto novo", clear));
    expect(result.current.medias.map((m) => [m.file.name, m.caption])).toEqual([
      ["a.png", ""],
      ["b.png", ""],
    ]);
    expect(clear).not.toHaveBeenCalled();
  });

  it("lista vazia não faz nada", () => {
    const { result } = renderHook(() => useMediaUpload(() => undefined));
    act(() => result.current.handleDropFiles([], "x", () => undefined));
    expect(result.current.medias).toEqual([]);
  });
});
