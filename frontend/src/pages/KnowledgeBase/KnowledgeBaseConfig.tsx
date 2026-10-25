/* @jsxImportSource react */
import React, { useCallback, useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { toast } from "react-toastify";
import {
  ArrowLeft,
  FlaskConical,
  Loader2,
  Plus,
  Upload,
} from "lucide-react";

import api from "../../services/api";
import toastError from "../../errors/toastError";
import { subscribeToSocket } from "../../services/sse-client";
import ConfirmationModal from "../../components/ConfirmationModal";
import RetrievalPlayground from "./RetrievalPlayground";

import {
  PageContainer,
  PageHeader,
  PageContent,
} from "@/components/ui/page-layout";
import { Button } from "@/components/ui/button";

import {
  type KBSource,
  type ChunkView,
  type SourceStatus,
} from "./components/KBSourceBadges";
import { AddSourceDialog } from "./components/AddSourceDialog";
import { KBSourceListItem } from "./components/KBSourceListItem";

interface KnowledgeBase {
  id: number;
  name: string;
  description?: string;
  sources?: KBSource[];
}

interface KnowledgeSourceEvent {
  action?: string;
  sourceId: number;
  status: SourceStatus;
  chunkCount?: number;
  lastError?: string;
}

const KnowledgeBaseConfig: React.FC = () => {
  const { knowledgeBaseId } = useParams<{ knowledgeBaseId: string }>();
  const navigate = useNavigate();

  const [kb, setKb] = useState<KnowledgeBase | null>(null);
  const [sources, setSources] = useState<KBSource[]>([]);
  const [loading, setLoading] = useState(true);

  const [dialogOpen, setDialogOpen] = useState(false);
  const [tab, setTab] = useState<"text" | "file" | "url">("text");
  const [textName, setTextName] = useState("");
  const [textContent, setTextContent] = useState("");
  const [urlName, setUrlName] = useState("");
  const [urlValue, setUrlValue] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const [confirmOpen, setConfirmOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<KBSource | null>(null);

  const [expandedId, setExpandedId] = useState<number | null>(null);
  const [chunksBySource, setChunksBySource] = useState<Record<number, ChunkView[]>>({});
  const [loadingChunks, setLoadingChunks] = useState<number | null>(null);
  const [playgroundOpen, setPlaygroundOpen] = useState(false);

  const toggleChunks = async (source: KBSource): Promise<void> => {
    if (expandedId === source.id) {
      setExpandedId(null);
      return;
    }
    setExpandedId(source.id);
    if (chunksBySource[source.id]) return;

    setLoadingChunks(source.id);
    try {
      const { data } = await api.get<{ chunks: ChunkView[] }>(
        `/knowledge-bases/${knowledgeBaseId}/sources/${source.id}/chunks`
      );
      setChunksBySource((prev) => ({ ...prev, [source.id]: data.chunks || [] }));
    } catch (err) {
      toastError(err);
    } finally {
      setLoadingChunks(null);
    }
  };

  const fetchBase = useCallback(async () => {
    if (!knowledgeBaseId) return;
    try {
      setLoading(true);
      const { data } = await api.get<KnowledgeBase>(
        `/knowledge-bases/${knowledgeBaseId}`
      );
      setKb(data);
      setSources(data.sources || []);
    } catch (err) {
      toastError(err);
    } finally {
      setLoading(false);
    }
  }, [knowledgeBaseId]);

  useEffect(() => {
    fetchBase();
  }, [fetchBase]);

  useEffect(() => {
    const cleanup = subscribeToSocket({
      knowledgeSource: (payload: KnowledgeSourceEvent) => {
        if (!payload || !payload.sourceId) return;
        setSources((prev) =>
          prev.map((s) => {
            if (s.id !== payload.sourceId) return s;
            return {
              ...s,
              status: payload.status,
              chunkCount:
                payload.chunkCount !== undefined
                  ? payload.chunkCount
                  : s.chunkCount,
              lastError:
                typeof payload.lastError === "string"
                  ? payload.lastError
                  : s.lastError,
            };
          })
        );
      },
    });
    return cleanup;
  }, []);

  const resetDialog = (): void => {
    setTextName("");
    setTextContent("");
    setUrlName("");
    setUrlValue("");
    setFile(null);
    setTab("text");
  };

  const handleAddSource = async () => {
    if (!knowledgeBaseId) return;

    const formData = new FormData();
    if (tab === "text") {
      formData.append("type", "text");
      if (textName.trim()) formData.append("name", textName.trim());
      formData.append("content", textContent);
    } else if (tab === "url") {
      formData.append("type", "url");
      formData.append("url", urlValue.trim());
      if (urlName.trim()) formData.append("name", urlName.trim());
    } else if (tab === "file") {
      if (!file) return;
      formData.append("type", "file");
      formData.append("file", file);
    }

    try {
      setSubmitting(true);
      const { data } = await api.post<KBSource>(
        `/knowledge-bases/${knowledgeBaseId}/sources`,
        formData
      );
      toast.success("Fonte enviada com sucesso! A indexação foi iniciada.");
      setSources((prev) => [data, ...prev]);
      setDialogOpen(false);
      resetDialog();
    } catch (err) {
      toastError(err);
    } finally {
      setSubmitting(false);
    }
  };

  const handleReingest = async (source: KBSource): Promise<void> => {
    if (!knowledgeBaseId) return;
    try {
      await api.post(
        `/knowledge-bases/${knowledgeBaseId}/sources/${source.id}/reingest`
      );
      toast.info("Reingestão solicitada.");
      setSources((prev) =>
        prev.map((s) =>
          s.id === source.id
            ? { ...s, status: "pending", lastError: undefined }
            : s
        )
      );
    } catch (err) {
      toastError(err);
    }
  };

  const handleRequestDelete = (source: KBSource): void => {
    setDeleteTarget(source);
    setConfirmOpen(true);
  };

  const handleConfirmDelete = async () => {
    if (!knowledgeBaseId || !deleteTarget) return;
    try {
      await api.delete(
        `/knowledge-bases/${knowledgeBaseId}/sources/${deleteTarget.id}`
      );
      toast.success("Fonte removida com sucesso.");
      setSources((prev) => prev.filter((s) => s.id !== deleteTarget.id));
      setConfirmOpen(false);
      setDeleteTarget(null);
    } catch (err) {
      toastError(err);
    }
  };

  const canSubmit =
    tab === "text"
      ? textContent.trim().length > 0
      : tab === "url"
      ? urlValue.trim().length > 0
      : file !== null;

  return (
    <PageContainer>
      <PageHeader
        title={kb?.name ? `Base: ${kb.name}` : "Base de Conhecimento"}
        description={
          kb?.description ||
          "Adicione fontes para que a IA aprenda a responder sobre sua empresa"
        }
      >
        <div className="flex gap-2">
          <Button
            variant="outline"
            onClick={() => navigate("/knowledge-bases")}
            className="gap-2"
          >
            <ArrowLeft className="h-4 w-4" />
            Voltar
          </Button>
          <Button
            variant="outline"
            onClick={() => setPlaygroundOpen(true)}
            disabled={sources.filter((s) => s.status === "ready").length === 0}
            className="gap-2"
          >
            <FlaskConical className="h-4 w-4" />
            Testar busca
          </Button>
          <Button onClick={() => setDialogOpen(true)} className="gap-2">
            <Plus className="h-4 w-4" />
            Adicionar fonte
          </Button>
        </div>
      </PageHeader>

      <PageContent>
        {loading ? (
          <div className="flex items-center justify-center py-20 text-muted-foreground">
            <Loader2 className="h-6 w-6 animate-spin" />
          </div>
        ) : sources.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-20 text-center gap-4">
            <div className="h-14 w-14 rounded-2xl bg-muted flex items-center justify-center">
              <Upload className="h-7 w-7 text-muted-foreground" />
            </div>
            <div>
              <p className="font-semibold text-foreground">
                Nenhuma fonte adicionada
              </p>
              <p className="text-sm text-muted-foreground mt-1">
                Adicione textos ou arquivos para a IA aprender com eles
              </p>
            </div>
            <Button onClick={() => setDialogOpen(true)}>
              <Plus className="h-4 w-4" />
              Adicionar fonte
            </Button>
          </div>
        ) : (
          <div className="flex flex-col gap-3">
            {sources.map((source) => {
              const isExpandable =
                source.status === "ready" && !!source.chunkCount;
              const isOpen = expandedId === source.id;
              const chunks = chunksBySource[source.id];
              return (
                <KBSourceListItem
                  key={source.id}
                  source={source}
                  isExpandable={isExpandable}
                  isOpen={isOpen}
                  chunks={chunks}
                  loadingChunks={loadingChunks}
                  onToggleChunks={toggleChunks}
                  onReingest={handleReingest}
                  onRequestDelete={handleRequestDelete}
                />
              );
            })}
          </div>
        )}
      </PageContent>

      <AddSourceDialog
        open={dialogOpen}
        onOpenChange={(open) => {
          setDialogOpen(open);
          if (!open) resetDialog();
        }}
        tab={tab}
        onTabChange={setTab}
        textName={textName}
        onTextNameChange={setTextName}
        textContent={textContent}
        onTextContentChange={setTextContent}
        urlName={urlName}
        onUrlNameChange={setUrlName}
        urlValue={urlValue}
        onUrlValueChange={setUrlValue}
        file={file}
        onFileChange={setFile}
        canSubmit={canSubmit}
        submitting={submitting}
        onSubmit={handleAddSource}
      />

      <ConfirmationModal
        title="Remover fonte de conhecimento"
        open={confirmOpen}
        onClose={setConfirmOpen}
        onConfirm={handleConfirmDelete}
      >
        Tem certeza que deseja remover esta fonte? Os chunks e embeddings
        armazenados serão excluídos permanentemente.
      </ConfirmationModal>

      {knowledgeBaseId && (
        <RetrievalPlayground
          open={playgroundOpen}
          onOpenChange={setPlaygroundOpen}
          knowledgeBaseId={knowledgeBaseId}
        />
      )}
    </PageContainer>
  );
};

export default KnowledgeBaseConfig;
