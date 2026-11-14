/* @jsxImportSource react */
import React from "react";
import { Plus } from "lucide-react";

import { i18n } from "../../translate/i18n";
import QueueModal from "../../components/QueueModal";
import ConfirmationModal from "../../components/ConfirmationModal";
import MoveBeforeDeleteModal from "../../components/MoveBeforeDeleteModal";
import { PageContainer, PageHeader, PageContent } from "../../components/ui/page-layout";
import { Button } from "../../components/ui/button";
import { useQueues } from "./hooks/useQueues";
import { QueuesTable } from "./components/QueuesTable";


const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
const Queues = () => {
  const {
    queues,
    loading,
    queueModalOpen,
    selectedQueue,
    confirmModalOpen,
    handleOpenQueueModal,
    handleCloseQueueModal,
    handleEditQueue,
    handleDeleteQueue,
    handleSetDefault,
    setSelectedQueue,
    setConfirmModalOpen,
    moveRequest,
    closeMoveRequest,
    handleMoveAndDelete,
  } = useQueues();

  return (
    <PageContainer>
      <ConfirmationModal
        title={
          selectedQueue
            ? `${i18n.t("queues.confirmationModal.deleteTitle")} ${selectedQueue.name}?`
            : ""
        }
        open={confirmModalOpen}
        onClose={() => setConfirmModalOpen(false)}
        onConfirm={() => selectedQueue && handleDeleteQueue(selectedQueue.id)}
      >
        {i18n.t("queues.confirmationModal.deleteMessage")}
      </ConfirmationModal>

      <MoveBeforeDeleteModal
        open={moveRequest !== null}
        title={i18n.t("moveBeforeDelete.queueTitle", { name: moveRequest?.queue.name ?? "" })}
        description={i18n.t("moveBeforeDelete.queueDescription", {
          tickets: moveRequest?.dependents.Tickets ?? 0,
          users: moveRequest?.dependents.Users ?? 0,
          connections: moveRequest?.dependents.Connections ?? 0,
          setores: moveRequest?.dependents.Setores ?? 0,
          children: moveRequest?.dependents.Children ?? 0,
        })}
        targetLabel={i18n.t("moveBeforeDelete.queueTarget")}
        targets={queues.filter((q) => q.id !== moveRequest?.queue.id)}
        onClose={closeMoveRequest}
        onConfirm={(target) => moveRequest && handleMoveAndDelete(moveRequest.queue.id, target)}
      />

      <QueueModal
        open={queueModalOpen}
        onClose={handleCloseQueueModal}
        queueId={selectedQueue?.id}
      />

      <PageHeader
        title={i18n.t("queues.title") as string}
        description={tr("uiText.queues.pageDescription")}
      >
        <Button onClick={handleOpenQueueModal}>
          <Plus className="mr-2 h-4 w-4" />
          {i18n.t("queues.buttons.add")}
        </Button>
      </PageHeader>

      <PageContent>
        <QueuesTable
          queues={queues}
          loading={loading}
          onEdit={handleEditQueue}
          onDelete={(queue) => {
            setSelectedQueue(queue);
            setConfirmModalOpen(true);
          }}
          onSetDefault={handleSetDefault}
        />
      </PageContent>
    </PageContainer>
  );
};

export default Queues;
