/* @jsxImportSource react */
import React from "react";
import {
  AlertCircle,
  AlertTriangle,
  CheckCircle2,
  Clock,
  FileText,
  Globe,
  Loader2,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

export type SourceStatus = "pending" | "processing" | "ready" | "error";

export interface KBSource {
  id: number;
  type: string;
  fileName?: string;
  url?: string;
  status: SourceStatus;
  chunkCount?: number;
  lastError?: string;
  objectKey?: string;
}

export interface ChunkView {
  ordinal: number;
  content: string;
  charCount: number;
  model: string;
  dim: number;
  contentHash: string;
}

export function sourceLabel(source: KBSource): string {
  if (source.fileName) return source.fileName;
  if (source.url) return source.url;
  return "Texto";
}

export function SourceIcon({ source }: { source: KBSource }): React.ReactElement {
  if (source.url || source.type === "url") {
    return <Globe className="h-5 w-5 text-muted-foreground" />;
  }
  return <FileText className="h-5 w-5 text-muted-foreground" />;
}

export function StatusBadge({ source }: { source: KBSource }): React.ReactElement {
  switch (source.status) {
    case "ready":
      if (!source.chunkCount) {
        return (
          <Badge className="gap-1 border-transparent bg-[hsl(var(--status-warning-bg))] text-[hsl(var(--status-warning-text))]">
            <AlertTriangle className="h-3.5 w-3.5" />
            Sem conteúdo
          </Badge>
        );
      }
      return (
        <Badge className="gap-1 border-transparent bg-[hsl(var(--status-success-bg))] text-[hsl(var(--status-success-text))]">
          <CheckCircle2 className="h-3.5 w-3.5" />
          Pronto
          <span className="font-normal">· {source.chunkCount} chunks</span>
        </Badge>
      );
    case "processing":
      return (
        <Badge className="gap-1 border-transparent bg-[hsl(var(--status-info-bg))] text-[hsl(var(--status-info-text))]">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          Processando
        </Badge>
      );
    case "error":
      return (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <Badge className="gap-1 border-transparent bg-[hsl(var(--status-error-bg))] text-[hsl(var(--status-error-text))] cursor-help">
                <AlertCircle className="h-3.5 w-3.5" />
                Erro
              </Badge>
            </TooltipTrigger>
            <TooltipContent>
              {source.lastError || "Falha ao processar a fonte"}
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      );
    case "pending":
    default:
      return (
        <Badge className="gap-1 border-transparent bg-[hsl(var(--status-warning-bg))] text-[hsl(var(--status-warning-text))]">
          <Clock className="h-3.5 w-3.5" />
          Aguardando
        </Badge>
      );
  }
}
