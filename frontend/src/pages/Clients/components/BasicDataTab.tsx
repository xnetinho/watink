import React from "react";
import { TabsContent } from "@/components/ui/tabs";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ClientFormData } from "../clientTypes";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface BasicDataTabProps {
  formData: ClientFormData;
  onChange: (field: keyof ClientFormData, value: string) => void;
}

const BasicDataTab: React.FC<BasicDataTabProps> = ({ formData, onChange }) => (
  <TabsContent value="basic" className="space-y-4 mt-4">
    <div className="grid grid-cols-4 gap-4">
      <div className="col-span-1 space-y-1.5">
        <label className="text-sm font-medium">{tr("uiText.common.type")}</label>
        <Select value={formData.type} onValueChange={(v) => onChange("type", v)}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="pf">{tr("uiText.clients.person")}</SelectItem>
            <SelectItem value="pj">{tr("uiText.clients.company")}</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div className="col-span-3 space-y-1.5">
        <label className="text-sm font-medium">
          Nome / Razão Social <span className="text-destructive">*</span>
        </label>
        <Input
          value={formData.name}
          onChange={(e) => onChange("name", e.target.value)}
          placeholder={tr("uiText.clients.namePlaceholder")}
        />
      </div>
    </div>
    {formData.type === "pf" && (
      <div className="space-y-1.5">
        <label className="text-sm font-medium">Nome Social</label>
        <Input
          value={formData.socialName}
          onChange={(e) => onChange("socialName", e.target.value)}
          placeholder="Nome social (opcional)"
        />
        <p className="text-xs text-muted-foreground">
          Quando preenchido, este nome substitui o nome civil em todo o sistema (LGPD).
        </p>
      </div>
    )}
    <div className="grid grid-cols-3 gap-4">
      <div className="space-y-1.5">
        <label className="text-sm font-medium">
          {formData.type === "pf" ? "CPF" : "CNPJ"}
        </label>
        <Input
          value={formData.document}
          onChange={(e) => onChange("document", e.target.value)}
          placeholder={formData.type === "pf" ? "000.000.000-00" : "00.000.000/0000-00"}
        />
      </div>
      <div className="space-y-1.5">
        <label className="text-sm font-medium">{tr("uiText.common.email")}</label>
        <Input
          type="email"
          value={formData.email}
          onChange={(e) => onChange("email", e.target.value)}
          placeholder="email@exemplo.com"
        />
      </div>
      <div className="space-y-1.5">
        <label className="text-sm font-medium">{tr("uiText.common.phone")}</label>
        <Input
          value={formData.phone}
          onChange={(e) => onChange("phone", e.target.value)}
          placeholder="(00) 00000-0000"
        />
      </div>
    </div>
    <div className="space-y-1.5">
      <label className="text-sm font-medium">{tr("uiText.clients.notes")}</label>
      <Textarea
        value={formData.notes}
        onChange={(e) => onChange("notes", e.target.value)}
        rows={3}
        placeholder={tr("uiText.clients.notesPlaceholder")}
      />
    </div>
  </TabsContent>
);

export default BasicDataTab;
