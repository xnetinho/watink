/* @jsxImportSource react */
import React, { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router";
import { toast } from "react-toastify";
import { Brain, Library, Loader2, Plus, Trash2 } from "lucide-react";

import api from "../../services/api";
import toastError from "../../errors/toastError";
import ConfirmationModal from "../../components/ConfirmationModal";

import {
  PageContainer,
  PageHeader,
  PageContent,
} from "@/components/ui/page-layout";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

import { i18n } from "../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface KnowledgeBase {
  id: number;
  name: string;
  description?: string;
}

const KnowledgeBase: React.FC = () => {
  const navigate = useNavigate();

  const [bases, setBases] = useState<KnowledgeBase[]>([]);
  const [loading, setLoading] = useState(true);

  const [dialogOpen, setDialogOpen] = useState(false);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [saving, setSaving] = useState(false);

  const [confirmOpen, setConfirmOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<KnowledgeBase | null>(null);

  const fetchBases = useCallback(async () => {
    setLoading(true);
    try {
      const { data } = await api.get<KnowledgeBase[]>("/knowledge-bases");
      setBases(Array.isArray(data) ? data : []);
    } catch (err) {
      toastError(err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchBases();
  }, [fetchBases]);

  const handleCreate = async () => {
    if (!name.trim()) return;
    setSaving(true);
    try {
      await api.post("/knowledge-bases", {
        name: name.trim(),
        description: description.trim(),
      });
      toast.success(tr("uiText.kb.created"));
      setDialogOpen(false);
      setName("");
      setDescription("");
      await fetchBases();
    } catch (err) {
      toastError(err);
    } finally {
      setSaving(false);
    }
  };

  const handleRequestDelete = (
    e: React.MouseEvent,
    base: KnowledgeBase
  ): void => {
    e.stopPropagation();
    setDeleteTarget(base);
    setConfirmOpen(true);
  };

  const handleConfirmDelete = async () => {
    if (!deleteTarget) return;
    try {
      await api.delete(`/knowledge-bases/${deleteTarget.id}`);
      toast.success(tr("uiText.kb.removed"));
      await fetchBases();
    } catch (err) {
      toastError(err);
    } finally {
      setDeleteTarget(null);
    }
  };

  return (
    <PageContainer>
      <ConfirmationModal
        title={
          deleteTarget ? tr("uiText.kb.deleteTitle", { name: deleteTarget.name }) : tr("uiText.kb.deleteBaseTitle")
        }
        open={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        onConfirm={handleConfirmDelete}
      >
        {tr("uiText.kb.deleteConfirm")}
      </ConfirmationModal>

      <PageHeader
        title={tr("uiText.kb.title")}
        description={tr("uiText.kb.subtitle")}
      >
        <Button onClick={() => setDialogOpen(true)}>
          <Plus className="h-4 w-4" />
          {tr("uiText.kb.newShort")}
        </Button>
      </PageHeader>

      <PageContent>
        {loading ? (
          <div className="flex items-center justify-center py-20 text-muted-foreground">
            <Loader2 className="h-6 w-6 animate-spin" />
          </div>
        ) : bases.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-20 text-center gap-4">
            <div className="h-14 w-14 rounded-2xl bg-muted flex items-center justify-center">
              <Library className="h-7 w-7 text-muted-foreground" />
            </div>
            <div>
              <p className="font-semibold text-foreground">
                {tr("uiText.kb.none")}
              </p>
              <p className="text-sm text-muted-foreground mt-1">
                {tr("uiText.kb.noneHint")}
              </p>
            </div>
            <Button onClick={() => setDialogOpen(true)}>
              <Plus className="h-4 w-4" />
              Criar Base
            </Button>
          </div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {bases.map((base) => (
              <Card
                key={base.id}
                role="button"
                tabIndex={0}
                onClick={() => navigate(`/knowledge-bases/${base.id}`)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault();
                    navigate(`/knowledge-bases/${base.id}`);
                  }
                }}
                className="cursor-pointer transition-transform hover:-translate-y-0.5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                <CardHeader className="flex flex-row items-start justify-between gap-3 space-y-0">
                  <div className="flex items-start gap-3 min-w-0">
                    <div className="h-10 w-10 rounded-xl bg-[hsl(var(--primary))]/10 flex items-center justify-center shrink-0">
                      <Brain className="h-5 w-5 text-[hsl(var(--primary))]" />
                    </div>
                    <div className="min-w-0">
                      <CardTitle className="text-base truncate">
                        {base.name}
                      </CardTitle>
                      <CardDescription className="mt-1 line-clamp-2">
                        {base.description || tr("uiText.activities.noDescription")}
                      </CardDescription>
                    </div>
                  </div>
                  <Button
                    variant="ghost"
                    size="icon"
                    aria-label={tr("uiText.kb.deleteBase")}
                    onClick={(e) => handleRequestDelete(e, base)}
                    className="shrink-0 text-muted-foreground hover:text-[hsl(var(--destructive))]"
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </CardHeader>
                <CardContent />
              </Card>
            ))}
          </div>
        )}
      </PageContent>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{tr("uiText.kb.newBase")}</DialogTitle>
          </DialogHeader>
          <div className="flex flex-col gap-4 py-2">
            <div className="flex flex-col gap-2">
              <Label htmlFor="kb-name">{tr("uiText.common.name")}</Label>
              <Input
                id="kb-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder={tr("uiText.kb.namePlaceholder")}
                autoFocus
              />
            </div>
            <div className="flex flex-col gap-2">
              <Label htmlFor="kb-description">{tr("uiText.common.description")}</Label>
              <Textarea
                id="kb-description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder={tr("uiText.kb.descPlaceholder")}
                rows={3}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              {tr("uiText.common.cancel")}
            </Button>
            <Button onClick={handleCreate} disabled={!name.trim() || saving}>
              {saving && <Loader2 className="h-4 w-4 animate-spin" />}
              {tr("uiText.common.create")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </PageContainer>
  );
};

export default KnowledgeBase;
