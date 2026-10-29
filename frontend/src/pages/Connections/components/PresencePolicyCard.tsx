import React, { useEffect, useState } from "react";
import { Radio } from "lucide-react";
import { i18n } from "../../../translate/i18n";
import { Card, CardContent, CardHeader, CardTitle } from "../../../components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../../components/ui/select";
import { Input } from "../../../components/ui/input";
import { Label } from "../../../components/ui/label";
import {
  PRESENCE_POLICY_OPTIONS,
  DEFAULT_IDLE_MINUTES,
  MAX_IDLE_MINUTES,
  clampIdleMinutes,
  optionFor,
  toPresencePolicy,
  type PresencePolicy,
} from "../presencePolicy";

interface Props {
  policy?: string;
  idleMinutes?: number;
  saving: boolean;
  onChange: (policy: PresencePolicy, idleMinutes: number) => Promise<void>;
}

// i18n.t() deste projeto pode devolver null; a tela só quer string.
const tr = (key: string): string => String(i18n.t(key) ?? "");

const PresencePolicyCard: React.FC<Props> = ({ policy, idleMinutes, saving, onChange }) => {
  const current = toPresencePolicy(policy);
  const [idle, setIdle] = useState(String(idleMinutes ?? DEFAULT_IDLE_MINUTES));
  useEffect(() => setIdle(String(idleMinutes ?? DEFAULT_IDLE_MINUTES)), [idleMinutes]);

  const minutes = clampIdleMinutes(Number(idle));
  const commitIdle = () => {
    setIdle(String(minutes));
    if (minutes !== (idleMinutes ?? DEFAULT_IDLE_MINUTES)) void onChange(current, minutes);
  };

  return (
    <Card data-testid="presence-policy-card">
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          <Radio className="h-4 w-4" /> {tr("connections.presencePolicy.title")}
        </CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <div className="flex flex-col gap-2">
          <Label htmlFor="presence-policy">{tr("connections.presencePolicy.fieldLabel")}</Label>
          <Select value={current} onValueChange={(v) => void onChange(toPresencePolicy(v), minutes)} disabled={saving}>
            <SelectTrigger id="presence-policy" className="max-w-md" aria-label={tr("connections.presencePolicy.ariaLabel")}>
              <SelectValue>{tr(optionFor(current).labelKey)}</SelectValue>
            </SelectTrigger>
            <SelectContent className="max-w-md">
              {PRESENCE_POLICY_OPTIONS.map((o) => (
                <SelectItem key={o.value} value={o.value} textValue={tr(o.labelKey)} className="items-start">
                  <div className="flex flex-col gap-0.5 py-0.5">
                    <span className="font-medium">{tr(o.labelKey)}</span>
                    <span className="whitespace-normal text-xs text-muted-foreground">{tr(o.summaryKey)}</span>
                  </div>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <p className="max-w-md text-xs text-muted-foreground" data-testid="presence-policy-summary">
            {tr(optionFor(current).summaryKey)}
          </p>
        </div>

        {current === "on_demand" && (
          <div className="flex flex-col gap-2">
            <Label htmlFor="presence-idle">{tr("connections.presencePolicy.idleLabel")}</Label>
            <Input
              id="presence-idle"
              type="number"
              min={1}
              max={MAX_IDLE_MINUTES}
              value={idle}
              disabled={saving}
              onChange={(e) => setIdle(e.target.value)}
              onBlur={commitIdle}
              onKeyDown={(e) => e.key === "Enter" && (e.target as HTMLInputElement).blur()}
              className="max-w-[140px]"
            />
          </div>
        )}
      </CardContent>
    </Card>
  );
};

export default PresencePolicyCard;
