import React, { useState } from "react";
import { Search, Plus, Edit, Trash2, Loader2, ShieldCheck } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import ConfirmationModal from "../../../components/ConfirmationModal";

import { useCargosTab } from "../hooks/useCargosTab";
import CargoPanel from "./CargoPanel";
import type { CargoListItem } from "../acessosTypes";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
const CargosTab: React.FC = () => {
  const {
    cargos,
    loading,
    searchParam,
    handleSearch,
    panelOpen,
    editingCargo,
    panelLoading,
    catalog,
    openCreate,
    openEdit,
    closePanel,
    saveCargo,
    deleteCargo,
  } = useCargosTab();

  const [confirmOpen, setConfirmOpen] = useState(false);
  const [deletingCargo, setDeletingCargo] = useState<CargoListItem | null>(null);

  return (
    <div className="flex flex-col gap-4">
      <ConfirmationModal
        title={deletingCargo ? tr("uiText.acessos.deleteCargoTitle", { name: deletingCargo.name }) : ""}
        open={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        onConfirm={async () => {
          if (deletingCargo) await deleteCargo(deletingCargo);
          setConfirmOpen(false);
          setDeletingCargo(null);
        }}
      >
        {tr("uiText.acessos.cargoDeleteHint")}
      </ConfirmationModal>

      <CargoPanel
        open={panelOpen}
        loading={panelLoading}
        editingCargo={editingCargo}
        catalog={catalog}
        onClose={closePanel}
        onSave={saveCargo}
      />

      <div className="flex items-center gap-2">
        <div className="relative w-full max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Buscar cargo..."
            value={searchParam}
            onChange={handleSearch}
            className="pl-9 h-10"
          />
        </div>
        <Button onClick={openCreate} className="ml-auto gap-2">
          <Plus className="h-4 w-4" />
          {tr("uiText.acessos.newCargo")}
        </Button>
      </div>

      <div className="rounded-2xl bg-card shadow-[0px_4px_20px_rgba(0,0,0,0.08)]">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{tr("uiText.common.name")}</TableHead>
              <TableHead>{tr("uiText.acessos.description")}</TableHead>
              <TableHead className="text-right w-[110px]">{tr("uiText.common.actions")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {cargos.map((cargo) => (
              <TableRow key={cargo.id}>
                <TableCell>
                  <div className="flex items-center gap-2">
                    <ShieldCheck className="h-4 w-4 text-primary" />
                    <span className="font-semibold">{cargo.name}</span>
                    {cargo.isSystem && (
                      <Badge variant="secondary" className="text-xs">
                        Sistema
                      </Badge>
                    )}
                  </div>
                </TableCell>
                <TableCell>
                  <span className="text-muted-foreground">
                    {cargo.description || tr("uiText.acessos.noDescription")}
                  </span>
                </TableCell>
                <TableCell className="text-right">
                  <div className="flex items-center justify-end gap-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8"
                      onClick={() => openEdit(cargo)}
                    >
                      <Edit className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8 text-destructive"
                      onClick={() => {
                        setDeletingCargo(cargo);
                        setConfirmOpen(true);
                      }}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
            {loading && (
              <TableRow>
                <TableCell colSpan={3} className="h-24 text-center">
                  <div className="flex items-center justify-center">
                    <Loader2 className="h-6 w-6 animate-spin text-primary" />
                  </div>
                </TableCell>
              </TableRow>
            )}
            {!loading && cargos.length === 0 && (
              <TableRow>
                <TableCell colSpan={3} className="h-24 text-center text-muted-foreground">
                  Nenhum cargo encontrado.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  );
};

export default CargosTab;
