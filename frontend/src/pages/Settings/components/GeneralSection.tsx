import React from "react";
import { toast } from "react-toastify";
import { SettingsIcon, Copy, Globe2, KeyRound } from "lucide-react";
import { Button } from "../../../components/ui/button";
import { Input } from "../../../components/ui/input";
import { Label } from "../../../components/ui/label";
import { FormField } from "../../../components/ui/form-field";
import { Switch } from "../../../components/ui/switch";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "../../../components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../../components/ui/select";
import { Separator } from "../../../components/ui/separator";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
const timezones = [
  { value: "America/Sao_Paulo", label: "uiText.settings.general.tz.saoPaulo" },
  { value: "America/Manaus", label: "uiText.settings.general.tz.manaus" },
  { value: "America/Noronha", label: "uiText.settings.general.tz.noronha" },
  { value: "America/Caracas", label: "uiText.settings.general.tz.caracas" },
  { value: "America/New_York", label: "uiText.settings.general.tz.newYork" },
  { value: "Europe/London", label: "uiText.settings.general.tz.london" },
  { value: "Europe/Paris", label: "uiText.settings.general.tz.paris" },
  { value: "UTC", label: "Coordinated Universal Time (UTC)" },
];

interface GeneralSectionProps {
  getSettingValue: (key: string) => string;
  handleUpdateSetting: (key: string, value: string) => Promise<void>;
  handleLanguageChange: (lang: string) => Promise<void>;
}

const GeneralSection: React.FC<GeneralSectionProps> = ({
  getSettingValue,
  handleUpdateSetting,
  handleLanguageChange,
}) => (
  <div className="space-y-6">
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-primary">
          <SettingsIcon className="h-5 w-5" />
          {tr("uiText.settings.general.title")}
        </CardTitle>
        <CardDescription>{tr("uiText.settings.general.subtitle")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="flex items-center justify-between">
          <div className="space-y-0.5">
            <Label>{tr("uiText.settings.general.signature")}</Label>
            <p className="text-sm text-muted-foreground">{tr("uiText.settings.general.signatureHint")}</p>
          </div>
          <Switch
            checked={getSettingValue("userSignature") === "enabled"}
            onCheckedChange={(checked) => handleUpdateSetting("userSignature", checked ? "enabled" : "disabled")}
          />
        </div>

        <Separator />

        <div className="flex items-center justify-between">
          <div className="space-y-0.5">
            <Label>{tr("uiText.settings.general.checkin")}</Label>
            <p className="text-sm text-muted-foreground">{tr("uiText.settings.general.checkinHint")}</p>
          </div>
          <Switch
            checked={getSettingValue("forceCheckIn") === "enabled"}
            onCheckedChange={(checked) => handleUpdateSetting("forceCheckIn", checked ? "enabled" : "disabled")}
          />
        </div>

      </CardContent>
    </Card>

    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-primary">
          <Globe2 className="h-5 w-5" />
          {tr("uiText.settings.general.location")}
        </CardTitle>
        <CardDescription>{tr("uiText.settings.general.locationHint")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="grid gap-4 md:grid-cols-2">
          <FormField htmlFor="system-lang" label="Idioma Global">
            <Select value={getSettingValue("language") || "pt"} onValueChange={handleLanguageChange}>
              <SelectTrigger id="system-lang">
                <SelectValue placeholder="Idioma" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="pt">{tr("uiText.settings.general.langPt")}</SelectItem>
                <SelectItem value="en">English</SelectItem>
                <SelectItem value="es">Español</SelectItem>
              </SelectContent>
            </Select>
          </FormField>

          <FormField htmlFor="system-timezone" label={tr("uiText.settings.general.timezone")}>
            <Select
              value={getSettingValue("timezone") || "America/Sao_Paulo"}
              onValueChange={(v) => handleUpdateSetting("timezone", v)}
            >
              <SelectTrigger id="system-timezone">
                <SelectValue placeholder="Selecione o Timezone" />
              </SelectTrigger>
              <SelectContent>
                {timezones.map((tz) => (
                  <SelectItem key={tz.value} value={tz.value}>{tr(tz.label)}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FormField>
        </div>
      </CardContent>
    </Card>

    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-primary">
          <KeyRound className="h-5 w-5" />
          {tr("uiText.settings.general.integration")}
        </CardTitle>
        <CardDescription>{tr("uiText.settings.general.integrationHint")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <FormField htmlFor="api-token" label={tr("uiText.settings.general.apiToken")}>
          <div className="flex gap-2">
            <Input id="api-token" readOnly value={getSettingValue("userApiToken")} className="font-mono bg-muted" />
            <Button variant="outline" size="icon" onClick={() => {
              navigator.clipboard.writeText(getSettingValue("userApiToken"));
              toast.success("Token copiado!");
            }}>
              <Copy className="h-4 w-4" />
            </Button>
          </div>
        </FormField>
      </CardContent>
    </Card>
  </div>
);

export default GeneralSection;
