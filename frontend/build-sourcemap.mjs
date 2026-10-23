// Mapas de fonte só em build pontual de depuração (VITE_SOURCEMAP=true). O build/ vai inteiro para a imagem do
// business, que serve /assets/ sem filtro: publicar o mapa expõe o TypeScript original a qualquer visitante (F12).
// Qualquer outro valor, inclusive um typo, deixa desligado.
export function buildSourcemap(env) {
  return String(env.VITE_SOURCEMAP ?? "").trim().toLowerCase() === "true";
}
