import React from "react";
import { Loader2, PlugZap, RefreshCw, QrCode, Trash2 } from "lucide-react";
import ActionCard from "./ActionCard";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface ConnectionActionGridProps {
  isConnected: boolean;
  isBusy: boolean;
  connecting: boolean;
  restarting: boolean;
  onDisconnect: () => void;
  onRestart: () => void;
  onStartQr: () => void;
  onDelete: () => void;
}

const ConnectionActionGrid = ({
  isConnected,
  isBusy,
  connecting,
  restarting,
  onDisconnect,
  onRestart,
  onStartQr,
  onDelete,
}: ConnectionActionGridProps) => (
  <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
    <ActionCard
      icon={<PlugZap className="h-5 w-5" />}
      label="Desconectar"
      tone="default"
      disabled={!isConnected}
      onClick={onDisconnect}
    />
    <ActionCard
      icon={restarting ? <Loader2 className="h-5 w-5 animate-spin" /> : <RefreshCw className="h-5 w-5" />}
      label={tr("uiText.connections.restartSession")}
      tone="default"
      disabled={isBusy || restarting}
      onClick={onRestart}
    />
    <ActionCard
      icon={<QrCode className="h-5 w-5" />}
      label="Gerar QR Code"
      tone="default"
      disabled={isConnected || isBusy || connecting}
      onClick={onStartQr}
    />
    <ActionCard
      icon={<Trash2 className="h-5 w-5" />}
      label={tr("uiText.connections.deleteConnection")}
      tone="destructive"
      onClick={onDelete}
    />
  </div>
);

export default ConnectionActionGrid;
