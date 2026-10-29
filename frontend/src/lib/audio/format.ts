/** Segundos como m:ss (0:07, 1:05). */
export function formatDuration(seconds: number): string {
  if (!isFinite(seconds) || seconds < 0) return "0:00";
  return `${Math.floor(seconds / 60)}:${Math.floor(seconds % 60).toString().padStart(2, "0")}`;
}
