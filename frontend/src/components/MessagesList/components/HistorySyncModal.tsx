import React from "react";
import { Loader2 } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "../../ui/dialog";
import { Button } from "../../ui/button";
import { Input } from "../../ui/input";
import { Label } from "../../ui/label";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface Props {
  open: boolean;
  fromDate: string;
  loading: boolean;
  onOpenChange: (v: boolean) => void;
  onFromDateChange: (v: string) => void;
  onSync: () => void;
}

const HistorySyncModal: React.FC<Props> = ({
  open,
  fromDate,
  loading,
  onOpenChange,
  onFromDateChange,
  onSync,
}) => (
  <Dialog open={open} onOpenChange={onOpenChange}>
    <DialogContent className="max-w-xs">
      <DialogHeader>
        <DialogTitle>{tr("uiText.messagesList.historyTitle")}</DialogTitle>
      </DialogHeader>
      <div className="py-2 space-y-2">
        <Label htmlFor="history-date">{tr("uiText.messagesList.historyStart")}</Label>
        <Input
          id="history-date"
          type="date"
          value={fromDate}
          onChange={(e) => onFromDateChange(e.target.value)}
        />
        <p className="text-xs text-muted-foreground">
          {tr("uiText.messagesList.historyHint")}
        </p>
      </div>
      <DialogFooter>
        <Button variant="outline" onClick={() => onOpenChange(false)}>
          {tr("uiText.common.cancel")}
        </Button>
        <Button onClick={onSync} disabled={loading || !fromDate}>
          {loading ? (
            <Loader2 className="h-4 w-4 animate-spin mr-2" />
          ) : null}
          {tr("uiText.messagesList.fetch")}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
);

export default HistorySyncModal;
