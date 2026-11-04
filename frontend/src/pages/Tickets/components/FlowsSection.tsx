import React, { useState, useEffect, useCallback } from "react";
import { Play, Zap, Trash2 } from "lucide-react";
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
import { i18n } from "../../../translate/i18n";

// i18n.t() deste projeto pode devolver null; a tela só quer string.
const tr = (key: string, opts: Record<string, string | number> = {}): string => String(i18n.t(key, opts) ?? "");

interface Flow {
  id: number;
  name: string;
  active: boolean;
  triggerType?: string;
}

interface FlowRun {
  id: string;
  flowId: number;
  status: string;
  flow?: { id: number; name: string };
}

// Rótulo do status da execução: vem dos arquivos de idioma (tickets.details.flows.status.*).
const statusLabel = (status: string): string => {
  const key = `tickets.details.flows.status.${status}`;
  const t = tr(key);
  return t === key ? status : t;
};

const ACTIVE_STATUSES = ["running", "waiting_message", "waiting_until", "waiting_event"];

interface FlowsSectionProps {
  contactId: number;
}

const FlowsSection: React.FC<FlowsSectionProps> = ({ contactId }) => {
  const [dialogOpen, setDialogOpen] = useState(false);
  const [flows, setFlows] = useState<Flow[]>([]);
  const [flowRuns, setFlowRuns] = useState<FlowRun[]>([]);
  const [selectedFlow, setSelectedFlow] = useState("");
  const [starting, setStarting] = useState(false);

  const fetchFlowRuns = useCallback(async () => {
    try {
      const { data } = await api.get<FlowRun[]>("/flowruns", {
        params: { contactId, status: "active" },
      });
      setFlowRuns(Array.isArray(data) ? data : []);
    } catch {
      setFlowRuns([]);
    }
  }, [contactId]);

  useEffect(() => {
    if (contactId) fetchFlowRuns();
  }, [contactId, fetchFlowRuns]);

  const openDialog = async () => {
    try {
      const { data } = await api.get<Flow[]>("/flows");
      setFlows(Array.isArray(data) ? data.filter((f) => f.active) : []);
    } catch {
      setFlows([]);
    }
    setSelectedFlow("");
    setDialogOpen(true);
  };

  const handleStart = async () => {
    if (!selectedFlow) return;
    setStarting(true);
    try {
      await api.post(`/flows/${selectedFlow}/run`, { contactId: String(contactId) });
      toast.success(tr("tickets.details.flows.started"));
      setDialogOpen(false);
      fetchFlowRuns();
    } catch (err: unknown) {
      // 409 = o contato já tem uma execução ativa deste fluxo. Decide pelo código HTTP, nunca pelo texto da resposta.
      const status = (err as { response?: { status?: number } })?.response?.status;
      if (status === 409) {
        toast.warning(tr("tickets.details.flows.alreadyIn"));
      } else {
        toast.error(tr("tickets.details.flows.startError"));
      }
    } finally {
      setStarting(false);
    }
  };

  const handleAbort = async (runId: string) => {
    try {
      await api.delete(`/flowruns/${runId}`);
      toast.success(tr("tickets.details.flows.stopped"));
      fetchFlowRuns();
    } catch {
      toast.error(tr("tickets.details.flows.stopError"));
    }
  };

  return (
    <>
      <div className="flex flex-col gap-2">
        <div className="flex items-center justify-between">
          <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide flex items-center gap-1">
            <Zap className="w-3 h-3" />
            {tr("tickets.details.flows.title")}
          </p>
          <Button
            variant="ghost"
            size="icon"
            className="h-5 w-5"
            onClick={openDialog}
            title={tr("tickets.details.flows.start")}
          >
            <Play className="w-3.5 h-3.5" />
          </Button>
        </div>

        {flowRuns.length === 0 ? (
          <p className="text-xs text-muted-foreground">{tr("tickets.details.flows.none")}</p>
        ) : (
          flowRuns.map((run) => {
            const isActive = ACTIVE_STATUSES.includes(run.status);
            return (
              <div
                key={run.id}
                className="flex items-start justify-between rounded-md border border-border bg-muted/40 px-3 py-2"
              >
                <div className="flex flex-col gap-0.5 min-w-0">
                  <span className="text-xs font-semibold truncate">
                    {run.flow?.name ?? `Fluxo #${run.flowId}`}
                  </span>
                  <span className="text-xs text-muted-foreground truncate">
                    {statusLabel(run.status)}
                  </span>
                </div>
                {isActive && (
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-6 w-6 shrink-0 mt-0.5"
                    onClick={() => handleAbort(run.id)}
                    title={tr("tickets.details.flows.stop")}
                  >
                    <Trash2 className="w-3.5 h-3.5 text-muted-foreground" />
                  </Button>
                )}
              </div>
            );
          })
        )}
      </div>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{tr("tickets.details.flows.dialogTitle")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-3 py-2">
            {flows.length === 0 ? (
              <p className="text-sm text-muted-foreground">{tr("tickets.details.flows.noneAvailable")}</p>
            ) : (
              <Select value={selectedFlow} onValueChange={setSelectedFlow}>
                <SelectTrigger className="w-full">
                  <SelectValue placeholder={tr("tickets.details.flows.select")} />
                </SelectTrigger>
                <SelectContent>
                  {flows.map((f) => (
                    <SelectItem key={f.id} value={String(f.id)}>
                      {f.name}
                      {f.triggerType && (
                        <span className="ml-2 text-xs text-muted-foreground">
                          ({f.triggerType})
                        </span>
                      )}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              {tr("uiText.common.cancel")}
            </Button>
            <Button
              onClick={handleStart}
              disabled={!selectedFlow || flows.length === 0 || starting}
            >
              <Play className="w-4 h-4 mr-1" />
              {starting ? tr("tickets.details.flows.starting") : tr("tickets.details.flows.startButton")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
};

export default FlowsSection;
