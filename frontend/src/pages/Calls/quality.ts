// Mesmos limites do backend (internal/calls/quality.go). A tabela mostra o nível
// do RESUMO da chamada (médias e piores), então a classificação é refeita aqui a
// partir das métricas agregadas.

export type QualityLevel = 1 | 2 | 3;

export interface QualitySummary {
  rttMax: number | null;
  lossMax: number | null;
  jitterMax: number | null;
  mosEstimated: number | null;
}

/** Pior fator manda; sem nenhuma medição devolve null (nunca "boa" por falta de dado). */
export function summaryLevel(s: QualitySummary): QualityLevel | null {
  if (s.rttMax == null && s.lossMax == null && s.jitterMax == null) return null;
  let level: QualityLevel = 3;
  const lower = (to: QualityLevel) => {
    if (to < level) level = to;
  };
  if (s.rttMax != null) {
    if (s.rttMax > 400) lower(1);
    else if (s.rttMax > 200) lower(2);
  }
  if (s.lossMax != null) {
    if (s.lossMax > 5) lower(1);
    else if (s.lossMax > 2) lower(2);
  }
  if (s.jitterMax != null) {
    if (s.jitterMax > 60) lower(1);
    else if (s.jitterMax > 30) lower(2);
  }
  return level;
}

export function formatMos(mos: number | null): string {
  return mos == null ? "—" : mos.toFixed(1).replace(".", ",");
}
