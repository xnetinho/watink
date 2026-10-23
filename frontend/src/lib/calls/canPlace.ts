export type PlaceBlock = "not_individual" | "disconnected" | "proxy" | "busy" | null;

export interface PlaceInput {
  isGroup?: boolean;
  isCommunity?: boolean;
  isSubGroup?: boolean;
}

export interface ConnectionLike {
  status?: string;
  proxyMode?: unknown;
  proxyId?: unknown;
  proxyGroupId?: unknown;
}

/** Conexão com proxy configurado (qualquer modo, inclusive grupo sem pick atual). */
export function connectionHasProxy(c: ConnectionLike | undefined): boolean {
  if (!c) return false;
  const mode = typeof c.proxyMode === "string" ? c.proxyMode : "";
  return (mode !== "" && mode !== "none") || c.proxyId != null || c.proxyGroupId != null;
}

/**
 * Motivo pelo qual NÃO se pode ligar a partir deste ticket, ou null se pode. A
 * ordem é a da explicação mais útil ao operador. O servidor repete todas estas
 * checagens (esta função só decide o que mostrar).
 */
export function placeBlock(ticket: PlaceInput, connection: ConnectionLike | undefined, inCall: boolean): PlaceBlock {
  if (ticket.isGroup || ticket.isCommunity || ticket.isSubGroup) return "not_individual";
  if (inCall) return "busy";
  if (!connection || connection.status !== "CONNECTED") return "disconnected";
  if (connectionHasProxy(connection)) return "proxy";
  return null;
}

export const PLACE_BLOCK_KEY: Record<Exclude<PlaceBlock, null>, string> = {
  not_individual: "calls.place.notIndividual",
  disconnected: "calls.place.disconnected",
  proxy: "calls.place.proxy",
  busy: "calls.place.busy",
};
