import React from "react";
import { Loader2 } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "../../../components/ui/dialog";
import { Button } from "../../../components/ui/button";
import { Input } from "../../../components/ui/input";
import { Label } from "../../../components/ui/label";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface PhoneInputDialogProps {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  phoneNumber: string;
  onPhoneNumberChange: (v: string) => void;
  onConfirm: () => void;
  pairingLoading: boolean;
}

const PhoneInputDialog: React.FC<PhoneInputDialogProps> = ({
  open,
  onOpenChange,
  phoneNumber,
  onPhoneNumberChange,
  onConfirm,
  pairingLoading,
}) => (
  <Dialog open={open} onOpenChange={onOpenChange}>
    <DialogContent>
      <DialogHeader>
        <DialogTitle>{tr("uiText.connections.pairingNumber")}</DialogTitle>
      </DialogHeader>
      <div className="py-4">
        <Label htmlFor="phoneNumber" className="mb-2 block">
          Digite o número do telefone com DDD (Ex: 5585999999999):
        </Label>
        <Input
          id="phoneNumber"
          autoFocus
          placeholder="5585999999999"
          value={phoneNumber}
          onChange={(e) => onPhoneNumberChange(e.target.value)}
        />
      </div>
      <DialogFooter>
        <Button variant="outline" onClick={() => onOpenChange(false)}>
          {tr("uiText.common.cancel")}
        </Button>
        <Button
          onClick={onConfirm}
          disabled={!phoneNumber || phoneNumber.length < 10 || pairingLoading}
        >
          {pairingLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          {tr("uiText.connections.generateCode")}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
);

export default PhoneInputDialog;
