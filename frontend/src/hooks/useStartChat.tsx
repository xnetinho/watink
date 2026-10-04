import React, { useCallback, useState } from "react";
import { useNavigate } from "react-router";
import { Loader2 } from "lucide-react";

import api from "../services/api";
import toastError from "../errors/toastError";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

interface ConnectionOption {
  id: number;
  name: string;
  number?: string;
}

interface ConflictBody {
  code?: string;
  connections?: ConnectionOption[];
}

interface ApiConflict {
  response?: { status?: number; data?: ConflictBody };
}

export function useStartChat(onStarted?: () => void) {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [pendingContactId, setPendingContactId] = useState<number | null>(null);
  const [options, setOptions] = useState<ConnectionOption[]>([]);
  const [selected, setSelected] = useState("");

  const start = useCallback(
    async (contactId: number | string, whatsappId?: number) => {
      if (!contactId) return;
      setLoading(true);
      try {
        const { data: ticket } = await api.post("/tickets", {
          contactId: Number(contactId),
          ...(whatsappId ? { whatsappId } : {}),
        });
        setPendingContactId(null);
        navigate(`/tickets/${ticket.id}`);
        onStarted?.();
      } catch (err) {
        const body = (err as ApiConflict).response?.data;
        if ((err as ApiConflict).response?.status === 409 && body?.code === "CONNECTION_REQUIRED" && body.connections) {
          setOptions(body.connections);
          setSelected(String(body.connections[0]?.id ?? ""));
          setPendingContactId(Number(contactId));
        } else {
          toastError(err);
        }
      } finally {
        setLoading(false);
      }
    },
    [navigate, onStarted],
  );

  const connectionDialog = (
    <Dialog open={pendingContactId !== null} onOpenChange={(open) => !open && setPendingContactId(null)}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Iniciar conversa por qual conexão?</DialogTitle>
        </DialogHeader>
        <RadioGroup value={selected} onValueChange={setSelected} className="py-2">
          {options.map((o) => (
            <div key={o.id} className="flex items-center gap-2">
              <RadioGroupItem value={String(o.id)} id={`conn-${o.id}`} />
              <Label htmlFor={`conn-${o.id}`}>
                {o.name}
                {o.number ? ` (${o.number})` : ""}
              </Label>
            </div>
          ))}
        </RadioGroup>
        <DialogFooter>
          <Button variant="outline" onClick={() => setPendingContactId(null)} disabled={loading}>
            Cancelar
          </Button>
          <Button
            onClick={() => pendingContactId !== null && start(pendingContactId, Number(selected))}
            disabled={loading || !selected}
          >
            {loading ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : "Iniciar conversa"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );

  return { startChat: start, startingChat: loading, connectionDialog };
}
