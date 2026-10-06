/* @jsxImportSource react */
import React, { useContext, useState } from "react";
import { Phone } from "lucide-react";
import { PageContainer, PageHeader, PageContent } from "@/components/ui/page-layout";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { AuthContext } from "../../context/Auth/AuthContext";
import { check } from "../../components/Can";
import ConfirmationModal from "../../components/ConfirmationModal";
import api from "@/services/api";
import toastError from "@/errors/toastError";
import notify from "@/lib/notify";
import { t } from "@/lib/calls/t";
import { CALL_STATUSES, type CallLogRow } from "./types";
import { useCallHistory } from "./hooks/useCallHistory";
import CallsTable from "./components/CallsTable";

/** Histórico de chamadas de voz. Rota protegida por calls:read (menu e API). */
const Calls: React.FC = () => {
  const { user } = useContext(AuthContext);
  const { filters, update, rows, total, loading, error, reload, removeRecordingLocally } = useCallHistory();
  const [toDelete, setToDelete] = useState<CallLogRow | null>(null);
  const canDelete = check(user, "calls:delete");
  const pages = Math.max(1, Math.ceil(total / filters.pageSize));

  const confirmDelete = async () => {
    if (!toDelete) return;
    const target = toDelete;
    setToDelete(null);
    try {
      await api.delete(`/calls/${encodeURIComponent(target.callId)}/recording`);
      removeRecordingLocally(target.callId);
      notify.success(t("calls.history.deleted"));
    } catch (err) {
      toastError(err);
    }
  };

  return (
    <PageContainer>
      <PageHeader
        title={
          <span className="flex items-center gap-2">
            <Phone className="h-5 w-5 text-muted-foreground" />
            {t("calls.history.title")}
          </span>
        }
      >
        <div className="flex flex-wrap items-center gap-2">
          <Select value={filters.status || "all"} onValueChange={(v) => update({ status: v === "all" ? "" : v })}>
            <SelectTrigger className="w-[170px]" aria-label={t("calls.history.filterStatus")}>
              <SelectValue placeholder={t("calls.history.filterStatus")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t("calls.history.all")}</SelectItem>
              {CALL_STATUSES.map((s) => (
                <SelectItem key={s} value={s}>{t(`calls.status.${s}`)}</SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select value={filters.direction || "all"} onValueChange={(v) => update({ direction: v === "all" ? "" : v })}>
            <SelectTrigger className="w-[160px]" aria-label={t("calls.history.filterDirection")}>
              <SelectValue placeholder={t("calls.history.filterDirection")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t("calls.history.all")}</SelectItem>
              <SelectItem value="incoming">{t("calls.history.directionIncoming")}</SelectItem>
              <SelectItem value="outgoing">{t("calls.history.directionOutgoing")}</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </PageHeader>

      <PageContent className="p-0">
        <div className="space-y-4 p-6">
          <CallsTable
            rows={rows}
            loading={loading}
            error={error}
            onRetry={() => void reload()}
            canDelete={canDelete}
            onDelete={setToDelete}
          />
          {pages > 1 && (
            <div className="flex items-center justify-end gap-2" data-testid="calls-pager">
              <Button variant="outline" size="sm" disabled={filters.page <= 1} onClick={() => update({ page: filters.page - 1 })}>
                ‹
              </Button>
              <span className="text-sm tabular-nums">{filters.page} / {pages}</span>
              <Button variant="outline" size="sm" disabled={filters.page >= pages} onClick={() => update({ page: filters.page + 1 })}>
                ›
              </Button>
            </div>
          )}
        </div>
      </PageContent>

      <ConfirmationModal
        title={t("calls.history.deleteConfirmTitle")}
        open={!!toDelete}
        onClose={() => setToDelete(null)}
        onConfirm={() => void confirmDelete()}
      >
        {t("calls.history.deleteConfirmText")}
      </ConfirmationModal>
    </PageContainer>
  );
};

export default Calls;
