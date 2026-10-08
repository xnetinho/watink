import { describe, expect, it } from "vitest";
import { applyMatrix, planRotation, type Rotation } from "../videoRotation";

// Quadro 4×2: o canto superior esquerdo (0,0) é a "cabeça" da imagem. Depois de girar, a cabeça tem de
// ir parar no canto certo do canvas de saída; o tamanho do canvas troca nos quartos ímpares.
describe("planRotation", () => {
  it("0: sem rotação, mesmo tamanho", () => {
    const p = planRotation(4, 2, 0);
    expect([p.canvasWidth, p.canvasHeight]).toEqual([4, 2]);
    expect(applyMatrix(p.matrix, 0, 0)).toEqual([0, 0]);
    expect(applyMatrix(p.matrix, 4, 2)).toEqual([4, 2]);
  });

  it("1 (90° horário): largura e altura trocam; (0,0) vai para o canto superior direito", () => {
    const p = planRotation(4, 2, 1);
    expect([p.canvasWidth, p.canvasHeight]).toEqual([2, 4]);
    expect(applyMatrix(p.matrix, 0, 0)).toEqual([2, 0]);
    expect(applyMatrix(p.matrix, 4, 0)).toEqual([2, 4]);
    expect(applyMatrix(p.matrix, 0, 2)).toEqual([0, 0]);
  });

  it("2 (180°): mesmo tamanho; (0,0) vai para o canto oposto", () => {
    const p = planRotation(4, 2, 2);
    expect([p.canvasWidth, p.canvasHeight]).toEqual([4, 2]);
    expect(applyMatrix(p.matrix, 0, 0)).toEqual([4, 2]);
    expect(applyMatrix(p.matrix, 4, 2)).toEqual([0, 0]);
  });

  it("3 (270° horário): largura e altura trocam; (0,0) vai para o canto inferior esquerdo", () => {
    const p = planRotation(4, 2, 3);
    expect([p.canvasWidth, p.canvasHeight]).toEqual([2, 4]);
    expect(applyMatrix(p.matrix, 0, 0)).toEqual([0, 4]);
    expect(applyMatrix(p.matrix, 4, 0)).toEqual([0, 0]);
    expect(applyMatrix(p.matrix, 0, 2)).toEqual([2, 4]);
  });

  it("os quatro cantos do quadro sempre caem DENTRO do canvas de saída (nada fica de fora)", () => {
    for (const rot of [0, 1, 2, 3] as Rotation[]) {
      const p = planRotation(640, 480, rot);
      for (const [x, y] of [[0, 0], [640, 0], [0, 480], [640, 480]] as const) {
        const [ox, oy] = applyMatrix(p.matrix, x, y);
        expect(ox).toBeGreaterThanOrEqual(0);
        expect(ox).toBeLessThanOrEqual(p.canvasWidth);
        expect(oy).toBeGreaterThanOrEqual(0);
        expect(oy).toBeLessThanOrEqual(p.canvasHeight);
      }
    }
  });

  it("quatro quartos de volta voltam ao início (a soma das rotações é cíclica)", () => {
    expect(planRotation(10, 6, 0).matrix).toEqual([1, 0, 0, 1, 0, 0]);
    // 4 não existe no tipo, mas um valor fora do intervalo é normalizado em vez de quebrar o desenho
    expect(planRotation(10, 6, 4 as Rotation).matrix).toEqual(planRotation(10, 6, 0).matrix);
  });

  it("o retrato do celular (CVO 3) vira uma imagem mais ALTA que larga", () => {
    // o celular em pé manda 640×480 deitado; o receptor gira 270° e obtém 480×640 (retrato)
    const p = planRotation(640, 480, 3);
    expect(p.canvasHeight).toBeGreaterThan(p.canvasWidth);
  });
});
