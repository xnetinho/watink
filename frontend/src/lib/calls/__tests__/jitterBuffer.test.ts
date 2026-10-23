import { describe, expect, it } from "vitest";
import { JitterBuffer } from "../jitterBuffer";
import { FRAME_SAMPLES } from "../pcm";

const frame = (v: number) => new Int16Array(FRAME_SAMPLES).fill(v);

describe("JitterBuffer", () => {
  it("só começa a entregar depois de juntar a reserva (~60 ms = 3 quadros)", () => {
    const jb = new JitterBuffer();
    jb.push(frame(1));
    jb.push(frame(2));
    expect(jb.pull()).toBeNull();
    jb.push(frame(3));
    expect(jb.pull()?.[0]).toBe(1);
  });

  it("entrega na ordem de chegada", () => {
    const jb = new JitterBuffer();
    [1, 2, 3, 4].forEach((v) => jb.push(frame(v)));
    expect([jb.pull()?.[0], jb.pull()?.[0], jb.pull()?.[0], jb.pull()?.[0]]).toEqual([1, 2, 3, 4]);
  });

  it("acima de ~200 ms descarta os quadros MAIS ANTIGOS, não os novos", () => {
    const jb = new JitterBuffer();
    for (let v = 1; v <= 30; v++) jb.push(frame(v)); // 600 ms
    expect(jb.length).toBe(10); // 200 ms
    expect(jb.dropped).toBe(20);
    expect(jb.pull()?.[0]).toBe(21); // o mais antigo que sobrou
  });

  it("o atraso nunca passa do máximo, por mais áudio que chegue", () => {
    const jb = new JitterBuffer();
    for (let v = 0; v < 5000; v++) jb.push(frame(v % 100));
    expect(jb.delayMs).toBeLessThanOrEqual(200);
  });

  it("esvaziou: devolve null, conta o underrun e volta a encher antes de tocar", () => {
    const jb = new JitterBuffer();
    [1, 2, 3].forEach((v) => jb.push(frame(v)));
    jb.pull();
    jb.pull();
    jb.pull();
    expect(jb.pull()).toBeNull();
    expect(jb.underruns).toBe(1);
    jb.push(frame(9));
    expect(jb.pull()).toBeNull(); // reenchendo
    jb.push(frame(9));
    jb.push(frame(9));
    expect(jb.pull()?.[0]).toBe(9);
  });

  it("reserva e teto são configuráveis", () => {
    const jb = new JitterBuffer({ targetMs: 20, maxMs: 40 });
    jb.push(frame(1));
    expect(jb.pull()?.[0]).toBe(1);
    for (let v = 0; v < 10; v++) jb.push(frame(v));
    expect(jb.length).toBe(2);
  });

  it("clear limpa e exige reencher", () => {
    const jb = new JitterBuffer();
    [1, 2, 3].forEach((v) => jb.push(frame(v)));
    jb.clear();
    expect(jb.length).toBe(0);
    expect(jb.pull()).toBeNull();
  });

  it("atraso reflete os quadros enfileirados", () => {
    const jb = new JitterBuffer();
    [1, 2, 3, 4, 5].forEach((v) => jb.push(frame(v)));
    expect(jb.delayMs).toBe(100);
  });
});
