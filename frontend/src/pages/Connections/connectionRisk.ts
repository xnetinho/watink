import { i18n } from "../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");

/**
 * Classificação de risco de ban/throttle a partir do último sinal
 * `session.risk` persistido na conexão (whatsmeow IQ 401/403/429/463).
 *
 * Estado sempre DERIVADO (nunca persistido) — recalculado a partir de
 * `lastRiskCode`/`lastRiskAt`, no mesmo espírito do Checklist de Onboarding.
 */

export type ConnectionHealthLevel = "ok" | "warning" | "critical";

const ONE_HOUR_MS = 60 * 60 * 1000;
const ONE_DAY_MS = 24 * ONE_HOUR_MS;

/** Códigos que por si só já indicam throttle de IP/volume (mesmos que o
 * business auto-isola o proxy). */
const CRITICAL_CODES = new Set([429, 463]);

export interface ConnectionRiskInfo {
  code?: number;
  at?: string | null;
}

export function getConnectionHealthLevel(risk: ConnectionRiskInfo): ConnectionHealthLevel {
  if (!risk.code || !risk.at) return "ok";

  const riskAt = new Date(risk.at).getTime();
  if (Number.isNaN(riskAt)) return "ok";

  const ageMs = Date.now() - riskAt;
  if (ageMs < 0 || ageMs > ONE_DAY_MS) return "ok";

  if (CRITICAL_CODES.has(risk.code) && ageMs < ONE_HOUR_MS) return "critical";
  return "warning";
}

/** Explicação em português de cada código de risco, para exibir ao usuário
 * exatamente o que está acontecendo — não só o número do erro. */
export const RISK_CODE_EXPLANATIONS: Record<number, string> = {
  401: "uiText.connections.risk.code401",
  403: "uiText.connections.risk.code403",
  429: "uiText.connections.risk.code429",
  463: "uiText.connections.risk.code463",
};

export function describeRiskCode(code?: number): string {
  if (!code) return "";
  return RISK_CODE_EXPLANATIONS[code] ? tr(RISK_CODE_EXPLANATIONS[code]) : tr("uiText.connections.risk.codeOther", { code });
}

/** Referência educativa dos limiares de segurança citados na pesquisa
 * anti-ban — conteúdo estático, não calculado a partir de dados reais. */
export const ANTI_BAN_THRESHOLDS: { label: string; safe: string; danger: string }[] = [
  { label: "uiText.connections.risk.responseRate", safe: "> 30%", danger: "< 15%" },
  { label: "uiText.connections.risk.blockRate", safe: "< 2%", danger: "> 2%" },
  { label: "uiText.connections.risk.newContacts", safe: "uiText.connections.risk.perDaySafe", danger: "uiText.connections.risk.perDayDanger" },
  { label: "uiText.connections.risk.proactive", safe: "uiText.connections.risk.proactiveSafe", danger: "uiText.connections.risk.proactiveDanger" },
];
