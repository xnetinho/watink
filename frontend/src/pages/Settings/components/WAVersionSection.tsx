/* @jsxImportSource react */
import React, { useCallback, useEffect, useState } from "react";
import { toast } from "react-toastify";
import { Loader2, RefreshCw, Smartphone } from "lucide-react";

import api from "../../../services/api";
import toastError from "../../../errors/toastError";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../../components/ui/card";
import { Badge } from "../../../components/ui/badge";
import { Button } from "../../../components/ui/button";
import { Input } from "../../../components/ui/input";
import { Label } from "../../../components/ui/label";
import { RadioGroup, RadioGroupItem } from "../../../components/ui/radio-group";
import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");

interface Probe {
  version?: string;
  error?: string;
}

interface EngineStatus {
  mode: string;
  manual?: string;
  embedded: string;
  official: Probe;
  registry: Probe;
  effective: string;
  source?: string;
}

interface WAVersionResponse {
  mode: "auto" | "manual";
  manualVersion: string | null;
  engine?: EngineStatus;
  engineError?: string;
  engineApplied?: boolean;
}

const VERSION_PATTERN = /^2\.\d{1,5}\.\d{1,12}$/;

const VersionRow: React.FC<{ label: string; hint: string; probe?: Probe; value?: string; testId: string }> = ({
  label,
  hint,
  probe,
  value,
  testId,
}) => (
  <div className="flex flex-col gap-0.5 rounded-md border p-3" data-testid={testId}>
    <span className="text-sm font-medium">{label}</span>
    <span className="text-xs text-muted-foreground">{hint}</span>
    {probe?.error ? (
      <span className="text-sm text-[hsl(var(--status-warning-text))]">
        {tr("uiText.settings.waVersion.unavailable")}: {probe.error}
      </span>
    ) : (
      <span className="font-mono text-sm">{probe?.version ?? value ?? "—"}</span>
    )}
  </div>
);

/**
 * Versão do cliente WhatsApp que o engine anuncia (superadmin). O WhatsApp recusa versões velhas (405); no modo automático o
 * engine busca sozinho (site oficial, registro wa-version, embutida) e no manual usa só o número digitado aqui.
 */
