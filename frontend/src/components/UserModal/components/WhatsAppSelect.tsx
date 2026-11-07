import React from "react";

import { i18n } from "../../../translate/i18n";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

interface WhatsApp {
  id: number;
  name: string;
}

interface WhatsAppSelectProps {
  whatsApps: WhatsApp[];
  connectionId: number | string;
  onChange: (value: number | string) => void;
}

const WhatsAppSelect: React.FC<WhatsAppSelectProps> = ({
  whatsApps,
  connectionId,
  onChange,
}) => {
  return (
    <div className="flex flex-col gap-2">
      <Label>{i18n.t("userModal.form.whatsapp")}</Label>
      <Select
        value={connectionId.toString()}
        onValueChange={(val) => onChange(val === "none" ? "" : parseInt(val))}
      >
        <SelectTrigger>
          <SelectValue placeholder={i18n.t("userModal.form.whatsapp")} />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="none">&nbsp;</SelectItem>
          {whatsApps.map((whatsapp) => (
            <SelectItem key={whatsapp.id} value={whatsapp.id.toString()}>
              {whatsapp.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
};

export default WhatsAppSelect;
