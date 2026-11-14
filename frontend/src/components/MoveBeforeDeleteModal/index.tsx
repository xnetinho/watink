import React, { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { i18n } from "../../translate/i18n";

export interface MoveTarget {
  id: number;
  name: string;
}

interface MoveBeforeDeleteModalProps {
  open: boolean;
  title: React.ReactNode;
  description: React.ReactNode;
  targetLabel: string;
  targets: MoveTarget[];
  onClose: () => void;
  onConfirm: (targetId: number) => void;
}

// Exclusão que precisa de destino: a fila ou o usuário que sai deixa tickets, vínculos ou carteira, e o servidor só exclui depois de saber para quem vão.
export default function MoveBeforeDeleteModal({ open, title, description, targetLabel, targets, onClose, onConfirm }: MoveBeforeDeleteModalProps) {
  const [target, setTarget] = useState("");

  useEffect(() => {
    if (open) setTarget("");
  }, [open]);

  return (
    <Dialog open={open} onOpenChange={(isOpen) => { if (!isOpen) onClose(); }}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription asChild>
            <div>{description}</div>
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-1">
          <Label>{targetLabel}</Label>
          <Select value={target} onValueChange={setTarget}>
            <SelectTrigger>
              <SelectValue placeholder={i18n.t("moveBeforeDelete.placeholder")} />
            </SelectTrigger>
            <SelectContent>
              {targets.map((t) => (
                <SelectItem key={t.id} value={String(t.id)}>
                  {t.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            {i18n.t("confirmationModal.buttons.cancel")}
          </Button>
          <Button variant="default" disabled={target === ""} onClick={() => onConfirm(Number(target))}>
            {i18n.t("moveBeforeDelete.confirm")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
