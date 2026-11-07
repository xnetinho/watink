import React, { useState, useEffect } from "react";
import { Loader2 } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import api from "../../services/api";
import { subscribeToSocket } from "../../services/sse-client";
import toastError from "../../errors/toastError";
import { i18n } from "../../translate/i18n";

interface ChannelConnectionEvent {
  action: string;
  connection: { id: number | string; pairingCode?: string; status?: string };
}


const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface PairingCodeModalProps {
  open: boolean;
  onClose: () => void;
  connectionId: number | string;
}

const PairingCodeModal = ({ open, onClose, connectionId }: PairingCodeModalProps) => {
  const [phoneNumber, setPhoneNumber] = useState("");
  const [loading, setLoading] = useState(false);
  const [pairingCode, setPairingCode] = useState("");

  useEffect(() => {
    if (!connectionId || !open) return;

    const handleSession = (data: ChannelConnectionEvent) => {
      if (data.action === "update" && data.connection.id === connectionId) {
        if (data.connection.pairingCode) {
          setPairingCode(data.connection.pairingCode);
          setLoading(false);
        }
        if (data.connection.status === "CONNECTED") onClose();
      }
    };

    return subscribeToSocket({ channelConnection: handleSession });
  }, [connectionId, open, onClose]);

  const handleGenerate = async () => {
    setLoading(true);
    try {
      await api.post(`/channels/connections/${connectionId}/session`, {
        usePairingCode: true,
        phoneNumber: phoneNumber.replace(/\D/g, ""),
      });
    } catch (err) {
      toastError(err);
      setLoading(false);
    }
  };

  const handleClose = () => {
    setPairingCode("");
    setPhoneNumber("");
    setLoading(false);
    onClose();
  };

  return (
    <Dialog open={open} onOpenChange={(isOpen) => { if (!isOpen) handleClose(); }}>
      <DialogContent className="max-w-xs">
        <DialogHeader>
          <DialogTitle>{i18n.t("connections.pairingCodeModal.title")}</DialogTitle>
        </DialogHeader>

        {!pairingCode ? (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              {i18n.t("connections.pairingCodeModal.instruction")}
            </p>
            <div className="space-y-1">
              <Label htmlFor="pairing-phone">{i18n.t("connections.pairingCodeModal.phoneNumber")}</Label>
              <Input
                id="pairing-phone"
                placeholder="5511999999999"
                value={phoneNumber}
                onChange={(e) => setPhoneNumber(e.target.value)}
                disabled={loading}
              />
            </div>
            <Button
              className="w-full"
              onClick={handleGenerate}
              disabled={loading || !phoneNumber}
            >
              {loading
                ? <><Loader2 className="mr-2 h-4 w-4 animate-spin" /> Gerando...</>
                : i18n.t("connections.pairingCodeModal.generate")
              }
            </Button>
          </div>
        ) : (
          <div className="flex flex-col items-center gap-4 py-4">
            <p className="text-4xl font-bold tracking-[0.15em] text-[hsl(var(--primary))]">
              {pairingCode}
            </p>
            <p className="text-sm text-muted-foreground text-center">
              {i18n.t("connections.pairingCodeModal.instructions")}
            </p>
            <Button variant="outline" onClick={handleClose}>{tr("uiText.common.close")}</Button>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
};

export default PairingCodeModal;
