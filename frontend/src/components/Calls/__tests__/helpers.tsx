import React from "react";
import { vi } from "vitest";
import { CallsContext, type CallsContextValue } from "@/context/Calls/CallsContext";
import { initialCallsState } from "@/context/Calls/callReducer";
import type { ActiveCall, CallQuality } from "@/context/Calls/types";
import { TooltipProvider } from "@/components/ui/tooltip";
import { VideoSink } from "@/lib/calls/videoSink";
import { i18n } from "@/translate/i18n";

// Os testes de componente rodam sempre em português, independente do idioma do
// navegador de quem executa (o jsdom detecta en-US).
void i18n.changeLanguage("pt");

export const baseCall = (over: Partial<ActiveCall> = {}): ActiveCall => ({
  callId: "C1",
  direction: "incoming",
  media: "audio",
  phase: "ringing",
  whatsappId: 1,
  contact: { name: "Maria Souza", number: "5511999990001" },
  ticketId: 10,
  connectedAt: null,
  muted: false,
  camera: false,
  cameraFailure: null,
  recording: false,
  quality: null,
  endReason: null,
  failure: null,
  ...over,
});

export const quality = (over: Partial<CallQuality> = {}): CallQuality => ({
  rttMs: 40,
  lossPct: 0,
  jitterMs: 5,
  level: 3,
  mosEstimated: 4.3,
  alerts: [],
  noPeerAudio: false,
  silentMs: 0,
  txBytesPerSec: 1500,
  rxBytesPerSec: 1600,
  txLevel: 0.2,
  rxLevel: 0.3,
  ...over,
});

export function makeCtx(over: Partial<CallsContextValue> = {}): CallsContextValue {
  return {
    state: initialCallsState(),
    canReceive: true,
    canPlace: true,
    ringing: [],
    active: null,
    paused: false,
    accept: vi.fn(async () => undefined),
    reject: vi.fn(async () => undefined),
    place: vi.fn(async () => undefined),
    end: vi.fn(async () => undefined),
    setMuted: vi.fn(),
    setCamera: vi.fn(async () => undefined),
    canSendVideo: true,
    setPaused: vi.fn(),
    startRecording: vi.fn(async () => undefined),
    stopRecording: vi.fn(async () => undefined),
    dismiss: vi.fn(),
    audioLevels: { tx: 0, rx: 0 },
    videoSink: new VideoSink(undefined, undefined),
    ...over,
  };
}

export const withCalls = (ctx: CallsContextValue, ui: React.ReactNode) => (
  <TooltipProvider>
    <CallsContext.Provider value={ctx}>{ui}</CallsContext.Provider>
  </TooltipProvider>
);
