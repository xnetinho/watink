import React, { useState } from "react";
import { GitBranch, Plus, Trash2 } from "lucide-react";
import { toast } from "react-toastify";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import api from "../../../services/api";
import { useDeals } from "../../../hooks/useDeals";
import { i18n } from "../../../translate/i18n";

// i18n.t() deste projeto pode devolver null; a tela só quer string.
const tr = (key: string, opts: Record<string, string | number> = {}): string => String(i18n.t(key, opts) ?? "");

interface Pipeline {
  id: number;
  name: string;
  stages: { id: number; name: string }[];
}

interface PipelinesSectionProps {
  ticketId: number;
  contactId: number;
  contactName: string;
}

const PipelinesSection: React.FC<PipelinesSectionProps> = ({ ticketId, contactId, contactName }) => {
  const [dialogOpen, setDialogOpen] = useState(false);
  const [pipelines, setPipelines] = useState<Pipeline[]>([]);
  const [selectedPipeline, setSelectedPipeline] = useState("");
  const [selectedStage, setSelectedStage] = useState("");
  const [saving, setSaving] = useState(false);

  const { deals, refetch } = useDeals({ ticketId });

  const openDialog = async () => {
    try {
      const { data } = await api.get<Pipeline[]>("/pipelines");
      setPipelines(Array.isArray(data) ? data : []);
    } catch {
      setPipelines([]);
    }
    setSelectedPipeline("");
    setSelectedStage("");
    setDialogOpen(true);
  };

  const handleSave = async () => {
    if (!selectedStage) return;
    setSaving(true);
    try {
      await api.post("/deals", {
        name: contactName,
        stageId: Number(selectedStage),
        contactId,
        ticketId,
      });
      toast.success(tr("tickets.details.pipelines.added"));
      setDialogOpen(false);
      refetch();
    } catch {
      toast.error(tr("tickets.details.pipelines.saveError"));
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (dealId: number) => {
    try {
      await api.put(`/deals/${dealId}`, { status: "deleted" });
      refetch();
    } catch {
      toast.error(tr("tickets.details.pipelines.removeError"));
    }
  };

  const currentStages = pipelines.find((p) => String(p.id) === selectedPipeline)?.stages ?? [];

  return (
    <>
      <div className="flex flex-col gap-2">
        <div className="flex items-center justify-between">
          <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide flex items-center gap-1">
            <GitBranch className="w-3 h-3" />
            {tr("tickets.details.pipelines.title")}
          </p>
          <Button
            variant="ghost"
            size="icon"
            className="h-5 w-5"
            onClick={openDialog}
            title={tr("tickets.details.pipelines.add")}
          >
            <Plus className="w-3.5 h-3.5" />
          </Button>
        </div>

        {deals.length === 0 ? (
          <p className="text-xs text-muted-foreground">{tr("tickets.details.pipelines.none")}</p>
        ) : (
          deals.map((deal) => (
            <div
              key={deal.id}
              className="flex items-start justify-between rounded-md border border-border bg-muted/40 px-3 py-2"
            >
              <div className="flex flex-col gap-0.5 min-w-0">
                <span className="text-xs font-semibold truncate">
                  {deal.stage?.pipeline?.name ?? tr("tickets.details.pipelines.fallbackPipeline")}
                </span>
                <span className="text-xs text-muted-foreground truncate">
                  {deal.stage?.name ?? tr("tickets.details.pipelines.fallbackStage")}
                </span>
              </div>
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6 shrink-0 mt-0.5"
                onClick={() => handleDelete(deal.id)}
                title={tr("tickets.details.pipelines.remove")}
              >
                <Trash2 className="w-3.5 h-3.5 text-muted-foreground" />
              </Button>
            </div>
          ))
        )}
      </div>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{tr("tickets.details.pipelines.dialogTitle")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-3 py-2">
            <div className="space-y-1">
              <label className="text-sm font-medium">{tr("tickets.details.pipelines.pipelineLabel")}</label>
              <Select
                value={selectedPipeline}
                onValueChange={(v) => { setSelectedPipeline(v); setSelectedStage(""); }}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder={tr("tickets.details.pipelines.selectPipeline")} />
                </SelectTrigger>
                <SelectContent>
                  {pipelines.map((p) => (
                    <SelectItem key={p.id} value={String(p.id)}>{p.name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            {selectedPipeline && (
              <div className="space-y-1">
                <label className="text-sm font-medium">{tr("tickets.details.pipelines.stageLabel")}</label>
                <Select value={selectedStage} onValueChange={setSelectedStage}>
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder={tr("tickets.details.pipelines.selectStage")} />
                  </SelectTrigger>
                  <SelectContent>
                    {currentStages.map((s) => (
                      <SelectItem key={s.id} value={String(s.id)}>{s.name}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>{tr("tickets.details.pipelines.cancel")}</Button>
            <Button
              onClick={handleSave}
              disabled={!selectedPipeline || !selectedStage || saving}
            >
              {saving ? tr("tickets.details.pipelines.saving") : tr("tickets.details.pipelines.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
};

export default PipelinesSection;
