import React, { useEffect, useRef, useState } from "react";
import {
  Camera,
  Check,
  Headphones,
  Image as ImageIcon,
  Mic,
  SlidersHorizontal,
  Sparkles,
  Volume2,
} from "lucide-react";
import {
  getMediaDevices,
  getStoredDevices,
  playTestSound,
  storeDevices,
  type MediaDeviceInfoItem,
} from "../../lib/calls/deviceManager";
import {
  PRESET_BACKGROUNDS,
  VirtualBackgroundProcessor,
  type BackgroundMode,
  type VirtualBackgroundConfig,
} from "../../lib/calls/virtualBackground";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "../ui/dialog";
import { Button } from "../ui/button";

import { i18n } from "../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface DeviceSettingsModalProps {
  open: boolean;
  currentStream?: MediaStream | null;
  onClose: () => void;
  onSave: (
    audioId: string,
    videoId: string,
    audioOutputId: string,
    bgMode: BackgroundMode,
    bgImageUrl?: string,
  ) => void;
}

export const DeviceSettingsModal: React.FC<DeviceSettingsModalProps> = ({
  open,
  currentStream,
  onClose,
  onSave,
}) => {
  const [activeTab, setActiveTab] = useState<"devices" | "effects">("devices");
  const [devices, setDevices] = useState<MediaDeviceInfoItem[]>([]);
  const [audioId, setAudioId] = useState("");
  const [audioOutputId, setAudioOutputId] = useState("");
  const [videoId, setVideoId] = useState("");
  const [bgMode, setBgMode] = useState<BackgroundMode>("none");
  const [bgImageUrl, setBgImageUrl] = useState("");
  const [isPlayingTest, setIsPlayingTest] = useState(false);

  const previewVideoRef = useRef<HTMLVideoElement | null>(null);
  const [previewStream, setPreviewStream] = useState<MediaStream | null>(null);
  const modalProcessorRef = useRef<VirtualBackgroundProcessor>(new VirtualBackgroundProcessor());

  useEffect(() => {
    if (!open) {
      modalProcessorRef.current.stop();
      if (previewStream && previewStream !== currentStream) {
        previewStream.getTracks().forEach((t) => t.stop());
      }
      setPreviewStream(null);
      return;
    }

    getMediaDevices().then((list) => {
      setDevices(list);
      const stored = getStoredDevices();
      if (stored.audioInputId) setAudioId(stored.audioInputId);
      if (stored.audioOutputId) setAudioOutputId(stored.audioOutputId);
      if (stored.videoInputId) setVideoId(stored.videoInputId);
    });

    // Iniciar preview de câmera do modal
    const startPreview = async () => {
      let baseStream = currentStream;
      if (!baseStream) {
        try {
          const stored = getStoredDevices();
          const videoConstraint: Record<string, unknown> = {
            width: { ideal: 640 },
            height: { ideal: 480 },
            facingMode: "user",
          };
          if (stored.videoInputId) {
            videoConstraint.deviceId = { exact: stored.videoInputId };
          }
          baseStream = await navigator.mediaDevices.getUserMedia({
            video: videoConstraint,
            audio: false,
          });
        } catch {
          return;
        }
      }

      setPreviewStream(baseStream);
      renderPreview(baseStream, { mode: bgMode, imageUrl: bgImageUrl });
    };

    void startPreview();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  // Atualiza o preview ao vivo quando o fundo muda
  const renderPreview = async (stream: MediaStream, config: VirtualBackgroundConfig) => {
    if (config.mode === "none") {
      modalProcessorRef.current.stop();
      if (previewVideoRef.current) {
        previewVideoRef.current.srcObject = stream;
        void previewVideoRef.current.play().catch(() => {});
      }
    } else {
      const processed = await modalProcessorRef.current.start(stream, config);
      if (previewVideoRef.current) {
        previewVideoRef.current.srcObject = processed;
        void previewVideoRef.current.play().catch(() => {});
      }
    }
  };

  const handleModeChange = (mode: BackgroundMode, url?: string) => {
    setBgMode(mode);
    if (url !== undefined) setBgImageUrl(url);
    if (previewStream) {
      void renderPreview(previewStream, { mode, imageUrl: url !== undefined ? url : bgImageUrl });
    }
  };

  const handleTestSound = async () => {
    setIsPlayingTest(true);
    await playTestSound(audioOutputId);
    setIsPlayingTest(false);
  };

  const handleSave = () => {
    modalProcessorRef.current.stop();
    if (previewStream && previewStream !== currentStream) {
      previewStream.getTracks().forEach((t) => t.stop());
    }
    storeDevices({ audioInputId: audioId, audioOutputId, videoInputId: videoId });
    onSave(audioId, videoId, audioOutputId, bgMode, bgImageUrl);
    onClose();
  };

  const handleClose = () => {
    modalProcessorRef.current.stop();
    if (previewStream && previewStream !== currentStream) {
      previewStream.getTracks().forEach((t) => t.stop());
    }
    onClose();
  };

  const mics = devices.filter((d) => d.kind === "audioinput");
  const speakers = devices.filter((d) => d.kind === "audiooutput");
  const cameras = devices.filter((d) => d.kind === "videoinput");

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-3xl overflow-hidden p-0 gap-0 rounded-2xl border bg-background shadow-2xl">
        <DialogHeader className="border-b px-6 py-4">
          <DialogTitle className="flex items-center gap-2 text-lg font-semibold">
            <SlidersHorizontal className="h-5 w-5 text-primary" />
            {tr("uiText.calls.settingsTitle")}
          </DialogTitle>
        </DialogHeader>

        <div className="grid grid-cols-1 md:grid-cols-12 min-h-[420px]">
          {/* Lado Esquerdo: Preview da Câmera (Google Meet style) */}
          <div className="md:col-span-6 bg-muted/30 p-6 flex flex-col justify-between border-b md:border-b-0 md:border-r border-border">
            <div className="relative aspect-video w-full overflow-hidden rounded-xl bg-black/90 shadow-inner flex items-center justify-center">
              <video
                ref={previewVideoRef}
                muted
                playsInline
                className="h-full w-full object-cover transform -scale-x-100"
              />
              <div className="absolute top-3 left-3 rounded-full bg-black/60 backdrop-blur-md px-3 py-1 text-xs font-medium text-white flex items-center gap-1.5 shadow">
                <Camera className="h-3.5 w-3.5" />
                <span>{tr("uiText.calls.youLive")}</span>
              </div>
            </div>

            <div className="mt-4 rounded-lg bg-card/60 border p-3 text-xs text-muted-foreground flex flex-col gap-1">
              <span className="font-medium text-foreground">{tr("uiText.calls.livePreview")}</span>
              <span>{tr("uiText.calls.livePreviewHint")}</span>
            </div>
          </div>

          {/* Lado Direito: Abas de Dispositivos e Efeitos */}
          <div className="md:col-span-6 flex flex-col justify-between p-6">
            <div>
              {/* Abas estilizadas estilo Google Meet */}
              <div className="flex border-b border-border mb-5">
                <button
                  type="button"
                  onClick={() => setActiveTab("devices")}
                  className={`pb-2.5 px-3 text-sm font-medium transition-colors border-b-2 flex items-center gap-2 ${
                    activeTab === "devices"
                      ? "border-primary text-primary"
                      : "border-transparent text-muted-foreground hover:text-foreground"
                  }`}
                >
                  <Headphones className="h-4 w-4" />
                  Dispositivos
                </button>
                <button
                  type="button"
                  onClick={() => setActiveTab("effects")}
                  className={`pb-2.5 px-3 text-sm font-medium transition-colors border-b-2 flex items-center gap-2 ${
                    activeTab === "effects"
                      ? "border-primary text-primary"
                      : "border-transparent text-muted-foreground hover:text-foreground"
                  }`}
                >
                  <Sparkles className="h-4 w-4" />
                  Efeitos e Fundo
                </button>
              </div>

              {activeTab === "devices" ? (
                <div className="space-y-4">
                  <div>
                    <label className="text-xs font-semibold text-foreground flex items-center gap-1.5 mb-1">
                      <Mic className="h-3.5 w-3.5 text-primary" />
                      Microfone
                    </label>
                    <select
                      value={audioId}
                      onChange={(e) => setAudioId(e.target.value)}
                      className="w-full rounded-lg border border-input bg-card px-3 py-2 text-sm shadow-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                    >
                      <option value="">{tr("uiText.common.systemDefault")}</option>
                      {mics.map((m) => (
                        <option key={m.deviceId} value={m.deviceId}>
                          {m.label}
                        </option>
                      ))}
                    </select>
                  </div>

                  <div>
                    <div className="flex items-center justify-between mb-1">
                      <label className="text-xs font-semibold text-foreground flex items-center gap-1.5">
                        <Volume2 className="h-3.5 w-3.5 text-primary" />
                        Alto-falante / Fone
                      </label>
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        disabled={isPlayingTest}
                        onClick={handleTestSound}
                        className="h-6 px-2 text-xs text-primary hover:bg-primary/10"
                      >
                        {isPlayingTest ? "Tocando..." : "Testar som"}
                      </Button>
                    </div>
                    <select
                      value={audioOutputId}
                      onChange={(e) => setAudioOutputId(e.target.value)}
                      className="w-full rounded-lg border border-input bg-card px-3 py-2 text-sm shadow-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                    >
                      <option value="">{tr("uiText.common.systemDefault")}</option>
                      {speakers.map((s) => (
                        <option key={s.deviceId} value={s.deviceId}>
                          {s.label}
                        </option>
                      ))}
                    </select>
                  </div>

                  <div>
                    <label className="text-xs font-semibold text-foreground flex items-center gap-1.5 mb-1">
                      <Camera className="h-3.5 w-3.5 text-primary" />
                      {tr("uiText.calls.camera")}
                    </label>
                    <select
                      value={videoId}
                      onChange={(e) => setVideoId(e.target.value)}
                      className="w-full rounded-lg border border-input bg-card px-3 py-2 text-sm shadow-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                    >
                      <option value="">{tr("uiText.common.systemDefault")}</option>
                      {cameras.map((c) => (
                        <option key={c.deviceId} value={c.deviceId}>
                          {c.label}
                        </option>
                      ))}
                    </select>
                  </div>
                </div>
              ) : (
                <div className="space-y-4">
                  <div>
                    <span className="text-xs font-semibold text-foreground mb-2 block">
                      {tr("uiText.calls.backgrounds")}
                    </span>
                    <div className="grid grid-cols-3 gap-2.5">
                      {/* Sem efeito */}
                      <button
                        type="button"
                        onClick={() => handleModeChange("none")}
                        className={`group relative flex flex-col items-center justify-center rounded-xl border p-3 text-center transition-all ${
                          bgMode === "none"
                            ? "border-primary bg-primary/10 shadow-sm"
                            : "border-border bg-card hover:bg-muted/50"
                        }`}
                      >
                        <div className="h-7 w-7 rounded-full bg-muted flex items-center justify-center text-muted-foreground group-hover:text-foreground">
                          🚫
                        </div>
                        <span className="mt-1.5 text-[11px] font-medium leading-tight">Nenhum</span>
                        {bgMode === "none" && (
                          <Check className="absolute top-1.5 right-1.5 h-3.5 w-3.5 text-primary" />
                        )}
                      </button>

                      {/* Desfoque leve */}
                      <button
                        type="button"
                        onClick={() => handleModeChange("blur_slight")}
                        className={`group relative flex flex-col items-center justify-center rounded-xl border p-3 text-center transition-all ${
                          bgMode === "blur_slight"
                            ? "border-primary bg-primary/10 shadow-sm"
                            : "border-border bg-card hover:bg-muted/50"
                        }`}
                      >
                        <div className="h-7 w-7 rounded-full bg-muted/80 backdrop-blur-sm flex items-center justify-center text-muted-foreground">
                          🌫️
                        </div>
                        <span className="mt-1.5 text-[11px] font-medium leading-tight">Desfoque Leve</span>
                        {bgMode === "blur_slight" && (
                          <Check className="absolute top-1.5 right-1.5 h-3.5 w-3.5 text-primary" />
                        )}
                      </button>

                      {/* Desfoque intenso */}
                      <button
                        type="button"
                        onClick={() => handleModeChange("blur_full")}
                        className={`group relative flex flex-col items-center justify-center rounded-xl border p-3 text-center transition-all ${
                          bgMode === "blur_full"
                            ? "border-primary bg-primary/10 shadow-sm"
                            : "border-border bg-card hover:bg-muted/50"
                        }`}
                      >
                        <div className="h-7 w-7 rounded-full bg-muted/80 backdrop-blur-md flex items-center justify-center text-muted-foreground">
                          🌫️
                        </div>
                        <span className="mt-1.5 text-[11px] font-medium leading-tight">Desfoque Total</span>
                        {bgMode === "blur_full" && (
                          <Check className="absolute top-1.5 right-1.5 h-3.5 w-3.5 text-primary" />
                        )}
                      </button>

                      {/* Presets Corporativos */}
                      {PRESET_BACKGROUNDS.map((preset) => (
                        <button
                          key={preset.id}
                          type="button"
                          onClick={() => handleModeChange("image", preset.url)}
                          className={`group relative flex flex-col items-center justify-center overflow-hidden rounded-xl border p-1 text-center transition-all ${
                            bgMode === "image" && bgImageUrl === preset.url
                              ? "border-primary ring-2 ring-primary/40 shadow-sm"
                              : "border-border bg-card hover:opacity-90"
                          }`}
                        >
                          <img
                            src={preset.url}
                            alt={tr(preset.name)}
                            className="h-10 w-full rounded-lg object-cover"
                          />
                          <span className="mt-1 text-[10px] font-medium truncate w-full px-1">
                            {tr(preset.name)}
                          </span>
                          {bgMode === "image" && bgImageUrl === preset.url && (
                            <Check className="absolute top-1.5 right-1.5 h-3.5 w-3.5 rounded-full bg-primary text-primary-foreground p-0.5 shadow" />
                          )}
                        </button>
                      ))}
                    </div>
                  </div>

                  {/* Upload customizado */}
                  <div className="rounded-xl border border-dashed border-border/80 bg-muted/20 p-3">
                    <label className="text-[11px] font-semibold text-foreground flex items-center gap-1.5 mb-1.5 cursor-pointer">
                      <ImageIcon className="h-3.5 w-3.5 text-primary" />
                      {tr("uiText.calls.uploadPhoto")}
                    </label>
                    <input
                      type="file"
                      accept="image/*"
                      onChange={(e) => {
                        const file = e.target.files?.[0];
                        if (file) {
                          const reader = new FileReader();
                          reader.onload = () => {
                            const res = reader.result as string;
                            handleModeChange("image", res);
                          };
                          reader.readAsDataURL(file);
                        }
                      }}
                      className="w-full text-xs text-muted-foreground file:mr-2 file:rounded-md file:border-0 file:bg-primary/10 file:text-primary file:px-2.5 file:py-1 file:text-xs file:font-medium hover:file:bg-primary/20 cursor-pointer"
                    />
                  </div>
                </div>
              )}
            </div>

            <div className="flex justify-end gap-2.5 pt-6 border-t border-border mt-4">
              <Button variant="outline" onClick={handleClose}>
                {tr("uiText.common.cancel")}
              </Button>
              <Button onClick={handleSave} className="px-5 shadow-sm">
                {tr("uiText.calls.done")}
              </Button>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
};
