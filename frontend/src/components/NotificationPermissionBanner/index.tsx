import React, { useState, useEffect } from "react";
import { Bell, X } from "lucide-react";
import { Button } from "../ui/button";

import { i18n } from "../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
const NotificationPermissionBanner: React.FC = () => {
  const [permission, setPermission] = useState<NotificationPermission | null>(null);
  const [dismissed, setDismissed] = useState(false);

  useEffect(() => {
    if (!("Notification" in window)) return;
    setPermission(Notification.permission);
  }, []);

  const handleEnable = async () => {
    // Chrome requires requestPermission to be called from a user gesture
    const result = await Notification.requestPermission();
    setPermission(result);
  };

  if (
    !("Notification" in window) ||
    permission !== "default" ||
    dismissed
  ) {
    return null;
  }

  return (
    <div className="fixed bottom-4 left-1/2 -translate-x-1/2 z-50 flex items-center gap-3 rounded-xl border border-border bg-card shadow-lg px-4 py-3 text-sm">
      <Bell size={16} className="text-primary shrink-0" />
      <span className="text-foreground">
        {tr("uiText.notificationPermissionBanner.enable")}
      </span>
      <Button size="sm" onClick={handleEnable}>
        Habilitar
      </Button>
      <button
        onClick={() => setDismissed(true)}
        className="text-muted-foreground hover:text-foreground"
        aria-label={tr("uiText.common.close")}
      >
        <X size={14} />
      </button>
    </div>
  );
};

export default NotificationPermissionBanner;
