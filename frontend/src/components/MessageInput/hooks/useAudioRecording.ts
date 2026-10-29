import { useRef, useState } from "react";
import toastError from "../../../errors/toastError";
import api from "../../../services/api";
import { i18n } from "../../../translate/i18n";

// Gravação com o MediaRecorder do próprio navegador (sem biblioteca): o antigo mic-recorder-to-mp3 dependia do `lamejs`,
// que referencia a classe global `Lame` e quebrava no build ("Lame is not defined"). O engine converte o que chegar
// (webm/Opus no Chrome, Firefox e Edge; mp4/AAC no Safari) para OGG/Opus e envia como MENSAGEM DE VOZ.
const CANDIDATE_TYPES = ["audio/webm;codecs=opus", "audio/ogg;codecs=opus", "audio/mp4;codecs=mp4a.40.2", "audio/webm", "audio/mp4"];

/** Primeiro formato que este navegador sabe gravar; "" deixa o navegador escolher. */
export const pickRecordingType = (supports: (t: string) => boolean = (t) => MediaRecorder.isTypeSupported(t)): string =>
  CANDIDATE_TYPES.find((t) => supports(t)) ?? "";

/** Extensão do arquivo pelo mimetype gravado. */
export const extensionFor = (mime: string): string => {
  const m = mime.toLowerCase();
  if (m.includes("webm")) return "webm";
  if (m.includes("ogg")) return "ogg";
  if (m.includes("mp4") || m.includes("aac")) return "m4a";
  return "audio";
};

// Abaixo disso é um clique sem querer (menos de ~0,5 s), não uma mensagem.
const MIN_RECORDING_MS = 500;

export interface UseAudioRecordingReturn {
  recording: boolean;
  handleStartRecording: () => Promise<void>;
  handleUploadAudio: () => Promise<void>;
  handleCancelAudio: () => Promise<void>;
}

export function useAudioRecording(
  ticketId: string | undefined,
  setLoading: (v: boolean) => void
): UseAudioRecordingReturn {
  const [recording, setRecording] = useState(false);
  const recorderRef = useRef<MediaRecorder | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const chunksRef = useRef<Blob[]>([]);
  const startedAtRef = useRef(0);

  const releaseMic = () => {
    streamRef.current?.getTracks().forEach((t) => t.stop());
    streamRef.current = null;
    recorderRef.current = null;
    chunksRef.current = [];
  };

  /** Para a gravação e devolve o áudio inteiro (o MediaRecorder entrega os pedaços só ao parar). */
  const stopAndCollect = (): Promise<Blob | null> =>
    new Promise((resolve) => {
      const rec = recorderRef.current;
      if (!rec || rec.state === "inactive") return resolve(null);
      rec.onstop = () => resolve(new Blob(chunksRef.current, { type: rec.mimeType || "audio/webm" }));
      rec.stop();
    });

  const handleStartRecording = async () => {
    setLoading(true);
    try {
      if (!navigator.mediaDevices?.getUserMedia || typeof MediaRecorder === "undefined") {
        throw new Error(String(i18n.t("messagesInput.recording.unsupported") ?? "Recording not supported"));
      }
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      streamRef.current = stream;
      chunksRef.current = [];
      const type = pickRecordingType();
      const rec = new MediaRecorder(stream, type ? { mimeType: type } : undefined);
      rec.ondataavailable = (e) => {
        if (e.data && e.data.size > 0) chunksRef.current.push(e.data);
      };
      recorderRef.current = rec;
      rec.start();
      startedAtRef.current = Date.now();
      setRecording(true);
    } catch (err) {
      releaseMic();
      toastError(err);
    }
    setLoading(false);
  };

  const handleUploadAudio = async () => {
    setLoading(true);
    try {
      const tooShort = Date.now() - startedAtRef.current < MIN_RECORDING_MS;
      const blob = await stopAndCollect();
      releaseMic();
      setRecording(false);
      if (!blob || blob.size === 0 || tooShort) {
        setLoading(false);
        return;
      }
      const formData = new FormData();
      formData.append("medias", blob, `${Date.now()}.${extensionFor(blob.type)}`);
      // Gravação do Watink sempre vai como MENSAGEM DE VOZ (bolha com onda), e sem legenda.
      formData.append("body", "");
      formData.append("audioKind", "voice");
      formData.append("fromMe", "true");
      await api.post(`/messages/${ticketId}`, formData);
    } catch (err) {
      releaseMic();
      setRecording(false);
      toastError(err);
    }
    setLoading(false);
  };

  const handleCancelAudio = async () => {
    try {
      await stopAndCollect();
    } finally {
      releaseMic();
      setRecording(false);
    }
  };

  return { recording, handleStartRecording, handleUploadAudio, handleCancelAudio };
}
