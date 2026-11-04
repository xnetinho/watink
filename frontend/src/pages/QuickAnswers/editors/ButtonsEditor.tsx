import { Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { genId } from "../quickAnswersHelpers";
import type { QuickAnswerContentButtons } from "../quickAnswersTypes";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
export const ButtonsEditor = ({
  content,
  onChange,
  errors,
}: {
  content: QuickAnswerContentButtons;
  onChange: (c: QuickAnswerContentButtons) => void;
  errors: Record<string, string | undefined>;
}) => (
  <div className="space-y-4">
    <div className="space-y-1.5">
      <Label>{tr("uiText.quick.message")}</Label>
      <Textarea
        rows={4}
        placeholder={tr("uiText.quick.formatHint")}
        value={content.body}
        onChange={(e) => onChange({ ...content, body: e.target.value })}
        className="resize-none"
        aria-invalid={!!errors.body}
      />
      {errors.body && <p className="text-xs text-destructive">{errors.body}</p>}
    </div>
    <div className="space-y-1.5">
      <Label>{tr("uiText.quick.footer")} <span className="text-muted-foreground font-normal">(opcional)</span></Label>
      <Input
        placeholder={tr("uiText.quick.footerText")}
        value={content.footer ?? ""}
        onChange={(e) => onChange({ ...content, footer: e.target.value })}
      />
    </div>
    <div className="space-y-2">
      <Label>{tr("uiText.quick.buttons")}</Label>
      {content.buttons.map((btn, idx) => (
        <div key={btn.id} className="flex gap-2 items-center">
          <span className="text-xs text-muted-foreground w-5 shrink-0">{idx + 1}.</span>
          <Input
            placeholder={tr("uiText.quick.buttonLabel", { n: idx + 1 })}
            value={btn.label}
            onChange={(e) => {
              const buttons = content.buttons.map((b, i) =>
                i === idx ? { ...b, label: e.target.value } : b
              );
              onChange({ ...content, buttons });
            }}
          />
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="shrink-0 text-muted-foreground hover:text-destructive"
            onClick={() => onChange({ ...content, buttons: content.buttons.filter((_, i) => i !== idx) })}
            disabled={content.buttons.length <= 1}
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      ))}
      {errors.buttons && <p className="text-xs text-destructive">{errors.buttons}</p>}
      {content.buttons.length < 3 && (
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() =>
            onChange({
              ...content,
              buttons: [...content.buttons, { id: genId("btn", content.buttons.length), label: "" }],
            })
          }
        >
          <Plus className="h-4 w-4 mr-1" /> Adicionar botão
        </Button>
      )}
    </div>
  </div>
);
