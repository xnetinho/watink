import React, { useState } from "react";
import { CalendarDays, Copy, Check, MapPin, Video, QrCode, Radio } from "lucide-react";
import { parseData } from "../utils/messageHelpers";
import api from "../../../services/api";
import toastError from "../../../errors/toastError";
import { Message } from "../types";

const card = "flex max-w-[300px] flex-col gap-1.5 rounded-xl bg-[hsl(var(--bg-surface))] px-3 py-2.5 text-sm";

export const getRich = (message: Message): Record<string, unknown> =>
  (parseData(message.dataJson).rich as Record<string, unknown> | undefined) ?? {};

const isHttps = (u: unknown): u is string => typeof u === "string" && /^https:\/\//i.test(u);

export const PixCard: React.FC<{ message: Message }> = ({ message }) => {
  const r = getRich(message);
  const [copied, setCopied] = useState(false);
  const key = typeof r.key === "string" ? r.key : "";
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(typeof r.code === "string" && r.code ? r.code : key);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 2000);
    } catch {
      /* sem permissão de área de transferência: a chave segue visível para copiar à mão */
    }
  };
  return (
    <div className={card} data-testid="pix-card">
      <div className="flex items-center gap-2 font-medium text-[hsl(var(--action-primary))]">
        <QrCode className="h-4 w-4" /> Pagar com PIX
      </div>
      {typeof r.name === "string" && r.name && <div className="text-xs text-[hsl(var(--text-muted))]">{r.name}</div>}
      {typeof r.amount === "number" && r.amount > 0 && (
        <div className="text-base font-semibold">{r.amount.toLocaleString("pt-BR", { style: "currency", currency: "BRL" })}</div>
      )}
      {key && (
        <div className="break-all text-xs">
          <span className="text-[hsl(var(--text-muted))]">Chave{typeof r.keyType === "string" ? ` (${r.keyType})` : ""}: </span>
          {key}
        </div>
      )}
      {key && (
        <button
          type="button"
          onClick={copy}
          className="mt-0.5 flex items-center justify-center gap-1.5 rounded-lg bg-[hsl(var(--action-primary)/0.1)] py-1.5 text-xs font-medium text-[hsl(var(--action-primary))]"
        >
          {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
          {copied ? "Copiado" : "Copiar chave PIX"}
        </button>
      )}
    </div>
  );
};

const fmtWhen = (secs: unknown): string => {
  const n = Number(secs);
  if (!n) return "";
  return new Date(n * 1000).toLocaleString("pt-BR", { dateStyle: "short", timeStyle: "short" });
};

const LegacyText: React.FC<{ message: Message; testId: string }> = ({ message, testId }) => (
  <div className={card} data-testid={testId}>
    <div className="whitespace-pre-wrap break-words">{message.body}</div>
  </div>
);

export const EventCard: React.FC<{ message: Message }> = ({ message }) => {
  const r = getRich(message);
  if (!r.name) return <LegacyText message={message} testId="event-card" />;
  const when = fmtWhen(r.start);
  const end = fmtWhen(r.end);
  return (
    <div className={card} data-testid="event-card">
      <div className="flex items-center gap-2 font-medium">
        <CalendarDays className="h-4 w-4 shrink-0 text-[hsl(var(--action-primary))]" />
        <span className="break-words">{String(r.name ?? "Evento")}</span>
      </div>
      {r.canceled === true && <div className="text-xs font-medium text-[hsl(var(--status-error))]">Evento cancelado</div>}
      {when && <div className="text-xs text-[hsl(var(--text-muted))]">{when}{end ? ` até ${end}` : ""}</div>}
      {typeof r.location === "string" && r.location && (
        <div className="flex items-center gap-1.5 text-xs"><MapPin className="h-3.5 w-3.5 shrink-0" />{r.location}</div>
      )}
      {typeof r.description === "string" && r.description && <div className="whitespace-pre-wrap text-xs">{r.description}</div>}
      {isHttps(r.joinLink) && r.canceled !== true && (
        <a
          href={r.joinLink}
          target="_blank"
          rel="noopener noreferrer"
          className="mt-0.5 flex items-center justify-center gap-1.5 rounded-lg bg-[hsl(var(--action-primary))] py-1.5 text-xs font-medium text-white"
        >
          <Video className="h-3.5 w-3.5" /> Entrar na reunião
        </a>
      )}
    </div>
  );
};

