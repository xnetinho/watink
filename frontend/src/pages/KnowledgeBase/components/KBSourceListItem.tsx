/* @jsxImportSource react */
import React from "react";
import {
  AlertCircle,
  AlertTriangle,
  ChevronDown,
  Loader2,
  RotateCw,
  Trash2,
} from "lucide-react";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import {
  type KBSource,
  type ChunkView,
  sourceLabel,
  SourceIcon,
  StatusBadge,
} from "./KBSourceBadges";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface KBSourceListItemProps {
  source: KBSource;
  isExpandable: boolean;
  isOpen: boolean;
  chunks?: ChunkView[];
  loadingChunks: number | null;
  onToggleChunks: (source: KBSource) => void;
  onReingest: (source: KBSource) => void;
  onRequestDelete: (source: KBSource) => void;
}

export const KBSourceListItem: React.FC<KBSourceListItemProps> = ({
  source,
  isExpandable,
  isOpen,
  chunks,
  loadingChunks,
  onToggleChunks,
  onReingest,
  onRequestDelete,
}) => {
  return (
    <div className="flex flex-col">
      <Card className="flex items-center gap-4 px-4 py-3">
        <div className="h-10 w-10 rounded-xl bg-muted flex items-center justify-center shrink-0">
          <SourceIcon source={source} />
        </div>
        <div className="min-w-0 flex-1">
          <p className="font-medium text-foreground truncate">
            {sourceLabel(source)}
          </p>
          {source.url ? (
            <a
              href={source.url}
              target="_blank"
              rel="noreferrer"
              title={source.url}
              className="block truncate text-xs text-muted-foreground hover:text-foreground hover:underline"
            >
              {source.url}
            </a>
          ) : (
            <p className="text-xs uppercase tracking-wide text-muted-foreground">
              {source.type}
            </p>
          )}
        </div>
        <StatusBadge source={source} />
        {isExpandable && (
          <Button
            variant="ghost"
            size="icon"
            aria-label={isOpen ? "Recolher chunks" : "Ver chunks"}
            onClick={() => onToggleChunks(source)}
            className="shrink-0 text-muted-foreground"
          >
            <ChevronDown
              className={`h-4 w-4 transition-transform ${
                isOpen ? "rotate-180" : ""
              }`}
            />
          </Button>
        )}
        {source.type !== "text" && (
          <Button
            variant="ghost"
            size="icon"
            aria-label="Reprocessar fonte"
            title={tr("uiText.kb.reprocess")}
            onClick={() => onReingest(source)}
            className="shrink-0 text-muted-foreground"
          >
            <RotateCw className="h-4 w-4" />
          </Button>
        )}
        <Button
          variant="ghost"
          size="icon"
          aria-label="Remover fonte"
          onClick={() => onRequestDelete(source)}
          className="shrink-0 text-muted-foreground hover:text-[hsl(var(--destructive))]"
        >
          <Trash2 className="h-4 w-4" />
        </Button>
      </Card>

      {source.status === "error" && (
        <div className="mx-2 mt-1 flex items-start gap-2 rounded-xl border border-[hsl(var(--status-error-text))]/30 bg-[hsl(var(--status-error-bg))]/40 p-3">
          <AlertCircle className="mt-0.5 h-4 w-4 shrink-0 text-[hsl(var(--status-error-text))]" />
          <div className="min-w-0">
            <p className="text-sm font-medium text-[hsl(var(--status-error-text))]">
              {tr("uiText.kb.processFailThis")}
            </p>
            <p className="break-words text-xs text-muted-foreground">
              {source.lastError || tr("uiText.kb.ingestUnknown")}
            </p>
            {source.type !== "text" && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => onReingest(source)}
                className="mt-2 gap-1.5"
              >
                <RotateCw className="h-3.5 w-3.5" />
                Tentar novamente
              </Button>
            )}
          </div>
        </div>
      )}

      {source.status === "ready" && !source.chunkCount && (
        <div className="mx-2 mt-1 flex items-start gap-2 rounded-xl border border-[hsl(var(--status-warning-text))]/30 bg-[hsl(var(--status-warning-bg))]/40 p-3">
          <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-[hsl(var(--status-warning-text))]" />
          <div className="min-w-0">
            <p className="text-sm font-medium text-[hsl(var(--status-warning-text))]">
              {tr("uiText.kb.noExtracted")}
            </p>
            <p className="break-words text-xs text-muted-foreground">
              {tr("uiText.kb.noExtractedHint")}
            </p>
          </div>
        </div>
      )}

      {(source.status === "processing" || source.status === "pending") && (
        <div className="mx-2 mt-1 flex items-center gap-2 rounded-xl border bg-muted/30 px-3 py-2 text-xs text-muted-foreground">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          {source.status === "processing"
            ? tr("uiText.kb.processing")
            : tr("uiText.kb.queued")}
        </div>
      )}

      {isOpen && (
        <div className="mx-2 mt-1 rounded-xl border bg-muted/30 p-3">
          {loadingChunks === source.id ? (
            <div className="flex items-center justify-center py-4 text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
            </div>
          ) : !chunks || chunks.length === 0 ? (
            <p className="py-3 text-center text-sm text-muted-foreground">
              {tr("uiText.kb.noChunk")}
            </p>
          ) : (
            <div className="flex flex-col gap-2">
              <p className="text-xs text-muted-foreground">
                {chunks.length} chunk(s) · modelo {chunks[0].model} ·{" "}
                {chunks[0].dim} dimensões
              </p>
              {chunks.map((ch) => (
                <div
                  key={ch.ordinal}
                  className="rounded-lg border bg-background p-2.5"
                >
                  <div className="mb-1 flex items-center gap-2 text-[11px] text-muted-foreground">
                    <span className="font-medium">#{ch.ordinal}</span>
                    <span>· {ch.charCount} caracteres</span>
                  </div>
                  <p className="whitespace-pre-wrap text-sm text-foreground">
                    {ch.content}
                  </p>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
};
