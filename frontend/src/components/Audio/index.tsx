import React, { useRef, useEffect, useState, useCallback } from "react";
import { Play, Pause, Mic } from "lucide-react";
import { Avatar } from "../ui/avatar";
import { cn } from "../../lib/utils";
import { i18n } from "../../translate/i18n";
import { formatDuration } from "../../lib/audio/format";
import { WAVEFORM_BARS, loadWaveform, restingWaveform } from "../../lib/audio/waveform";

const LS_RATE = "audioMessageRate";

interface AudioProps {
  url: string;
  mimetype?: string;
  /** Quem enviou: a foto (ou as iniciais) aparece ao lado da onda, como no WhatsApp Web. */
  senderName?: string;
  senderPicUrl?: string | null;
  /** Mensagem de voz (PTT): mostra o microfone sobre o avatar. Arquivo de áudio fica só com o avatar. */
  voice?: boolean;
}

const formatTime = formatDuration;

const RATES: number[] = [1, 1.5, 2, 0.5];

const tr = (key: string): string => String(i18n.t(key) ?? "");

const Audio: React.FC<AudioProps> = ({ url, mimetype, senderName, senderPicUrl, voice = false }) => {
  const audioRef = useRef<HTMLAudioElement>(null);
  const waveRef = useRef<HTMLDivElement>(null);
  const draggingRef = useRef(false);

  const [playing, setPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [bars, setBars] = useState<number[]>(() => restingWaveform());
  const [rateIdx, setRateIdx] = useState(() => {
    const saved = parseFloat(localStorage.getItem(LS_RATE) ?? "1");
    return RATES.indexOf(saved) !== -1 ? RATES.indexOf(saved) : 0;
  });

  const rate = RATES[rateIdx];
  const ratio = duration > 0 ? Math.min(1, currentTime / duration) : 0;

  useEffect(() => {
    const el = audioRef.current;
    if (!el) return;
    el.playbackRate = rate;
    localStorage.setItem(LS_RATE, String(rate));
  }, [rate]);

  // A onda vem do próprio arquivo (o Watink não guarda a do WhatsApp). Falhou ou é grande demais: fica a onda de repouso.
  useEffect(() => {
    let alive = true;
    void loadWaveform(url, WAVEFORM_BARS).then((peaks) => {
      if (alive && peaks) setBars(peaks);
    });
    return () => {
      alive = false;
    };
  }, [url]);

  // `timeupdate` dispara ~4x/s: a bolinha andaria aos saltos. Enquanto toca, lê a posição a cada quadro da tela.
  useEffect(() => {
    const el = audioRef.current;
    if (!playing || !el) return;
    let raf = 0;
    const tick = () => {
      if (!draggingRef.current) setCurrentTime(el.currentTime);
      raf = requestAnimationFrame(tick);
    };
    raf = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(raf);
  }, [playing]);

  useEffect(() => {
    const el = audioRef.current;
    if (!el) return;

    const onTimeUpdate = () => {
      if (!draggingRef.current) setCurrentTime(el.currentTime);
    };
    const onDurationChange = () => setDuration(el.duration);
    const onPlay = () => setPlaying(true);
    const onPause = () => setPlaying(false);
    const onEnded = () => {
      setPlaying(false);
      setCurrentTime(0);
    };

    el.addEventListener("timeupdate", onTimeUpdate);
    el.addEventListener("durationchange", onDurationChange);
    el.addEventListener("loadedmetadata", onDurationChange);
    el.addEventListener("play", onPlay);
    el.addEventListener("pause", onPause);
    el.addEventListener("ended", onEnded);

    return () => {
      el.removeEventListener("timeupdate", onTimeUpdate);
      el.removeEventListener("durationchange", onDurationChange);
      el.removeEventListener("loadedmetadata", onDurationChange);
      el.removeEventListener("play", onPlay);
      el.removeEventListener("pause", onPause);
      el.removeEventListener("ended", onEnded);
    };
  }, []);

  const togglePlay = useCallback(() => {
    const el = audioRef.current;
    if (!el) return;
    if (playing) el.pause();
    else el.play().catch(() => {});
  }, [playing]);

  const seekTo = useCallback(
    (clientX: number) => {
      const el = audioRef.current;
      const bar = waveRef.current;
      if (!el || !bar || !duration) return;
      const rect = bar.getBoundingClientRect();
      if (rect.width <= 0) return;
      const r = Math.max(0, Math.min(1, (clientX - rect.left) / rect.width));
      el.currentTime = r * duration;
      setCurrentTime(r * duration);
    },
    [duration],
  );

  // Arrastar a onda "raspa" o áudio, como no WhatsApp Web: o ponteiro fica preso à onda até soltar.
  const onPointerDown = (e: React.PointerEvent<HTMLDivElement>) => {
    draggingRef.current = true;
    e.currentTarget.setPointerCapture?.(e.pointerId);
    seekTo(e.clientX);
  };
  const onPointerMove = (e: React.PointerEvent<HTMLDivElement>) => {
    if (draggingRef.current) seekTo(e.clientX);
  };
  const endDrag = (e: React.PointerEvent<HTMLDivElement>) => {
    draggingRef.current = false;
    e.currentTarget.releasePointerCapture?.(e.pointerId);
  };

  // Teclado: setas movem 5 s, Home/End vão às pontas (a onda é um controle deslizante).
  const onKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
    const el = audioRef.current;
    if (!el || !duration) return;
    const step = e.key === "ArrowRight" ? 5 : e.key === "ArrowLeft" ? -5 : 0;
    if (step) el.currentTime = Math.max(0, Math.min(duration, el.currentTime + step));
    else if (e.key === "Home") el.currentTime = 0;
    else if (e.key === "End") el.currentTime = duration;
    else return;
    e.preventDefault();
    setCurrentTime(el.currentTime);
  };

  const cycleRate = useCallback(() => {
    setRateIdx((i) => (i + 1) % RATES.length);
  }, []);

  const resolvedMime = mimetype ?? "audio/ogg";
  const playedBars = Math.round(ratio * bars.length);

  return (
    <div
      className="flex w-full min-w-[260px] max-w-[340px] items-center gap-2.5 py-1.5"
      data-testid="audio-player"
      data-voice={voice ? "true" : "false"}
    >
      <audio ref={audioRef} preload="metadata">
        <source src={url} type={resolvedMime} />
        <source src={url} />
      </audio>

      <div className="relative shrink-0" data-testid="audio-avatar">
        <Avatar size="lg" src={senderPicUrl} name={senderName} />
        {voice && (
          <span
            className="absolute -bottom-0.5 -right-0.5 flex h-5 w-5 items-center justify-center rounded-full bg-background text-primary shadow"
            data-testid="audio-voice-badge"
          >
            <Mic size={12} aria-hidden="true" />
          </span>
        )}
      </div>

      <button
        type="button"
        onClick={togglePlay}
        className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-muted-foreground transition-colors hover:text-foreground"
        aria-label={playing ? tr("chat.audioPlayer.pause") : tr("chat.audioPlayer.play")}
      >
        {playing ? <Pause size={22} fill="currentColor" /> : <Play size={22} fill="currentColor" />}
      </button>

      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <div
          ref={waveRef}
          role="slider"
          tabIndex={0}
          aria-label={tr("chat.audioPlayer.seek")}
          aria-valuemin={0}
          aria-valuemax={Math.round(duration)}
          aria-valuenow={Math.round(currentTime)}
          aria-valuetext={`${formatTime(currentTime)} / ${formatTime(duration)}`}
          onPointerDown={onPointerDown}
          onPointerMove={onPointerMove}
          onPointerUp={endDrag}
          onPointerCancel={endDrag}
          onKeyDown={onKeyDown}
          className="group relative flex h-8 cursor-pointer touch-none select-none items-center gap-[2px] rounded-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          data-testid="audio-waveform"
        >
          {bars.map((v, i) => (
            <span
              key={i}
              className={cn("w-[3px] flex-1 rounded-full", i < playedBars ? "bg-primary" : "bg-muted-foreground/40")}
              style={{ height: `${Math.max(12, Math.round(v * 100))}%` }}
              data-played={i < playedBars ? "true" : "false"}
            />
          ))}
          <span
            className="pointer-events-none absolute top-1/2 h-3 w-3 -translate-x-1/2 -translate-y-1/2 rounded-full bg-primary shadow"
            style={{ left: `${ratio * 100}%` }}
            data-testid="audio-knob"
          />
        </div>

        <div className="flex items-center justify-between text-[11px] tabular-nums text-muted-foreground">
          <span data-testid="audio-time">{playing || currentTime > 0 ? formatTime(currentTime) : formatTime(duration)}</span>
          <button
            type="button"
            onClick={cycleRate}
            className="rounded-full bg-muted px-1.5 py-px text-[10px] font-medium transition-colors hover:bg-accent"
            aria-label={`${tr("chat.audioPlayer.speed")} ${rate}x`}
          >
            {rate}x
          </button>
        </div>
      </div>
    </div>
  );
};

Audio.displayName = "Audio";

export default Audio;