export const PollCard: React.FC<{ message: Message }> = ({ message }) => {
  const r = getRich(message);
  const options = Array.isArray(r.options) ? (r.options as string[]) : [];
  const mine = (parseData(message.dataJson).myVote as string[] | undefined) ?? [];
  const [chosen, setChosen] = useState<string[]>(mine);
  const [busy, setBusy] = useState(false);
  const [sent, setSent] = useState(mine.length > 0);
  if (options.length === 0) return <LegacyText message={message} testId="poll-card" />;

  // Votos de todos (dataJson.votes[jid] = opções), inclusive o do operador (chave "__me__").
  const votes = (parseData(message.dataJson).votes as Record<string, string[]> | undefined) ?? {};
  const tally: Record<string, number> = {};
  Object.values(votes).forEach((list) => (list ?? []).forEach((o) => (tally[o] = (tally[o] ?? 0) + 1)));
  const voters = Object.values(votes).filter((l) => (l ?? []).length > 0).length;
  const limit = Number(r.selectable) || options.length;
  const multi = limit > 1;
  // O id da mensagem de canal agnóstico começa com "<canal>:" (ex.: telegram:2:123:7); o do WhatsApp é o id cru.
  const isTelegram = String(message.id).includes(":");
  // WhatsApp: vota-se em enquete recebida. Telegram: a Bot API não tem como um bot votar, então só mostra o resultado.
  const canVote = !message.fromMe && !isTelegram;

  const toggle = (o: string) => {
    setSent(false);
    // Na única, clicar na opção já marcada a desmarca (permite remover o voto); na múltipla alterna até o limite.
    setChosen((cur) => {
      if (cur.includes(o)) return cur.filter((x) => x !== o);
      if (!multi) return [o];
      return cur.length >= limit ? cur : [...cur, o];
    });
  };
  const vote = async () => {
    setBusy(true);
    try {
      await api.post(`/message/${message.id}/vote`, { options: chosen });
      setSent(true);
    } catch (err) {
      toastError(err);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="flex w-[330px] max-w-full flex-col gap-2.5 rounded-xl bg-[hsl(var(--bg-surface))] px-4 py-3 text-[15px]" data-testid="poll-card">
      <div className="text-base font-semibold leading-snug">{String(r.question ?? "")}</div>
      <div className="text-xs text-[hsl(var(--text-muted))]">
        {isTelegram ? (multi ? "Enquete · várias respostas" : "Enquete") : multi ? `Selecione até ${limit} opções` : "Selecione uma opção"}
      </div>
      {options.map((o, i) => {
        const on = chosen.includes(o);
        const id = `poll-${message.id}-${i}`;
        const n = tally[o] ?? 0;
        const pct = voters > 0 ? Math.round((n / voters) * 100) : 0;
        return (
          <div key={i} className="flex flex-col gap-1">
            <label htmlFor={id} className={`flex items-center gap-2.5 ${canVote ? "cursor-pointer" : ""}`}>
              {canVote && (
                <input
                  id={id}
                  type={multi ? "checkbox" : "radio"}
                  name={`poll-${message.id}`}
                  checked={on}
                  disabled={busy}
                  // radio não dispara onChange ao clicar no que já está marcado: o clique cuida de desmarcar.
                  onChange={() => undefined}
                  onClick={() => toggle(o)}
                  className="h-5 w-5 shrink-0 accent-[hsl(var(--action-primary))]"
                />
              )}
              <span className="flex-1 break-words">{o}</span>
              {n > 0 && (
                <span className="text-sm font-medium" data-testid="poll-count">
                  {n}
                </span>
              )}
            </label>
            {voters > 0 && (
              <div className="ml-[30px] h-1 overflow-hidden rounded-full bg-[var(--border-subtle)]" aria-hidden>
                <div className="h-full rounded-full bg-[hsl(var(--action-primary))]" style={{ width: `${pct}%` }} />
              </div>
            )}
          </div>
        );
      })}
      <div className="text-xs text-[hsl(var(--text-muted))]" data-testid="poll-voters">
        {voters} {voters === 1 ? "voto" : "votos"}
      </div>
      {canVote && (
        <button
          type="button"
          onClick={vote}
          disabled={busy || sent || (chosen.length === 0 && mine.length === 0)}
          className="rounded-lg bg-[hsl(var(--action-primary))] py-2 text-sm font-medium text-white disabled:opacity-50"
        >
          {busy ? "Enviando…" : sent ? "Voto enviado" : chosen.length === 0 ? "Remover voto" : "Votar"}
        </button>
      )}
    </div>
  );
};

export const LiveLocationBadge: React.FC<{ message: Message }> = ({ message }) =>
  getRich(message).live === true ? (
    <div className="mb-1 flex items-center gap-1.5 text-xs font-medium text-[hsl(var(--status-error))]" data-testid="live-location">
      <Radio className="h-3.5 w-3.5" /> Localização em tempo real
    </div>
  ) : null;
