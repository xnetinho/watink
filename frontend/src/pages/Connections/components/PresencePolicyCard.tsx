import React, { useEffect, useState } from "react";
import { Radio } from "lucide-react";
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
          <Radio className="h-4 w-4" /> Presença do número
        </CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <div className="flex flex-col gap-2">
          <Label htmlFor="presence-policy">Quando o número fica disponível (online)</Label>
          <Select value={current} onValueChange={(v) => void onChange(toPresencePolicy(v), minutes)} disabled={saving}>
            <SelectTrigger id="presence-policy" className="max-w-md" aria-label="Política de presença">
              <SelectValue>{optionFor(current).label}</SelectValue>
            </SelectTrigger>
            <SelectContent className="max-w-md">
              {PRESENCE_POLICY_OPTIONS.map((o) => (
                <SelectItem key={o.value} value={o.value} textValue={o.label} className="items-start">
                  <div className="flex flex-col gap-0.5 py-0.5">
                    <span className="font-medium">{o.label}</span>
                    <span className="whitespace-normal text-xs text-muted-foreground">{o.summary}</span>
                  </div>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <p className="max-w-md text-xs text-muted-foreground" data-testid="presence-policy-summary">
            {optionFor(current).summary}
          </p>
        </div>

        {current === "on_demand" && (
          <div className="flex flex-col gap-2">
            <Label htmlFor="presence-idle">Voltar ao normal após (minutos sem operador)</Label>
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
