import React from "react";
import { useParams, useNavigate, Navigate } from "react-router";
import { Users, Building2, ShieldCheck } from "lucide-react";

import { PageContainer, PageHeader, PageContent } from "../../components/ui/page-layout";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "../../components/ui/tabs";

import { ACESSOS_TABS, type AcessosTab } from "./acessosTypes";
import UsuariosTab from "./components/UsuariosTab";
import SetoresTab from "./components/SetoresTab";
import CargosTab from "./components/CargosTab";

import { i18n } from "../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
/**
 * Central de Acessos (ADR 0022) — substitui as 7 rotas antigas
 * (/users, /users/:userId, /groups, /groups/:groupId, /roles, /roles/:roleId,
 * /access) por uma única área com abas: Usuários · Setores · Cargos.
 *
 * Sub-rotas (/acessos/usuarios, /acessos/setores, /acessos/cargos) em vez de
 * query param — permite voltar/avançar do navegador trocar de aba
 * corretamente. Rota-base /acessos redireciona para /acessos/usuarios
 * (ver routes/index.tsx).
 */
const Acessos: React.FC = () => {
  const { tab } = useParams<{ tab: string }>();
  const navigate = useNavigate();

  const activeTab: AcessosTab = ACESSOS_TABS.includes(tab as AcessosTab)
    ? (tab as AcessosTab)
    : "usuarios";

  if (tab && !ACESSOS_TABS.includes(tab as AcessosTab)) {
    return <Navigate to="/acessos/usuarios" replace />;
  }

  return (
    <PageContainer>
      <PageHeader
        title="Acessos"
        description={tr("uiText.acessos.pageDescription")}
      />
      <PageContent>
        <Tabs
          value={activeTab}
          onValueChange={(value) => navigate(`/acessos/${value}`)}
          className="flex flex-col gap-4"
        >
          <TabsList>
            <TabsTrigger value="usuarios" className="gap-1.5">
              <Users className="h-4 w-4" />
              {tr("uiText.acessos.users")}
            </TabsTrigger>
            <TabsTrigger value="setores" className="gap-1.5">
              <Building2 className="h-4 w-4" />
              Setores
            </TabsTrigger>
            <TabsTrigger value="cargos" className="gap-1.5">
              <ShieldCheck className="h-4 w-4" />
              Cargos
            </TabsTrigger>
          </TabsList>

          <TabsContent value="usuarios">
            <UsuariosTab />
          </TabsContent>
          <TabsContent value="setores">
            <SetoresTab />
          </TabsContent>
          <TabsContent value="cargos">
            <CargosTab />
          </TabsContent>
        </Tabs>
      </PageContent>
    </PageContainer>
  );
};

export default Acessos;
