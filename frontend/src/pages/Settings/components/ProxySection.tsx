import React, { useCallback, useEffect, useMemo, useState } from "react";
import { toast } from "react-toastify";
import { Plus, Upload, Pencil, Trash2, ShieldAlert, ShieldCheck, Loader2, Network, Wifi, Gauge, Search, FolderPlus } from "lucide-react";

import { Button } from "../../../components/ui/button";
import { Badge } from "../../../components/ui/badge";
import { Input } from "../../../components/ui/input";
import { Label } from "../../../components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../../../components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../../../components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../../../components/ui/table";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "../../../components/ui/card";

import api from "../../../services/api";
import toastError from "../../../errors/toastError";
import type { Proxy, ProxyGroup } from "../../../types/domain";
import ProxyFormDialog from "../../Proxies/ProxyFormDialog";
import ProxyImportDialog from "../../Proxies/ProxyImportDialog";
import ProxyGroupsPanel from "../../Proxies/ProxyGroupsPanel";
import ConnectionGroupsDialog from "../../Proxies/ConnectionGroupsDialog";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
const STATUS_BADGE: Record<string, { label: string; variant: "default" | "secondary" | "destructive" | "outline" }> = {
  active: { label: "uiText.settings.proxy.statusActive", variant: "default" },
  isolated: { label: "uiText.settings.proxy.statusIsolated", variant: "destructive" },
  banned: { label: "uiText.settings.proxy.statusBanned", variant: "destructive" },
  disabled: { label: "uiText.settings.proxy.statusDisabled", variant: "secondary" },
};

