import React, { useEffect, useState } from "react";
import { Phone, TriangleAlert } from "lucide-react";
import { format, parseISO } from "date-fns";

import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Skeleton } from "@/components/ui/skeleton";
import notify from "@/lib/notify";
import { t } from "@/lib/calls/t";
import api from "../../../services/api";
import {
  RECORDING_MODES,
  buildSaveBody,
  normalizeMode,
  saveState,
  type RecordingConfig,
  type RecordingMode,
} from "../utils/callRecording";

const MODE_LABEL: Record<RecordingMode, string> = {
  off: "calls.settings.modeOff",
  optional: "calls.settings.modeOptional",
  auto: "calls.settings.modeAuto",
};

function ackedLabel(iso: string | null): string {
  if (!iso) return "";
  try {
    return format(parseISO(iso), "dd/MM/yyyy HH:mm");
  } catch {
    return iso;
  }
}

/**
 * Seção "Chamadas" das Configurações: modo de gravação (desligada, opcional,
 * automática). Sair de "desligada" mostra o termo de responsabilidade e só deixa
 * salvar depois do aceite, que o servidor registra com usuário e horário. Usa a
 * rota própria /calls/recording-config (calls:manage), não o PUT /settings genérico.
 */
const CallsSection: React.FC = () => {
  const [cfg, setCfg] = useState<RecordingConfig | null>(null);
  const [selected, setSelected] = useState<RecordingMode>("off");
  const [acked, setAcked] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    let alive = true;
    api
      .get<RecordingConfig>("/calls/recording-config")
      .then(({ data }) => {
        if (!alive) return;
        const mode = normalizeMode(data.mode);
        setCfg({ ...data, mode });
        setSelected(mode);
      })
      .catch((err) => notify.error(err));
    return () => {
      alive = false;
    };
  }, []);

  if (!cfg) return <Skeleton className="h-48 w-full" data-testid="calls-settings-loading" />;

  const state = saveState(cfg.mode, selected, acked, cfg.available);

  const save = async () => {
    setSaving(true);
    try {
      const { data } = await api.put<RecordingConfig>("/calls/recording-config", buildSaveBody(cfg.mode, selected, acked));
      const mode = normalizeMode(data.mode);
      setCfg({ ...data, mode });
      setSelected(mode);
      setAcked(false);
      notify.success(t("calls.settings.saved"));
    } catch (err) {
      notify.error(err);
    } finally {
      setSaving(false);
    }
  };

  return (
    <section className="space-y-6" data-testid="calls-settings">
      <header className="space-y-1">
        <h2 className="flex items-center gap-2 text-lg font-semibold">
          <Phone className="h-5 w-5 text-muted-foreground" />
          {t("calls.settings.title")}
        </h2>
        <p className="text-sm text-muted-foreground">{t("calls.settings.description")}</p>
      </header>

      {!cfg.available && (
        <div role="alert" className="flex items-start gap-2 rounded-md bg-status-warning-bg px-3 py-2 text-sm text-status-warning-text" data-testid="calls-no-storage">
          <TriangleAlert className="mt-0.5 h-4 w-4 shrink-0" />
          <span>{t("calls.settings.unavailable")}</span>
        </div>
      )}

      <div className="space-y-3">
        <Label className="text-sm font-medium">{t("calls.settings.mode")}</Label>
        <RadioGroup value={selected} onValueChange={(v) => { setSelected(normalizeMode(v)); setAcked(false); }}>
          {RECORDING_MODES.map((m) => (
            <div key={m} className="flex items-center gap-2">
              <RadioGroupItem value={m} id={`rec-${m}`} disabled={m !== "off" && !cfg.available} data-testid={`mode-${m}`} />
              <Label htmlFor={`rec-${m}`} className="font-normal">{t(MODE_LABEL[m])}</Label>
            </div>
          ))}
        </RadioGroup>
      </div>

      {state.showAck && (
        <div className="space-y-3 rounded-2xl bg-card p-4 shadow-[0px_4px_20px_rgba(0,0,0,0.08)]" data-testid="calls-ack">
          <h3 className="text-sm font-semibold">{t("calls.settings.ackTitle")}</h3>
          <p className="text-sm leading-relaxed text-muted-foreground">{t("calls.settings.ackText")}</p>
          <div className="flex items-center gap-2">
            <Checkbox id="calls-ack-check" checked={acked} onCheckedChange={(v) => setAcked(v === true)} data-testid="calls-ack-check" />
            <Label htmlFor="calls-ack-check" className="font-normal">{t("calls.settings.ackCheckbox")}</Label>
          </div>
        </div>
      )}

      {cfg.ackAt && cfg.mode !== "off" && (
        <p className="text-xs text-muted-foreground" data-testid="calls-ack-record">
          {t("calls.settings.ackBy")} #{cfg.ackBy} {t("calls.settings.ackAt")} {ackedLabel(cfg.ackAt)}
        </p>
      )}

      <Button onClick={() => void save()} disabled={!state.canSave || saving} data-testid="calls-save">
        {t("calls.settings.save")}
      </Button>
    </section>
  );
};

export default CallsSection;
