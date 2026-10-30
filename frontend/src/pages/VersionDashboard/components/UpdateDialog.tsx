import React from "react";
import { Rocket } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ReleaseMeta } from "../types";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface UpdateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  availableVersion: string;
  frontendVersion: string;
  releaseMeta: ReleaseMeta;
  changelog: string[];
  blockedByCompatibility: boolean;
}

const UpdateDialog: React.FC<UpdateDialogProps> = ({
  open,
  onOpenChange,
  availableVersion,
  frontendVersion,
  releaseMeta,
  changelog,
  blockedByCompatibility,
}) => (
  <Dialog open={open} onOpenChange={onOpenChange}>
    <DialogContent>
      <DialogHeader>
        <DialogTitle className="flex items-center gap-2">
          <Rocket className="h-5 w-5 text-muted-foreground" />
          {tr("uiText.versionDashboard.check")}
        </DialogTitle>
      </DialogHeader>
      <div className="space-y-2 text-sm">
        <p>
          Versão disponível:{" "}
          <strong>v{availableVersion !== "-" ? availableVersion : "latest"}</strong>
        </p>
        <p className="text-muted-foreground">Versão atual: v{frontendVersion}</p>

        {releaseMeta?.breaking && (
          <div className="mt-2 space-y-1">
            <Badge variant="destructive">{tr("uiText.versionDashboard.breaking")}</Badge>
            {releaseMeta?.minCompatibleFrom && (
              <p className="text-xs text-destructive">
                Compatível apenas a partir de v{releaseMeta.minCompatibleFrom}.
              </p>
            )}
            {releaseMeta?.migrationNotes && (
              <p className="text-xs text-muted-foreground">
                Migração necessária: {releaseMeta.migrationNotes}
              </p>
            )}
          </div>
        )}

        {changelog.length > 0 ? (
          <>
            <p className="pt-2">{tr("uiText.versionDashboard.changelog")}</p>
            <ul className="list-disc pl-5 space-y-1">
              {changelog.map((item, idx) => (
                <li key={`${idx}-${item}`}>{item}</li>
              ))}
            </ul>
          </>
        ) : (
          <p className="text-muted-foreground">
            {tr("uiText.versionDashboard.noChangelog")}
          </p>
        )}

        <p className="pt-2">
          A atualização é feita no servidor, puxando as novas imagens:
        </p>
        <pre className="rounded bg-muted p-2 text-xs">docker compose pull && docker compose up -d</pre>

        {blockedByCompatibility && (
          <p className="text-destructive">
            {tr("uiText.versionDashboard.blocked")}
          </p>
        )}
      </div>
      <DialogFooter>
        <Button variant="outline" onClick={() => onOpenChange(false)}>
          {tr("uiText.common.close")}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
);

export default UpdateDialog;
