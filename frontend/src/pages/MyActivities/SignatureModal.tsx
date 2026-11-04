/* @jsxImportSource react */
import React, { useState, useRef } from "react";
import SignatureCanvas from "react-signature-canvas";
import { Eraser } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";

import { i18n } from "../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface SignatureModalProps {
  open: boolean;
  onClose: () => void;
  onConfirm: (dataUrl: string) => void;
  title?: string;
}

const SignatureModal: React.FC<SignatureModalProps> = ({
  open,
  onClose,
  onConfirm,
  title = "Assinatura",
}) => {
  const sigPad = useRef<SignatureCanvas>(null);
  const [isEmpty, setIsEmpty] = useState(true);

  const handleClear = () => {
    sigPad.current?.clear();
    setIsEmpty(true);
  };

  const handleConfirm = () => {
    if (!sigPad.current || sigPad.current.isEmpty()) {
      alert(tr("uiText.activities.signFirst"));
      return;
    }
    const dataUrl = sigPad.current.getTrimmedCanvas().toDataURL("image/png");
    onConfirm(dataUrl);
  };

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>
        <div className="space-y-3">
          <p className="text-sm text-muted-foreground">
            {tr("uiText.activities.signHint")}
          </p>
          <div
            className="relative w-full rounded-md border border-border bg-muted/50"
            style={{ height: 300 }}
          >
            <SignatureCanvas
              ref={sigPad}
              penColor="var(--foreground)"
              canvasProps={{
                className: "w-full h-full",
              }}
              onBegin={() => setIsEmpty(false)}
            />
          </div>
        </div>
        <DialogFooter className="gap-2 sm:gap-0">
          <Button variant="secondary" size="sm" onClick={handleClear}>
            <Eraser className="mr-2 h-4 w-4" />
            {tr("uiText.common.clear")}
          </Button>
          <Button variant="outline" size="sm" onClick={onClose}>
            {tr("uiText.common.cancel")}
          </Button>
          <Button size="sm" onClick={handleConfirm} disabled={isEmpty}>
            Confirmar Assinatura
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default SignatureModal;