import React from "react";
import QRCode from "qrcode.react";
import { Loader2 } from "lucide-react";
import { Card, CardContent } from "../../../components/ui/card";
import { Button } from "../../../components/ui/button";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface QrCodePanelProps {
  qrcode?: string;
  onCancel: () => void;
}

const QrCodePanel: React.FC<QrCodePanelProps> = ({ qrcode, onCancel }) => (
  <Card>
    <CardContent className="flex flex-col items-center gap-4 p-6">
      <p className="text-sm font-medium">{tr("uiText.connections.scanQr")}</p>
      <div className="rounded-xl border bg-white p-4 shadow-sm">
        {qrcode ? (
          <QRCode value={qrcode} size={256} />
        ) : (
          <Loader2 className="h-10 w-10 animate-spin text-primary" />
        )}
      </div>
      <Button variant="outline" onClick={onCancel}>
        {tr("uiText.common.cancel")}
      </Button>
    </CardContent>
  </Card>
);

export default QrCodePanel;