const WAVersionSection: React.FC = () => {
  const [data, setData] = useState<WAVersionResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [mode, setMode] = useState<"auto" | "manual">("auto");
  const [manual, setManual] = useState("");
  const [saving, setSaving] = useState(false);

  const load = useCallback(() => {
    setLoading(true);
    api
      .get<WAVersionResponse>("/system/wa-version")
      .then(({ data: d }) => {
        setData(d);
        setMode(d.mode === "manual" ? "manual" : "auto");
        setManual(d.manualVersion ?? "");
      })
      .catch(toastError)
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const manualInvalid = mode === "manual" && !VERSION_PATTERN.test(manual.trim());

  const save = (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    api
      .put<WAVersionResponse>("/system/wa-version", { mode, manualVersion: mode === "manual" ? manual.trim() : "" })
      .then(({ data: d }) => {
        if (d.engineApplied === false) toast.warning(tr("uiText.settings.waVersion.savedEngineDown"));
        else toast.success(tr("uiText.settings.waVersion.saved"));
        load();
      })
      .catch(toastError)
      .finally(() => setSaving(false));
  };

  const engine = data?.engine;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-primary">
          <Smartphone className="h-5 w-5" />
          {tr("uiText.settings.waVersion.title")}
        </CardTitle>
        <CardDescription>{tr("uiText.settings.waVersion.intro")}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-6">
        {loading ? (
          <div className="flex items-center gap-2 text-muted-foreground">
            <Loader2 className="h-4 w-4 animate-spin" />
            {tr("uiText.settings.waVersion.loading")}
          </div>
        ) : (
          <>
            {data?.engineError && (
              <p className="text-sm text-[hsl(var(--status-warning-text))]" data-testid="wa-version-engine-error">
                {tr("uiText.settings.waVersion.engineDown")}
              </p>
            )}

            {engine && (
              <div className="flex flex-col gap-3">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-sm font-medium">{tr("uiText.settings.waVersion.inUse")}</span>
                  <Badge className="font-mono" data-testid="wa-version-effective">
                    {engine.effective}
                  </Badge>
                  {engine.source && (
                    <span className="text-xs text-muted-foreground">
                      {tr("uiText.settings.waVersion.source")}: {tr(`uiText.settings.waVersion.sources.${engine.source}`)}
                    </span>
                  )}
                </div>
                <div className="grid gap-3 md:grid-cols-3">
                  <VersionRow
                    testId="wa-version-official"
                    label={tr("uiText.settings.waVersion.official")}
                    hint={tr("uiText.settings.waVersion.officialHint")}
                    probe={engine.official}
                  />
                  <VersionRow
                    testId="wa-version-registry"
                    label={tr("uiText.settings.waVersion.registry")}
                    hint={tr("uiText.settings.waVersion.registryHint")}
                    probe={engine.registry}
                  />
                  <VersionRow
                    testId="wa-version-embedded"
                    label={tr("uiText.settings.waVersion.embedded")}
                    hint={tr("uiText.settings.waVersion.embeddedHint")}
                    value={engine.embedded}
                  />
                </div>
              </div>
            )}

            <form onSubmit={save} className="flex flex-col gap-4">
              <RadioGroup value={mode} onValueChange={(v) => setMode(v === "manual" ? "manual" : "auto")}>
                <div className="flex items-start gap-2">
                  <RadioGroupItem value="auto" id="wa-version-auto" className="mt-1" />
                  <Label htmlFor="wa-version-auto" className="flex flex-col gap-0.5 font-normal">
                    <span className="font-medium">{tr("uiText.settings.waVersion.modeAuto")}</span>
                    <span className="text-xs text-muted-foreground">{tr("uiText.settings.waVersion.modeAutoHint")}</span>
                  </Label>
                </div>
                <div className="flex items-start gap-2">
                  <RadioGroupItem value="manual" id="wa-version-manual" className="mt-1" />
                  <Label htmlFor="wa-version-manual" className="flex flex-col gap-0.5 font-normal">
                    <span className="font-medium">{tr("uiText.settings.waVersion.modeManual")}</span>
                    <span className="text-xs text-muted-foreground">{tr("uiText.settings.waVersion.modeManualHint")}</span>
                  </Label>
                </div>
              </RadioGroup>

              {mode === "manual" && (
                <div className="flex max-w-sm flex-col gap-1.5">
                  <Label htmlFor="wa-version-input">{tr("uiText.settings.waVersion.manualLabel")}</Label>
                  {/* i18n-ignore: exemplo de formato de versão, não é texto de tela */}
                  <Input id="wa-version-input" value={manual} onChange={(e) => setManual(e.target.value)} placeholder="2.3000.1052747707" className="font-mono" />
                  {manualInvalid && manual.trim() !== "" && (
                    <span className="text-xs text-[hsl(var(--status-warning-text))]" data-testid="wa-version-invalid">
                      {tr("uiText.settings.waVersion.invalid")}
                    </span>
                  )}
                </div>
              )}

              <div className="flex flex-wrap items-center gap-2">
                <Button type="submit" disabled={saving || manualInvalid}>
                  {saving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                  {tr("uiText.settings.waVersion.save")}
                </Button>
                <Button type="button" variant="outline" onClick={load} disabled={saving}>
                  <RefreshCw className="mr-2 h-4 w-4" />
                  {tr("uiText.settings.waVersion.refresh")}
                </Button>
              </div>
              <p className="text-xs text-muted-foreground">{tr("uiText.settings.waVersion.applyHint")}</p>
            </form>
          </>
        )}
      </CardContent>
    </Card>
  );
};

export default WAVersionSection;
