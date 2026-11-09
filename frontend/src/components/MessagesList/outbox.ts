import api from "../../services/api";
import toastError from "../../errors/toastError";
import { Message, MessagesAction } from "./types";

// Caixa de saída: a mensagem aparece na conversa no instante do Enter (relógio, ack 0) e o POST sai depois, em fila por ticket (a ordem de envio é a
// ordem digitada). Campo de texto e lista são componentes irmãos, então este módulo é o ponto que os liga. Falhou: vira ack 5 e o balão oferece reenviar.
// shortcut: só texto. Mídia e áudio têm upload demorado e precisam de barra de progresso; e o reenvio de uma falha por timeout pode duplicar (sem idempotência no servidor).
type Dispatch = (action: MessagesAction) => void;
interface Entry {
  ticketId: string;
  message: Message;
  build: () => Promise<Record<string, unknown>>;
}

const LOCAL_PREFIX = "local-";
const entries = new Map<string, Entry>();
const listeners = new Map<string, Set<Dispatch>>();
const tails = new Map<string, Promise<void>>();

export const isLocalMessageId = (id: unknown): boolean => String(id).startsWith(LOCAL_PREFIX);

export const newClientMessageId = (): string =>
  `${LOCAL_PREFIX}${typeof crypto !== "undefined" && crypto.randomUUID ? crypto.randomUUID() : `${Date.now()}-${Math.random()}`}`;

const emit = (ticketId: string, action: MessagesAction) => listeners.get(ticketId)?.forEach((d) => d(action));

// A lista de um ticket se inscreve ao abrir; o que ainda está na caixa de saída (enviando ou falho) é reposto, porque trocar de ticket zera a lista.
export function subscribeOutbox(ticketId: string | number, dispatch: Dispatch): () => void {
  const key = String(ticketId);
  if (!listeners.has(key)) listeners.set(key, new Set());
  listeners.get(key)!.add(dispatch);
  entries.forEach((e) => e.ticketId === key && dispatch({ type: "ADD_MESSAGE", payload: e.message }));
  return () => {
    listeners.get(key)?.delete(dispatch);
  };
}

async function post(clientId: string): Promise<void> {
  const entry = entries.get(clientId);
  if (!entry) return;
  try {
    const payload = await entry.build();
    const { data } = await api.post(`/messages/${entry.ticketId}`, { ...payload, clientMessageId: clientId });
    entries.delete(clientId);
    const id = data?.id ?? data?.messageId;
    if (id != null) {
      emit(entry.ticketId, {
        type: "CONFIRM_LOCAL",
        clientId,
        id: String(id),
        ack: typeof data?.ack === "number" ? data.ack : undefined,
      });
    }
  } catch (err) {
    entry.message = { ...entry.message, ack: 5 };
    emit(entry.ticketId, { type: "UPDATE_MESSAGE", payload: entry.message });
    toastError(err);
  }
}

function run(clientId: string, ticketId: string) {
  const next = (tails.get(ticketId) ?? Promise.resolve()).then(() => post(clientId));
  tails.set(ticketId, next);
}

export function enqueueMessage(
  ticketId: string | number,
  message: Message,
  build: () => Promise<Record<string, unknown>>
): void {
  const key = String(ticketId);
  entries.set(String(message.id), { ticketId: key, message, build });
  emit(key, { type: "ADD_MESSAGE", payload: message });
  run(String(message.id), key);
}

export function retryMessage(clientId: string): void {
  const entry = entries.get(clientId);
  if (!entry) return;
  entry.message = { ...entry.message, ack: 0 };
  emit(entry.ticketId, { type: "UPDATE_MESSAGE", payload: entry.message });
  run(clientId, entry.ticketId);
}
