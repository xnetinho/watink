/* @jsxImportSource react */
import React from "react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  CheckCircle2,
  Info,
  Loader2,
  PlayCircle,
  XCircle,
} from "lucide-react";
import type { AssistantTestResult } from "@/services/assistantService";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface AssistantTestDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  name: string;
  testMessage: string;
  onTestMessageChange: (val: string) => void;
  testing: boolean;
  testResult: AssistantTestResult | null;
  onRunTest: () => Promise<void>;
}

export const AssistantTestDialog: React.FC<AssistantTestDialogProps> = ({
  open,
  onOpenChange,
  name,
  testMessage,
  onTestMessageChange,
  testing,
  testResult,
  onRunTest,
}) => {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Testar Bot — {name || "Assistente"}</DialogTitle>
        </DialogHeader>
        <div className="flex flex-col gap-3">
          <div className="flex flex-col gap-1.5">
            <Label>{tr("uiText.assistants.testMessage")}</Label>
            <Textarea
              value={testMessage}
              onChange={(e) => onTestMessageChange(e.target.value)}
              placeholder={tr("uiText.assistants.testPlaceholder")}
              rows={2}
            />
          </div>
          <Button onClick={onRunTest} disabled={testing}>
            {testing ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" /> Testando...
              </>
            ) : (
              <>
                <PlayCircle className="mr-2 h-4 w-4" /> Enviar teste
              </>
            )}
          </Button>

          {testResult && (
            <div className="rounded-xl border p-3 flex flex-col gap-2">
              {testResult.testable === false ? (
                <div className="flex items-start gap-2 text-sm text-muted-foreground">
                  <Info className="h-4 w-4 mt-0.5 shrink-0" />
                  <span>{testResult.message}</span>
                </div>
              ) : testResult.success ? (
                <>
                  <div className="flex items-center gap-2 text-sm font-medium text-green-600">
                    <CheckCircle2 className="h-4 w-4" /> Resposta real do assistente
                  </div>
                  <p className="text-sm whitespace-pre-wrap bg-muted/40 rounded-lg p-3">
                    {testResult.reply}
                  </p>
                  {testResult.action && (
                    <p className="text-xs text-muted-foreground">
                      Ação do modelo: <span className="font-medium">{testResult.action}</span>
                      {typeof testResult.confidence === "number" &&
                        tr("uiText.assistants.confidence", { value: (testResult.confidence * 100).toFixed(0) })}
                    </p>
                  )}
                </>
              ) : (
                <div className="flex items-start gap-2 text-sm text-destructive">
                  <XCircle className="h-4 w-4 mt-0.5 shrink-0" />
                  <span>{testResult.error ?? tr("uiText.assistants.testFail")}</span>
                </div>
              )}
            </div>
          )}
        </div>
        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            {tr("uiText.common.close")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
