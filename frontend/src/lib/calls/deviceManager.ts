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
