/**
 * Geometria para exibir em pé o vídeo que o celular manda deitado. O aparelho do contato anuncia a
 * rotação (CVO) em quartos de volta HORÁRIOS que o RECEPTOR deve aplicar; o desenho é o mesmo que
 * a meowcaller usa no console web (`drawRemoteFrame`), conferido contra capturas reais.
 */
export type Rotation = 0 | 1 | 2 | 3;

export interface RotationPlan {
  /** Tamanho do canvas depois de girar: nos quartos ímpares largura e altura trocam. */
  canvasWidth: number;
  canvasHeight: number;
  /** Matriz 2D `[a, b, c, d, e, f]` (a mesma ordem de `ctx.setTransform`) a aplicar antes de desenhar em (0,0). */
  matrix: [number, number, number, number, number, number];
}

/** Planeja o desenho de um quadro `width × height` girado `rotation` quartos de volta no sentido horário. */
export function planRotation(width: number, height: number, rotation: Rotation): RotationPlan {
  const q = (((rotation as number) % 4) + 4) % 4;
  const portrait = q % 2 === 1;
  const canvasWidth = portrait ? height : width;
  const canvasHeight = portrait ? width : height;
  switch (q) {
    case 1: // 90° horário
      return { canvasWidth, canvasHeight, matrix: [0, 1, -1, 0, canvasWidth, 0] };
    case 2: // 180°
      return { canvasWidth, canvasHeight, matrix: [-1, 0, 0, -1, canvasWidth, canvasHeight] };
    case 3: // 270° horário (= 90° anti-horário)
      return { canvasWidth, canvasHeight, matrix: [0, -1, 1, 0, 0, canvasHeight] };
    default:
      return { canvasWidth, canvasHeight, matrix: [1, 0, 0, 1, 0, 0] };
  }
}

/** Aplica a matriz a um ponto (para os testes e para conferir a geometria sem um canvas). */
export function applyMatrix(m: RotationPlan["matrix"], x: number, y: number): [number, number] {
  return [m[0] * x + m[2] * y + m[4], m[1] * x + m[3] * y + m[5]];
}
