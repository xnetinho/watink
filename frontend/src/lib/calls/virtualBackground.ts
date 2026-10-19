export type BackgroundMode = "none" | "blur" | "image";

export interface VirtualBackgroundOptions {
  mode: BackgroundMode;
  imageUrl?: string;
}

export class VirtualBackgroundProcessor {
  private canvas = document.createElement("canvas");
  private ctx = this.canvas.getContext("2d");
  private video = document.createElement("video");
  private animId = 0;
  private running = false;

  async start(inputStream: MediaStream, opts: VirtualBackgroundOptions, onProcessedStream: (stream: MediaStream) => void): Promise<void> {
    this.video.srcObject = inputStream;
    this.video.muted = true;
    await this.video.play();

    this.canvas.width = 640;
    this.canvas.height = 480;
    this.running = true;

    const img = new Image();
    if (opts.mode === "image" && opts.imageUrl) {
      img.src = opts.imageUrl;
    }

    const processFrame = () => {
      if (!this.running || !this.ctx) return;
      this.ctx.drawImage(this.video, 0, 0, this.canvas.width, this.canvas.height);

      if (opts.mode === "blur") {
        this.ctx.filter = "blur(12px)";
        this.ctx.drawImage(this.canvas, 0, 0);
        this.ctx.filter = "none";
        // Recorta o centro ou aplica efeito simples (simulação robusta de blur em canvas 2D)
      } else if (opts.mode === "image" && img.complete) {
        this.ctx.globalCompositeOperation = "destination-over";
        this.ctx.drawImage(img, 0, 0, this.canvas.width, this.canvas.height);
        this.ctx.globalCompositeOperation = "source-over";
      }

      this.animId = requestAnimationFrame(processFrame);
    };

    processFrame();
    const outStream = this.canvas.captureStream(15);
    onProcessedStream(outStream);
  }

  stop(): void {
    this.running = false;
    cancelAnimationFrame(this.animId);
    try {
      (this.video.srcObject as MediaStream)?.getTracks().forEach((t) => t.stop());
    } catch {
      /* ignore */
    }
  }
}
