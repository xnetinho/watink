export interface MediaDeviceInfoItem {
  deviceId: string;
  label: string;
  kind: MediaDeviceKind;
}

export interface SelectedDevices {
  audioInputId: string;
  audioOutputId: string;
  videoInputId: string;
}

const STORAGE_KEY = "watink_selected_devices_v1";

export async function getMediaDevices(): Promise<MediaDeviceInfoItem[]> {
  if (!navigator.mediaDevices?.enumerateDevices) return [];
  const devices = await navigator.mediaDevices.enumerateDevices();
  return devices.map((d) => ({
    deviceId: d.deviceId,
    label: d.label || `${d.kind} (${d.deviceId.slice(0, 4)}...)`,
    kind: d.kind,
  }));
}

export function getStoredDevices(): SelectedDevices {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) return JSON.parse(raw) as SelectedDevices;
  } catch {
    /* ignore */
  }
  return { audioInputId: "", audioOutputId: "", videoInputId: "" };
}

export function storeDevices(sel: SelectedDevices): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(sel));
  } catch {
    /* ignore */
  }
}

/** Toca um som de teste curto ("ding-dong") no alto-falante selecionado para validação imediata do operador. */
export function playTestSound(sinkId?: string): Promise<void> {
  try {
    const ctx = new AudioContext();
    const osc = ctx.createOscillator();
    const gain = ctx.createGain();
    osc.type = "sine";
    osc.frequency.setValueAtTime(587.33, ctx.currentTime);
    osc.frequency.setValueAtTime(880, ctx.currentTime + 0.15);
    gain.gain.setValueAtTime(0.2, ctx.currentTime);
    gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.6);
    osc.connect(gain);
    gain.connect(ctx.destination);
    if (sinkId && typeof (ctx as unknown as { setSinkId?: (id: string) => Promise<void> }).setSinkId === "function") {
      void (ctx as unknown as { setSinkId: (id: string) => Promise<void> }).setSinkId(sinkId).catch(() => {});
    }
    osc.start();
    osc.stop(ctx.currentTime + 0.6);
    return new Promise((resolve) => setTimeout(resolve, 600));
  } catch {
    return Promise.resolve();
  }
}
