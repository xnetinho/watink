import React, { useEffect, useState } from "react";
import { getMediaDevices, getStoredDevices, storeDevices, type MediaDeviceInfoItem } from "../../lib/calls/deviceManager";
import { type BackgroundMode } from "../../lib/calls/virtualBackground";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "../ui/dialog";
import { Button } from "../ui/button";

interface DeviceSettingsModalProps {
  open: boolean;
  onClose: () => void;
  onSave: (audioId: string, videoId: string, bgMode: BackgroundMode) => void;
}

export const DeviceSettingsModal: React.FC<DeviceSettingsModalProps> = ({ open, onClose, onSave }) => {
  const [devices, setDevices] = useState<MediaDeviceInfoItem[]>([]);
  const [audioId, setAudioId] = useState("");
  const [videoId, setVideoId] = useState("");
  const [bgMode, setBgMode] = useState<BackgroundMode>("none");

  useEffect(() => {
    if (!open) return;
    getMediaDevices().then((list) => {
      setDevices(list);
      const stored = getStoredDevices();
      if (stored.audioInputId) setAudioId(stored.audioInputId);
      if (stored.videoInputId) setVideoId(stored.videoInputId);
    });
  }, [open]);

  const handleSave = () => {
    storeDevices({ audioInputId: audioId, audioOutputId: "", videoInputId: videoId });
    onSave(audioId, videoId, bgMode);
    onClose();
  };

  const mics = devices.filter((d) => d.kind === "audioinput");
  const cameras = devices.filter((d) => d.kind === "videoinput");

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Configurações de Áudio, Vídeo e Fundo</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 py-2">
          <div>
            <label className="text-sm font-medium">Microfone</label>
            <select
              value={audioId}
              onChange={(e) => setAudioId(e.target.value)}
              className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            >
              <option value="">Padrão do Sistema</option>
              {mics.map((m) => (
                <option key={m.deviceId} value={m.deviceId}>{m.label}</option>
              ))}
            </select>
          </div>

          <div>
            <label className="text-sm font-medium">Câmera</label>
            <select
              value={videoId}
              onChange={(e) => setVideoId(e.target.value)}
              className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            >
              <option value="">Padrão do Sistema</option>
              {cameras.map((c) => (
                <option key={c.deviceId} value={c.deviceId}>{c.label}</option>
              ))}
            </select>
          </div>

          <div>
            <label className="text-sm font-medium">Plano de Fundo Virtual</label>
            <select
              value={bgMode}
              onChange={(e) => setBgMode(e.target.value as BackgroundMode)}
              className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            >
              <option value="none">Normal (Sem Fundo Virtual)</option>
              <option value="blur">Desfoque de Fundo (Blur)</option>
              <option value="image">Imagem Corporativa</option>
            </select>
          </div>
        </div>
        <div className="flex justify-end space-x-2 pt-2">
          <Button variant="outline" onClick={onClose}>Cancelar</Button>
          <Button onClick={handleSave}>Salvar e Aplicar</Button>
        </div>
      </DialogContent>
    </Dialog>
  );
};
