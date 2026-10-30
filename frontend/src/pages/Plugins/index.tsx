import React, { useCallback, useContext, useEffect, useState } from "react";
import { toast } from "react-toastify";
import { Puzzle } from "lucide-react";

import api from "../../services/api";
import toastError from "../../errors/toastError";
import { AuthContext } from "../../context/Auth/AuthContext";
import Can from "../../components/Can";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { PageContainer, PageContent, PageHeader } from "@/components/ui/page-layout";
import { Switch } from "@/components/ui/switch";
import { EMBEDDED_PLUGINS } from "./embeddedPlugins";

import { i18n } from "../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
// Plugins embarcados no Watink. Não há marketplace nem licença: o administrador da empresa liga ou desliga cada um, e a
// alocação por empresa (PluginInstallations) é o único gate de runtime no backend.
const Plugins: React.FC = () => {
  const { user } = useContext(AuthContext);
  const [active, setActive] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const { data } = await api.get<{ active?: string[] }>("/plugins/installed");
      setActive(new Set(Array.isArray(data?.active) ? data.active : []));
    } catch (err) {
      toastError(err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const toggle = async (slug: string, next: boolean) => {
    setBusy(slug);
    try {
      await api.post(`/plugins/${slug}/${next ? "activate" : "deactivate"}`);
      setActive((prev) => {
        const copy = new Set(prev);
        if (next) copy.add(slug);
        else copy.delete(slug);
        return copy;
      });
      toast.success(next ? "Plugin ativado." : "Plugin desativado.");
    } catch (err) {
      toastError(err);
    } finally {
      setBusy(null);
    }
  };

  return (
    <Can
      user={user}
      perform="settings:update"
      no={() => (
        <PageContainer>
          <PageContent>
            <div className="flex h-64 items-center justify-center text-muted-foreground">
              {tr("uiText.plugins.noPermission")}
            </div>
          </PageContent>
        </PageContainer>
      )}
      yes={() => (
        <PageContainer>
          <PageHeader title="Plugins" description={tr("uiText.plugins.pageDescription")} />
          <PageContent>
            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3" aria-busy={loading}>
              {EMBEDDED_PLUGINS.map((plugin) => {
                const isActive = active.has(plugin.slug);
                return (
                  <Card key={plugin.slug} data-testid={`plugin-${plugin.slug}`}>
                    <CardHeader className="flex flex-row items-start justify-between gap-3 space-y-0">
                      <div className="flex items-center gap-3">
                        <Puzzle className="h-5 w-5 text-muted-foreground" aria-hidden="true" />
                        <CardTitle className="text-base">{tr(plugin.name)}</CardTitle>
                      </div>
                      <Switch
                        checked={isActive}
                        disabled={loading || busy === plugin.slug}
                        onCheckedChange={(next) => void toggle(plugin.slug, next)}
                        aria-label={tr(isActive ? "uiText.plugins.deactivate" : "uiText.plugins.activate", { name: tr(plugin.name) })}
                      />
                    </CardHeader>
                    <CardContent className="space-y-3">
                      <CardDescription>{tr(plugin.description)}</CardDescription>
                      <Badge variant={isActive ? "default" : "outline"}>{tr(isActive ? "uiText.plugins.active" : "uiText.plugins.inactive")}</Badge>
                    </CardContent>
                  </Card>
                );
              })}
            </div>
          </PageContent>
        </PageContainer>
      )}
    />
  );
};

export default Plugins;
