import { useState } from "react";
import toastError from "../../../errors/toastError";
import api from "../../../services/api";

export interface MediaItem {
  file: File;
  caption: string;
  /** Só para áudio: como sai (mensagem de voz ou arquivo de áudio). Anexo começa como arquivo. */
  audioKind?: "voice" | "file";
}

/** O anexo é um áudio? Pelo mimetype, ou pela extensão quando o navegador não informa. */
export const isAudioFile = (f: { type: string; name: string }): boolean =>
  f.type.startsWith("audio/") || /\.(opus|ogg|oga|mp3|m4a|aac|wav|amr|weba)$/i.test(f.name);

export interface UseMediaUploadReturn {
  medias: MediaItem[];
  handleChangeMedias: (e: React.ChangeEvent<HTMLInputElement>, currentInputMessage: string, clearInput: () => void) => void;
  handleInputPaste: (e: React.ClipboardEvent<HTMLTextAreaElement>, currentInputMessage: string, clearInput: () => void) => void;
  handleMediaCaptionChange: (index: number, value: string) => void;
  handleAudioKindChange: (index: number, kind: "voice" | "file") => void;
  handleUploadMedia: (ticketId: string | undefined, e?: React.FormEvent) => Promise<void>;
  handleRemoveMedia: (index: number) => void;
  clearMedias: () => void;
}

export function useMediaUpload(setLoading: (v: boolean) => void): UseMediaUploadReturn {
  const [medias, setMedias] = useState<MediaItem[]>([]);

  const handleChangeMedias = (
    e: React.ChangeEvent<HTMLInputElement>,
    currentInputMessage: string,
    clearInput: () => void
  ) => {
    if (!e.target.files) return;
    const selectedMedias = Array.from(e.target.files).map((file, index) => {
      let caption = "";
      if (index === 0 && medias.length === 0 && currentInputMessage) {
        caption = currentInputMessage;
      }
      return { file, caption, ...(isAudioFile(file) ? { audioKind: "file" as const } : {}) };
    });

    if (medias.length === 0 && currentInputMessage) {
      clearInput();
    }
    setMedias((prev) => [...prev, ...selectedMedias]);
  };

  const handleInputPaste = (
    e: React.ClipboardEvent<HTMLTextAreaElement>,
    currentInputMessage: string,
    clearInput: () => void
  ) => {
    if (e.clipboardData.files[0]) {
      const caption = medias.length === 0 && currentInputMessage ? currentInputMessage : "";
      const file = e.clipboardData.files[0];
      setMedias((prev) => [...prev, { file, caption, ...(isAudioFile(file) ? { audioKind: "file" as const } : {}) }]);
      if (medias.length === 0 && currentInputMessage) {
        clearInput();
      }
    }
  };

  const handleMediaCaptionChange = (index: number, value: string) => {
    setMedias((prev) => prev.map((media, i) => (i === index ? { ...media, caption: value } : media)));
  };

  const handleAudioKindChange = (index: number, kind: "voice" | "file") => {
    setMedias((prev) => prev.map((media, i) => (i === index ? { ...media, audioKind: kind } : media)));
  };

  const handleUploadMedia = async (ticketId: string | undefined, e?: React.FormEvent) => {
    setLoading(true);
    if (e) e.preventDefault();

    // O backend lê UM arquivo por requisição: cada anexo vai no seu próprio envio, com o seu próprio tipo de áudio.
    const sent = new Set<MediaItem>();
    try {
      for (const media of medias) {
        const form = new FormData();
        form.append("fromMe", "true");
        form.append("medias", media.file);
        form.append("body", media.caption);
        if (media.audioKind) form.append("audioKind", media.audioKind);
        await api.post(`/messages/${ticketId}`, form);
        sent.add(media);
      }
      setMedias([]);
    } catch (err) {
      // O que já saiu não pode sair de novo na próxima tentativa: fica só o que falhou e o que não foi enviado.
      setMedias((prev) => prev.filter((m) => !sent.has(m)));
      toastError(err);
    }
    setLoading(false);
  };

  const handleRemoveMedia = (index: number) => {
    setMedias((prev) => prev.filter((_, i) => i !== index));
  };

  const clearMedias = () => setMedias([]);

  return {
    medias,
    handleChangeMedias,
    handleInputPaste,
    handleMediaCaptionChange,
    handleAudioKindChange,
    handleUploadMedia,
    handleRemoveMedia,
    clearMedias,
  };
}
