import React from "react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import type { CargoListItem } from "../acessosTypes";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface UserCargoSectionProps {
  cargos: CargoListItem[];
  cargoId: number | null;
  onChange: (cargoId: number | null) => void;
}

const UserCargoSection: React.FC<UserCargoSectionProps> = ({ cargos, cargoId, onChange }) => {
  return (
    <div className="space-y-1.5">
      <Label>{tr("uiText.common.role")}</Label>
      <Select
        value={cargoId ? String(cargoId) : "none"}
        onValueChange={(v) => onChange(v === "none" ? null : Number(v))}
      >
        <SelectTrigger>
          <SelectValue placeholder={tr("uiText.acessos.selectCargo")} />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="none">
            <em>{tr("uiText.common.none")}</em>
          </SelectItem>
          {cargos.map((cargo) => (
            <SelectItem key={cargo.id} value={String(cargo.id)}>
              {cargo.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
};

export default UserCargoSection;
