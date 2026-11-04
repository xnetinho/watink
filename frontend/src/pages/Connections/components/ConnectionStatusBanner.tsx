import React from "react";
import { Loader2, Scan, CheckCircle2, PlugZap } from "lucide-react";
import { Card, CardContent } from "../../../components/ui/card";
import { Button } from "../../../components/ui/button";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface ConnectionStatusBannerProps {
  isConnected: boolean;
  isBusy: boolean;
  status: string;
  connecting: boolean;
  onDisconnect: () => void;
  onConnectQr: () => void;
  onConnectPairing: () => void;
}

const ConnectionStatusBanner: React.FC<ConnectionStatusBannerProps> = ({
  isConnected,
  isBusy,
  status,
  connecting,
  onDisconnect,
  onConnectQr,
  onConnectPairing,
}) => {
  if (isConnected) {
    return (
      <Card className="border-green-200 bg-green-50">
        <CardContent className="flex items-center gap-4 p-4">
          <CheckCircle2 className="h-8 w-8 text-green-500" />
          <div className="flex-1">
            <p className="font-semibold text-green-800">{tr("uiText.connections.sessionActive")}</p>
            <p className="text-sm text-green-700">{tr("uiText.connections.allGood")}</p>
          </div>
          <Button variant="default" onClick={onDisconnect}>
            <PlugZap className="mr-2 h-4 w-4" /> Desconectar
          </Button>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="border-amber-200 bg-amber-50">
      <CardContent className="flex flex-wrap items-center gap-4 p-4">
        <Scan className="h-8 w-8 text-amber-500" />
        <div className="flex-1 min-w-[200px]">
          <p className="font-semibold text-amber-800">
            {status === "QRCODE"
              ? tr("uiText.connections.waitingQr")
              : isBusy
              ? tr("uiText.connections.startingSession")
              : tr("uiText.connections.sessionDisconnected")}
          </p>
          <p className="text-sm text-amber-700">{tr("uiText.connections.connectNumber")}</p>
        </div>
        {!isBusy && status !== "QRCODE" && (
          <div className="flex gap-2">
            <Button onClick={onConnectQr} disabled={connecting}>
              {connecting ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Scan className="mr-2 h-4 w-4" />
              )}
              {tr("uiText.connections.connectQr")}
            </Button>
            <Button variant="outline" onClick={onConnectPairing}>
              {tr("uiText.connections.useCode")}
            </Button>
          </div>
        )}
      </CardContent>
    </Card>
  );
};

export default ConnectionStatusBanner;
