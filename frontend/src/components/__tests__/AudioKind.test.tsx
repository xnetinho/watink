import React from "react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";

const get = vi.fn();
const post = vi.fn();
vi.mock("../../services/api", () => ({ default: { get: (...a: unknown[]) => get(...a), post: (...a: unknown[]) => post(...a) } }));
vi.mock("react-toastify", () => ({ toast: { success: vi.fn(), warning: vi.fn(), error: vi.fn() } }));

import ForwardMessageDialog from "../ForwardMessageDialog";
import { AudioKindPicker } from "../AudioKindPicker";
import { isAudioFile } from "../MessageInput/hooks/useMediaUpload";
import { voiceOf } from "../../lib/messages/forward";

describe("AudioKindPicker", () => {
  it("mostra as duas formas em português e avisa a escolha", () => {
    const onChange = vi.fn();
    render(<AudioKindPicker value="file" onChange={onChange} />);
    expect(screen.getByText("Mensagem de voz")).toBeInTheDocument();
    expect(screen.getByText("Arquivo de áudio")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Mensagem de voz"));
    expect(onChange).toHaveBeenCalledWith("voice");
  });
  it("modo compacto esconde as explicações", () => {
    render(<AudioKindPicker value="file" onChange={() => undefined} compact />);
    expect(screen.queryByText(/microfone e onda/)).toBeNull();
  });
});

describe("isAudioFile / voiceOf", () => {
  it("reconhece áudio pelo tipo e pela extensão", () => {
    expect(isAudioFile({ type: "audio/mpeg", name: "a" })).toBe(true);
    expect(isAudioFile({ type: "", name: "voz.OPUS" })).toBe(true);
    expect(isAudioFile({ type: "", name: "a.m4a" })).toBe(true);
    expect(isAudioFile({ type: "application/pdf", name: "a.pdf" })).toBe(false);
    expect(isAudioFile({ type: "video/mp4", name: "a.mp4" })).toBe(false);
  });
  it("voz enviada pelo Watink (voice) e recebida do celular (rich.voice)", () => {
    expect(voiceOf({ dataJson: '{"voice":true}' })).toBe(true);
    expect(voiceOf({ dataJson: '{"rich":{"voice":true}}' })).toBe(true);
    expect(voiceOf({ dataJson: '{"mimetype":"audio/mpeg"}' })).toBe(false);
    expect(voiceOf({ dataJson: "lixo{" })).toBe(false);
    expect(voiceOf({})).toBe(false);
  });
});

const contacts = [{ id: 2, name: "Ana", number: "5511999990001", connectionSource: "whatsapp", connectionId: 1 }];

describe("encaminhar áudio: escolha de como enviar", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    get.mockImplementation((url: string) => Promise.resolve({ data: url === "/contacts" ? { contacts } : url === "/whatsapp" ? [{ id: 1, name: "Comercial", status: "CONNECTED" }] : [] }));
    post.mockResolvedValue({ data: { results: [{ contactId: 2, ok: true }] } });
  });

  const open = (props: Partial<React.ComponentProps<typeof ForwardMessageDialog>>) =>
    render(<ForwardMessageDialog open onClose={vi.fn()} messageId="m1" channel="whatsapp" {...props} />);

  it("áudio de arquivo: começa em 'Arquivo de áudio' e envia audioKind=file", async () => {
    open({ mediaType: "audio", sourceIsVoice: false });
    await screen.findByText("Ana");
    expect(screen.getByTestId("audio-kind-picker")).toBeInTheDocument();
    fireEvent.click(screen.getByLabelText("Ana"));
    fireEvent.click(screen.getByText("Enviar"));
    await waitFor(() => expect(post).toHaveBeenCalledWith("/message/m1/forward", { contactIds: [2], mode: "forward", audioKind: "file" }));
  });

  it("mensagem de voz: começa em 'Mensagem de voz', e o operador pode trocar para arquivo (nos dois modos)", async () => {
    open({ mediaType: "audio", sourceIsVoice: true });
    await screen.findByText("Ana");
    fireEvent.click(screen.getByLabelText("Ana"));
    fireEvent.click(screen.getByText("Reenviar como nova"));
    fireEvent.click(screen.getByText("Arquivo de áudio"));
    fireEvent.click(screen.getByText("Enviar"));
    await waitFor(() => expect(post).toHaveBeenCalledWith("/message/m1/forward", { contactIds: [2], mode: "copy", audioKind: "file" }));
  });

  it("outras mídias (imagem, texto) não mostram a escolha nem mandam audioKind", async () => {
    open({ mediaType: "image" });
    await screen.findByText("Ana");
    expect(screen.queryByTestId("audio-kind-picker")).toBeNull();
    fireEvent.click(screen.getByLabelText("Ana"));
    fireEvent.click(screen.getByText("Enviar"));
    await waitFor(() => expect(post).toHaveBeenCalledWith("/message/m1/forward", { contactIds: [2], mode: "forward" }));
  });
});
