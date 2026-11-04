export type BackgroundMode = "none" | "blur_slight" | "blur_full" | "image";

export interface VirtualBackgroundConfig {
  mode: BackgroundMode;
  imageUrl?: string;
}

export const PRESET_BACKGROUNDS = [
  {
    id: "office",
    name: "uiText.calls.backgroundNames.office",
    url: "https://images.unsplash.com/photo-1497366216548-37526070297c?auto=format&fit=crop&w=640&q=80",
  },
  {
    id: "meeting",
    name: "uiText.calls.backgroundNames.meeting",
    url: "https://images.unsplash.com/photo-1497215728101-856f4ea42174?auto=format&fit=crop&w=640&q=80",
  },
  {
    id: "books",
    name: "uiText.calls.backgroundNames.books",
    url: "https://images.unsplash.com/photo-1507842229458-5776306e6e24?auto=format&fit=crop&w=640&q=80",
  },
] as const;

export class VirtualBackgroundProcessor {
  private canvas: HTMLCanvasElement;
  private ctx: CanvasRenderingContext2D | null = null;
  private video: HTMLVideoElement;
  private animId = 0;
  private running = false;
  private bgImage: HTMLImageElement | null = null;
  private currentConfig: VirtualBackgroundConfig = { mode: "none" };

  constructor() {
    this.canvas = document.createElement("canvas");
    this.canvas.width = 640;
    this.canvas.height = 480;
    this.ctx = this.canvas.getContext("2d");
    this.video = document.createElement("video");
    this.video.muted = true;
    this.video.playsInline = true;
  }

  async start(
    inputStream: MediaStream,
    config: VirtualBackgroundConfig,
  ): Promise<MediaStream> {
    this.currentConfig = config;

    // Se o modo for "none", retorna o stream original diretamente sem nenhum overhead
    if (config.mode === "none") {
      this.stop();
      return inputStream;
    }

    if (config.mode === "image" && config.imageUrl) {
      this.bgImage = new Image();
      this.bgImage.crossOrigin = "anonymous";
      this.bgImage.src = config.imageUrl;
    } else {
      this.bgImage = null;
    }

    this.video.srcObject = inputStream;
    try {
      await this.video.play();
    } catch {
      /* ignore autoplay error */
    }

    this.running = true;
    this.loop();

    return this.canvas.captureStream(15);
  }

  updateConfig(config: VirtualBackgroundConfig): void {
    this.currentConfig = config;
    if (config.mode === "image" && config.imageUrl) {
      const img = new Image();
      img.crossOrigin = "anonymous";
      img.src = config.imageUrl;
      this.bgImage = img;
    } else {
      this.bgImage = null;
    }
  }

  private loop = (): void => {
    if (!this.running || !this.ctx) return;
    const { width, height } = this.canvas;
    const ctx = this.ctx;
    const mode = this.currentConfig.mode;

    if (this.video.readyState >= 2) {
      if (mode === "blur_slight" || mode === "blur_full") {
        const blurAmount = mode === "blur_slight" ? 8 : 16;
        // 1. Desenha o fundo borrado
        ctx.save();
        ctx.filter = `blur(${blurAmount}px)`;
        ctx.drawImage(this.video, -10, -10, width + 20, height + 20);
        ctx.restore();

        // 2. Sobrepõe o centro (área do operador) em foco com transição suave
        ctx.save();
        ctx.beginPath();
        const cx = width / 2;
        const cy = height * 0.52;
        const rx = width * 0.32;
        const ry = height * 0.44;
        ctx.ellipse(cx, cy, rx, ry, 0, 0, Math.PI * 2);
        ctx.clip();
        ctx.drawImage(this.video, 0, 0, width, height);
        ctx.restore();
      } else if (mode === "image" && this.bgImage && this.bgImage.complete && this.bgImage.naturalWidth > 0) {
        // 1. Desenha a imagem de fundo corporativa
        ctx.drawImage(this.bgImage, 0, 0, width, height);

        // 2. Sobrepõe o operador no centro
        ctx.save();
        ctx.beginPath();
        const cx = width / 2;
        const cy = height * 0.52;
        const rx = width * 0.32;
        const ry = height * 0.44;
        ctx.ellipse(cx, cy, rx, ry, 0, 0, Math.PI * 2);
        ctx.clip();
        ctx.drawImage(this.video, 0, 0, width, height);
        ctx.restore();
      } else {
        ctx.drawImage(this.video, 0, 0, width, height);
      }
    }

    this.animId = requestAnimationFrame(this.loop);
  };

  stop(): void {
    this.running = false;
    if (this.animId) {
      cancelAnimationFrame(this.animId);
      this.animId = 0;
    }
    try {
      this.video.pause();
      this.video.srcObject = null;
    } catch {
      /* ignore */
    }
  }
}