const ProxySection: React.FC = () => {
  const [proxies, setProxies] = useState<Proxy[]>([]);
  const [groups, setGroups] = useState<ProxyGroup[]>([]);
  const [loading, setLoading] = useState(true);
  const [formOpen, setFormOpen] = useState(false);
  const [importOpen, setImportOpen] = useState(false);
  const [connGroupsOpen, setConnGroupsOpen] = useState(false);
  const [editing, setEditing] = useState<Proxy | null>(null);
  const [testingId, setTestingId] = useState<number | null>(null);
  const [testingAll, setTestingAll] = useState(false);
  const [deletingAll, setDeletingAll] = useState(false);
  const [search, setSearch] = useState("");
  const [cityFilter, setCityFilter] = useState("all");
  const [statusFilter, setStatusFilter] = useState("all");
  const [assignOpen, setAssignOpen] = useState(false);
  const [assignGroupId, setAssignGroupId] = useState("none");
  const [assigning, setAssigning] = useState(false);

  const fetchProxies = useCallback(async () => {
    setLoading(true);
    try {
      const { data } = await api.get<Proxy[]>("/proxies");
      setProxies(Array.isArray(data) ? data : []);
    } catch (err) {
      toastError(err);
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchGroups = useCallback(async () => {
    try {
      const { data } = await api.get<ProxyGroup[]>("/proxy-groups");
      setGroups(Array.isArray(data) ? data : []);
    } catch {
      setGroups([]);
    }
  }, []);

  useEffect(() => { fetchProxies(); fetchGroups(); }, [fetchProxies, fetchGroups]);

  const groupName = (id?: number | null) =>
    id == null ? "—" : groups.find((g) => g.id === id)?.name ?? `#${id}`;

  const openCreate = () => { setEditing(null); setFormOpen(true); };
  const openEdit = (p: Proxy) => { setEditing(p); setFormOpen(true); };

  const handleToggleIsolate = async (p: Proxy) => {
    const action = p.status === "isolated" ? "activate" : "isolate";
    try {
      await api.post(`/proxies/${p.id}/${action}`);
      toast.success(action === "isolate" ? "Proxy isolado." : "Proxy reativado.");
      fetchProxies();
    } catch (err) {
      toastError(err);
    }
  };

  const handleDelete = async (p: Proxy) => {
    if (!window.confirm(`Remover o proxy ${p.host}:${p.port}?`)) return;
    try {
      await api.delete(`/proxies/${p.id}`);
      toast.success("Proxy removido.");
      fetchProxies();
    } catch (err) {
      toastError(err);
    }
  };

  const handleTestAll = async () => {
    setTestingAll(true);
    try {
      const { data } = await api.post<{ tested: number; ok: number; failed: number }>("/proxies/test-all");
      if (data.tested === 0) {
        toast.info(tr("uiText.settings.proxy.noneToTest"));
      } else {
        toast.success(tr("uiText.settings.proxy.testDone", { ok: data.ok, tested: data.tested, failed: data.failed }));
      }
      fetchProxies();
    } catch (err) {
      toastError(err);
    } finally {
      setTestingAll(false);
    }
  };

  const handleDeleteAll = async () => {
    if (proxies.length === 0) { toast.info(tr("uiText.settings.proxy.noneToRemove")); return; }
    if (!window.confirm(tr("uiText.settings.proxy.removeAllConfirm", { count: proxies.length }))) return;
    setDeletingAll(true);
    try {
      const { data } = await api.delete<{ deleted: number }>("/proxies");
      toast.success(`${data.deleted} proxies removidos.`);
      fetchProxies();
    } catch (err) {
      toastError(err);
    } finally {
      setDeletingAll(false);
    }
  };

  const cities = useMemo(() => {
    const set = new Set<string>();
    proxies.forEach((p) => { if (p.city) set.add(p.city); });
    return Array.from(set).sort();
  }, [proxies]);

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    return proxies.filter((p) => {
      if (cityFilter !== "all" && p.city !== cityFilter) return false;
      if (statusFilter !== "all" && p.status !== statusFilter) return false;
      if (q) {
        const hay = `${p.label} ${p.host} ${p.port} ${p.city ?? ""} ${p.country ?? ""} ${p.username ?? ""}`.toLowerCase();
        if (!hay.includes(q)) return false;
      }
      return true;
    });
  }, [proxies, search, cityFilter, statusFilter]);

  const filterActive = search.trim() !== "" || cityFilter !== "all" || statusFilter !== "all";

  const handleAssignGroup = async () => {
    if (filtered.length === 0) return;
    setAssigning(true);
    try {
      const { data } = await api.post<{ assigned: number }>("/proxies/assign-group", {
        proxyIds: filtered.map((p) => p.id),
        proxyGroupId: assignGroupId === "none" ? null : Number(assignGroupId),
      });
      toast.success(tr(assignGroupId === "none" ? "uiText.settings.proxy.removedFromGroup" : "uiText.settings.proxy.assignedToGroup", { count: data.assigned }));
      setAssignOpen(false);
      fetchProxies();
      fetchGroups();
    } catch (err) {
      toastError(err);
    } finally {
      setAssigning(false);
    }
  };

  const handleTest = async (p: Proxy) => {
    setTestingId(p.id);
    try {
      const { data } = await api.post<{ ok: boolean; ip?: string; city?: string; country?: string; latencyMs?: number; error?: string }>(`/proxies/${p.id}/test`);
      if (data.ok) {
        const loc = [data.city, data.country].filter(Boolean).join(", ");
        toast.success(`Proxy OK — IP ${data.ip}${loc ? ` · ${loc}` : ""} (${data.latencyMs}ms)`);
      } else {
        toast.error(tr("uiText.settings.proxy.failed", { error: data.error || tr("uiText.settings.proxy.noResponse") }));
      }
      fetchProxies();
    } catch (err) {
      toastError(err);
    } finally {
      setTestingId(null);
    }
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 space-y-0">
          <div>
            <CardTitle className="flex items-center gap-2 text-primary">
              <Network className="h-5 w-5" />
              Proxy
            </CardTitle>
            <CardDescription>
              Pool de proxies para isolamento de IP por conexão (anti-ban). Atrele um proxy a uma conexão na página de Conexões.
            </CardDescription>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button variant="outline" onClick={() => setConnGroupsOpen(true)}>
              <Network className="mr-2 h-4 w-4" /> Grupos de Conexões
            </Button>
            <Button variant="outline" onClick={handleTestAll} disabled={testingAll}>
              {testingAll ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Gauge className="mr-2 h-4 w-4" />}
              Testar todos
            </Button>
            <Button variant="outline" className="text-destructive hover:text-destructive" onClick={handleDeleteAll} disabled={deletingAll}>
              {deletingAll ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Trash2 className="mr-2 h-4 w-4" />}
              Deletar todos
            </Button>
            <Button variant="outline" onClick={() => setImportOpen(true)}>
              <Upload className="mr-2 h-4 w-4" /> Importar em massa
            </Button>
            <Button onClick={openCreate}>
              <Plus className="mr-2 h-4 w-4" /> Novo proxy
            </Button>
          </div>
        </CardHeader>

        <CardContent className="space-y-4">
          {/* Filtros + contagem + atribuição em massa por grupo */}
          <div className="flex flex-col lg:flex-row lg:items-end gap-3">
            <div className="relative flex-1 min-w-[180px]">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder={tr("uiText.settings.proxy.searchPlaceholder")}
                className="pl-8"
              />
            </div>
            <div className="w-full sm:w-48 space-y-1">
              <Label className="text-xs text-muted-foreground">{tr("uiText.clients.city")}</Label>
              <Select value={cityFilter} onValueChange={setCityFilter}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Todas as cidades</SelectItem>
                  {cities.map((c) => (<SelectItem key={c} value={c}>{c}</SelectItem>))}
                </SelectContent>
              </Select>
            </div>
            <div className="w-full sm:w-40 space-y-1">
              <Label className="text-xs text-muted-foreground">{tr("uiText.common.status")}</Label>
              <Select value={statusFilter} onValueChange={setStatusFilter}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">{tr("uiText.common.allMasc")}</SelectItem>
                  <SelectItem value="active">{tr("uiText.settings.proxy.statusActive")}</SelectItem>
                  <SelectItem value="isolated">{tr("uiText.settings.proxy.statusIsolated")}</SelectItem>
                  <SelectItem value="disabled">{tr("uiText.settings.proxy.statusDisabled")}</SelectItem>
                  <SelectItem value="banned">Banido</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <Button
              variant="outline"
              disabled={filtered.length === 0}
              onClick={() => { setAssignGroupId("none"); setAssignOpen(true); }}
              title={tr("uiText.settings.proxy.assignFiltered")}
            >
              <FolderPlus className="mr-2 h-4 w-4" /> {tr("uiText.settings.proxy.assignToGroup")}
            </Button>
          </div>

          <p className="text-xs text-muted-foreground">
            Mostrando <strong>{filtered.length}</strong> de <strong>{proxies.length}</strong> proxies
            {filterActive && cityFilter !== "all" ? tr("uiText.settings.proxy.inCity", { city: cityFilter }) : ""}.
          </p>

          {loading ? (
            <div className="flex justify-center py-16">
              <Loader2 className="h-6 w-6 animate-spin text-primary" />
            </div>
          ) : proxies.length === 0 ? (
            <div className="text-center text-sm text-muted-foreground py-16">
              {tr("uiText.settings.proxy.none")}
            </div>
          ) : filtered.length === 0 ? (
            <div className="text-center text-sm text-muted-foreground py-16">
              {tr("uiText.settings.proxy.noMatch")}
            </div>
          ) : (
        <div className="rounded-2xl shadow-[0px_4px_20px_rgba(0,0,0,0.08)] bg-card overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-12 text-muted-foreground">#</TableHead>
                <TableHead>{tr("uiText.settings.proxy.label")}</TableHead>
                <TableHead>Endpoint</TableHead>
                <TableHead>{tr("uiText.settings.proxy.location")}</TableHead>
                <TableHead>Grupo</TableHead>
                <TableHead>{tr("uiText.settings.proxy.user")}</TableHead>
                <TableHead>{tr("uiText.common.status")}</TableHead>
                <TableHead className="text-right">{tr("uiText.common.actions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filtered.map((p, idx) => {
                const badge = STATUS_BADGE[p.status] ?? { label: p.status, variant: "outline" as const };
                const isIsolated = p.status === "isolated";
                return (
                  <TableRow key={p.id}>
                    <TableCell className="text-xs text-muted-foreground tabular-nums">{idx + 1}</TableCell>
                    <TableCell className="font-medium">{p.label || "—"}</TableCell>
                    <TableCell className="font-mono text-xs">{p.scheme}://{p.host}:{p.port}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {p.city || p.country ? (
                        <span>{[p.city, p.countryCode || p.country].filter(Boolean).join(", ")}</span>
                      ) : (
                        <span className="text-muted-foreground/50">—</span>
                      )}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">{groupName(p.proxyGroupId)}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">{p.username || "—"}</TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1.5">
                        <Badge variant={badge.variant}>{STATUS_BADGE[p.status] ? tr(badge.label) : badge.label}</Badge>
                        <span
                          className={`inline-block h-2 w-2 rounded-full ${p.healthy ? "bg-emerald-500" : "bg-muted-foreground/30"}`}
                          title={p.healthy ? tr("uiText.settings.proxy.lastTestOk") : tr("uiText.settings.proxy.notTested")}
                        />
                      </div>
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-1">
                        <Button variant="ghost" size="icon" className="h-8 w-8" title={tr("uiText.settings.proxy.testConnection")} disabled={testingId === p.id} onClick={() => handleTest(p)}>
                          {testingId === p.id ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Wifi className="h-3.5 w-3.5" />}
                        </Button>
                        <Button variant="ghost" size="icon" className="h-8 w-8" title={tr("uiText.common.edit")} onClick={() => openEdit(p)}>
                          <Pencil className="h-3.5 w-3.5" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8"
                          title={isIsolated ? "Reativar" : "Isolar (IP queimado)"}
                          onClick={() => handleToggleIsolate(p)}
                        >
                          {isIsolated ? <ShieldCheck className="h-3.5 w-3.5 text-emerald-600" /> : <ShieldAlert className="h-3.5 w-3.5 text-amber-600" />}
                        </Button>
                        <Button variant="ghost" size="icon" className="h-8 w-8" title={tr("uiText.common.remove")} onClick={() => handleDelete(p)}>
                          <Trash2 className="h-3.5 w-3.5 text-muted-foreground" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
          )}
        </CardContent>
      </Card>

      <ProxyGroupsPanel onChanged={() => { fetchProxies(); fetchGroups(); }} />

      <ProxyFormDialog open={formOpen} proxy={editing} groups={groups} onClose={() => setFormOpen(false)} onSaved={fetchProxies} />
      <ProxyImportDialog open={importOpen} groups={groups} onClose={() => setImportOpen(false)} onImported={fetchProxies} />
      <ConnectionGroupsDialog open={connGroupsOpen} onClose={() => setConnGroupsOpen(false)} />

      <Dialog open={assignOpen} onOpenChange={setAssignOpen}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>{tr("uiText.settings.proxy.assignToGroup")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground">
              Os <strong>{filtered.length}</strong> proxies filtrados
              {cityFilter !== "all" ? ` (cidade: ${cityFilter})` : ""} serão atribuídos ao grupo escolhido.
            </p>
            <div className="space-y-1">
              <Label>Grupo</Label>
              <Select value={assignGroupId} onValueChange={setAssignGroupId}>
                <SelectTrigger className="w-full"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">{tr("uiText.settings.proxy.noGroupRemove")}</SelectItem>
                  {groups.map((g) => (<SelectItem key={g.id} value={String(g.id)}>{g.name}</SelectItem>))}
                </SelectContent>
              </Select>
              {groups.length === 0 && (
                <p className="text-xs text-muted-foreground">{tr("uiText.settings.proxy.createGroupFirst")}</p>
              )}
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setAssignOpen(false)}>{tr("uiText.common.cancel")}</Button>
            <Button onClick={handleAssignGroup} disabled={assigning || filtered.length === 0}>
              {assigning && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Atribuir
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
};

export default ProxySection;
