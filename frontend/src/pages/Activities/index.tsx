/* @jsxImportSource react */
import React, { useContext } from "react";
import { Search, Plus, ClipboardList } from "lucide-react";

import { PageContainer, PageHeader, PageContent } from "@/components/ui/page-layout";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import { Can } from "../../components/Can";
import { AuthContext } from "../../context/Auth/AuthContext";
import ConfirmationModal from "../../components/ConfirmationModal";
import { ACTIVITY_PRIORITY_LABELS, ACTIVITY_STATUS_LABELS } from "../MyActivities/activityTypes";
import { ActivitiesTable } from "./components/ActivitiesTable";
import ActivityFormDialog from "./components/ActivityFormDialog";
import { useActivities } from "./hooks/useActivities";

import { i18n } from "../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
// Tela de gestão de Atividades — lista do tenant inteiro (GET /activities),
// distinta de "Minhas Atividades" (visão do executor, filtro por
// assignee). Atribuição N:N e montador de checklist ficam em issue seguinte.
const Activities: React.FC = () => {
  const { user } = useContext(AuthContext);
  const {
    activities, loading, error,
    searchParam, setSearchParam, statusFilter, setStatusFilter, priorityFilter, setPriorityFilter,
    formOpen, selectedActivity, confirmDeleteOpen, activityToDelete,
    handleOpenForm, handleCloseForm, handleDeleteClick, handleConfirmDelete, setConfirmDeleteOpen,
    refetch,
  } = useActivities();

  return (
    <Can
      user={user}
      perform="activities:manage"
      yes={() => (
        <PageContainer>
          <PageHeader
            title={
              <span className="flex items-center gap-2">
                <ClipboardList className="h-5 w-5 text-muted-foreground" />
                Atividades
              </span>
            }
            description={tr("uiText.activities.pageDescription")}
          >
            <div className="flex items-center gap-2 flex-wrap">
              <Select value={statusFilter || "all"} onValueChange={(v) => setStatusFilter(v === "all" ? "" : v)}>
                <SelectTrigger className="w-[160px]">
                  <SelectValue placeholder={tr("uiText.common.status")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">{tr("uiText.activities.allStatuses")}</SelectItem>
                  {Object.entries(ACTIVITY_STATUS_LABELS).map(([value, label]) => (
                    <SelectItem key={value} value={value}>{tr(label)}</SelectItem>
                  ))}
                </SelectContent>
              </Select>

              <Select value={priorityFilter || "all"} onValueChange={(v) => setPriorityFilter(v === "all" ? "" : v)}>
                <SelectTrigger className="w-[160px]">
                  <SelectValue placeholder={tr("uiText.common.priority")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Todas as prioridades</SelectItem>
                  {Object.entries(ACTIVITY_PRIORITY_LABELS).map(([value, label]) => (
                    <SelectItem key={value} value={value}>{tr(label)}</SelectItem>
                  ))}
                </SelectContent>
              </Select>

              <div className="relative hidden md:block">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder={tr("uiText.activities.searchTitle")}
                  value={searchParam}
                  onChange={(e) => setSearchParam(e.target.value)}
                  className="pl-9 h-10 w-64"
                />
              </div>

              <Button onClick={() => handleOpenForm()}>
                <Plus className="mr-2 h-4 w-4" />
                {tr("uiText.activities.new")}
              </Button>
            </div>
          </PageHeader>

          <PageContent className="p-0">
            <div className="p-6">
              <ActivitiesTable
                activities={activities}
                loading={loading}
                error={error}
                onRetry={refetch}
                user={user}
                onEdit={handleOpenForm}
                onDelete={handleDeleteClick}
              />
            </div>
          </PageContent>

          <ActivityFormDialog
            open={formOpen}
            activity={selectedActivity}
            onClose={handleCloseForm}
          />

          <ConfirmationModal
            title={tr("uiText.activities.deleteTitle")}
            open={confirmDeleteOpen}
            onClose={() => setConfirmDeleteOpen(false)}
            onConfirm={handleConfirmDelete}
          >
            {activityToDelete
              ? tr("uiText.activities.deleteConfirm", { title: activityToDelete.title })
              : ""}
          </ConfirmationModal>
        </PageContainer>
      )}
      no={() => (
        <PageContainer>
          <PageContent>
            <p className="text-center text-muted-foreground py-16">
              {tr("uiText.activities.noPermission")}
            </p>
          </PageContent>
        </PageContainer>
      )}
    />
  );
};

export default Activities;
