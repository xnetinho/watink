import React from "react";
import { useNavigate } from "react-router";
import { SettingsIcon, Palette, Headphones, Brain, Library, HardDrive, Network, MapPin, Info, Sparkles, ClipboardList, Building2, Cloud, Phone } from "lucide-react";
import { Button } from "../../../components/ui/button";
import { t } from "@/lib/calls/t";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface SettingsSideNavProps {
  activeSection: string;
  activePlugins: string[];
  onSelect: (section: string) => void;
  isSuperAdmin?: boolean;
  /** Usuário tem calls:manage: só ele vê a seção Chamadas. */
  canManageCalls?: boolean;
}

const SettingsSideNav: React.FC<SettingsSideNavProps> = ({ activeSection, activePlugins, onSelect, isSuperAdmin, canManageCalls }) => {
  const navigate = useNavigate();

  const item = (section: string, Icon: React.ElementType, label: string, condition = true) =>
    condition ? (
      <Button
        key={section}
        variant={activeSection === section ? "secondary" : "ghost"}
        className="w-full justify-start text-left"
        onClick={() => onSelect(section)}
      >
        <Icon className="mr-2 h-4 w-4" />
        {label}
      </Button>
    ) : null;

  return (
    <aside className="w-full md:w-64 space-y-2 border p-3 rounded-lg bg-card">
      {item("general", SettingsIcon, tr("uiText.settings.nav.general"))}
      {item("company", Building2, tr("uiText.settings.nav.company"))}
      {item("personalization", Palette, tr("uiText.settings.nav.personalization"))}
      {item("helpdesk", Headphones, tr("uiText.settings.nav.helpdesk"), activePlugins.includes("helpdesk"))}
      {item("activities", ClipboardList, tr("uiText.settings.nav.activities"))}
      {item("calls", Phone, t("calls.settings.title"), !!canManageCalls)}
      {item("ai", Brain, tr("uiText.settings.nav.ai"))}
      {item("ai-gateways", Sparkles, tr("uiText.settings.nav.aiGateways"), activePlugins.includes("assistant"))}
      {item("address", MapPin, tr("uiText.settings.nav.address"))}
      {item("proxy", Network, "Proxy")}
      {item("storage", HardDrive, tr("uiText.settings.nav.storage"), !!isSuperAdmin)}
      {item("saas-mode", Cloud, tr("uiText.settings.nav.saas"), !!isSuperAdmin)}
      {item("about", Info, tr("uiText.settings.nav.about"))}

      <Button
        variant="ghost"
        className="w-full justify-start text-left"
        onClick={() => navigate("/knowledge-bases")}
      >
        <Library className="mr-2 h-4 w-4" />
        {tr("uiText.settings.nav.knowledge")}
      </Button>
    </aside>
  );
};

export default SettingsSideNav;
