import { describe, expect, it, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

const post = vi.fn();
vi.mock("../../../../services/api", () => ({ default: { post: (...a: unknown[]) => post(...a) } }));
const toastError = vi.fn();
vi.mock("../../../../errors/toastError", () => ({ default: (...a: unknown[]) => toastError(...a) }));

import { useMediaUpload } from "../useMediaUpload";

const files = (...fs: File[]) => ({ target: { files: fs } }) as unknown as React.ChangeEvent<HTMLInputElement>;
const f = (name: string, type: string) => new File(["x"], name, { type });

beforeEach(() => {
  post.mockReset().mockResolvedValue({});
  toastError.mockReset();
});

describe("anexos: tipo do áudio", () => {
  it("áudio anexado começa como ARQUIVO; imagem e PDF não têm escolha", () => {
    const { result } = renderHook(() => useMediaUpload(vi.fn()));
    act(() => result.current.handleChangeMedias(files(f("a.mp3", "audio/mpeg"), f("b.png", "image/png"), f("c.pdf", "application/pdf"), f("d.opus", "")), "", vi.fn()));
    expect(result.current.medias.map((m) => m.audioKind)).toEqual(["file", undefined, undefined, "file"]);
  });

  it("cada anexo vai na SUA requisição, com a escolha de cada áudio", async () => {
    const { result } = renderHook(() => useMediaUpload(vi.fn()));
    act(() => result.current.handleChangeMedias(files(f("a.mp3", "audio/mpeg"), f("b.png", "image/png"), f("c.m4a", "audio/mp4")), "", vi.fn()));
    act(() => result.current.handleAudioKindChange(2, "voice"));
    await act(async () => result.current.handleUploadMedia("9"));

    expect(post).toHaveBeenCalledTimes(3);
    const sent = post.mock.calls.map(([url, form]) => ({ url, name: (form.get("medias") as File).name, kind: form.get("audioKind"), n: form.getAll("medias").length }));
    expect(sent).toEqual([
      { url: "/messages/9", name: "a.mp3", kind: "file", n: 1 },
      { url: "/messages/9", name: "b.png", kind: null, n: 1 },
      { url: "/messages/9", name: "c.m4a", kind: "voice", n: 1 },
    ]);
    expect(result.current.medias).toHaveLength(0);
  });

  it("legenda de cada anexo vai junto do seu próprio arquivo", async () => {
    const { result } = renderHook(() => useMediaUpload(vi.fn()));
    act(() => result.current.handleChangeMedias(files(f("a.png", "image/png"), f("b.png", "image/png")), "", vi.fn()));
    act(() => result.current.handleMediaCaptionChange(1, "segunda"));
    await act(async () => result.current.handleUploadMedia("9"));
    expect(post.mock.calls.map(([, form]) => form.get("body"))).toEqual(["", "segunda"]);
  });

  it("falha no meio avisa e mantém SÓ o que não saiu (o enviado não pode duplicar)", async () => {
    post.mockResolvedValueOnce({}).mockRejectedValueOnce(new Error("rede"));
    const { result } = renderHook(() => useMediaUpload(vi.fn()));
    act(() => result.current.handleChangeMedias(files(f("a.png", "image/png"), f("b.png", "image/png")), "", vi.fn()));
    await act(async () => result.current.handleUploadMedia("9"));
    expect(toastError).toHaveBeenCalledTimes(1);
    expect(result.current.medias.map((m) => m.file.name)).toEqual(["b.png"]);
  });
});

describe("anexos: visualização única", () => {
  it("o olho de cada mídia vai na SUA requisição; sem o olho, nada é enviado", async () => {
    const { result } = renderHook(() => useMediaUpload(vi.fn()));
    act(() => result.current.handleChangeMedias(files(f("a.png", "image/png"), f("b.png", "image/png"), f("c.mp4", "video/mp4")), "", vi.fn()));
    act(() => result.current.handleViewOnceChange(0, true));
    act(() => result.current.handleViewOnceChange(2, true));
    await act(async () => result.current.handleUploadMedia("9"));
    expect(post.mock.calls.map(([, form]) => form.get("viewOnce"))).toEqual(["true", null, "true"]);
  });

  it("desligar o olho tira o campo", async () => {
    const { result } = renderHook(() => useMediaUpload(vi.fn()));
    act(() => result.current.handleChangeMedias(files(f("a.png", "image/png")), "", vi.fn()));
    act(() => result.current.handleViewOnceChange(0, true));
    act(() => result.current.handleViewOnceChange(0, false));
    await act(async () => result.current.handleUploadMedia("9"));
    expect(post.mock.calls[0][1].get("viewOnce")).toBeNull();
  });

  it("documento nunca leva o campo, mesmo se alguém ligar o olho (não existe visualização única de PDF)", async () => {
    const { result } = renderHook(() => useMediaUpload(vi.fn()));
    act(() => result.current.handleChangeMedias(files(f("c.pdf", "application/pdf")), "", vi.fn()));
    act(() => result.current.handleViewOnceChange(0, true));
    await act(async () => result.current.handleUploadMedia("9"));
    expect(post.mock.calls[0][1].get("viewOnce")).toBeNull();
  });

  it("áudio aceita (mensagem de voz ou arquivo)", async () => {
    const { result } = renderHook(() => useMediaUpload(vi.fn()));
    act(() => result.current.handleChangeMedias(files(f("a.mp3", "audio/mpeg")), "", vi.fn()));
    act(() => result.current.handleViewOnceChange(0, true));
    await act(async () => result.current.handleUploadMedia("9"));
    expect(post.mock.calls[0][1].get("viewOnce")).toBe("true");
  });
});
