import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { QuickAnswerContentMedia } from "../quickAnswersTypes";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
export const MediaEditor = ({
  content,
  onChange,
  errors,
}: {
  content: QuickAnswerContentMedia;
  onChange: (c: QuickAnswerContentMedia) => void;
  errors: Record<string, string | undefined>;
}) => (
  <div className="space-y-4">
    <div className="space-y-1.5">
      <Label>{tr("uiText.quick.mediaType")}</Label>
      <Select
        value={content.media_type}
        onValueChange={(v) => onChange({ ...content, media_type: v as QuickAnswerContentMedia["media_type"] })}
      >
        <SelectTrigger><SelectValue /></SelectTrigger>
        <SelectContent>
          <SelectItem value="image">Imagem</SelectItem>
          <SelectItem value="video">{tr("uiText.flow.form.video")}</SelectItem>
          <SelectItem value="audio">{tr("uiText.flow.form.audio")}</SelectItem>
          <SelectItem value="document">{tr("uiText.common.document")}</SelectItem>
        </SelectContent>
      </Select>
    </div>
    <div className="space-y-1.5">
      <Label>URL</Label>
      <Input
        placeholder="https://..."
        value={content.url}
        onChange={(e) => onChange({ ...content, url: e.target.value })}
        aria-invalid={!!errors.url}
      />
      {errors.url && <p className="text-xs text-destructive">{errors.url}</p>}
    </div>
    <div className="space-y-1.5">
      <Label>Legenda <span className="text-muted-foreground font-normal">(opcional)</span></Label>
      <Textarea
        rows={3}
        placeholder={tr("uiText.quick.fileDescription")}
        value={content.caption ?? ""}
        onChange={(e) => onChange({ ...content, caption: e.target.value })}
        className="resize-none"
      />
    </div>
  </div>
);
