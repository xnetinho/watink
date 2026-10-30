/* @jsxImportSource react */
import React from "react";
import { PageContainer, PageHeader, PageContent } from "../../components/ui/page-layout";
import ConfirmationModal from "../../components/ConfirmationModal";
import { useFlowManager } from "./hooks/useFlowManager";
import FlowFormDialog from "./components/FlowFormDialog";
import FlowToolbar from "./components/FlowToolbar";
import FlowGrid from "./components/FlowGrid";

import { i18n } from "../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
const FlowManager: React.FC = () => {
  const {
    whatsapps,
    loading,
    searchParam,
    setSearchParam,
    view,
    setView,
    openModal,
    setOpenModal,
    selectedFlow,
    newFlowName,
    setNewFlowName,
    selectedWhatsapp,
    setSelectedWhatsapp,
    confirmModalOpen,
    setConfirmModalOpen,
    deletingFlow,
    filteredFlows,
    isConnectionUsed,
    handleOpenModal,
    handleCloseModal,
    handleSaveFlow,
    handleDeleteFlow,
    handleRequestDelete,
    handleToggleActive,
  } = useFlowManager();

  return (
    <PageContainer>
      <ConfirmationModal
        title={
          deletingFlow
            ? `Remover Fluxo: ${deletingFlow.name}?`
            : "Remover Fluxo?"
        }
        open={confirmModalOpen}
        onClose={() => setConfirmModalOpen(false)}
        onConfirm={handleDeleteFlow}
      >
        {tr("uiText.flowManager.deleteHint")}
      </ConfirmationModal>

      <FlowFormDialog
        open={openModal}
        onOpenChange={setOpenModal}
        selectedFlow={selectedFlow}
        newFlowName={newFlowName}
        onFlowNameChange={setNewFlowName}
        selectedWhatsapp={selectedWhatsapp}
        onWhatsappChange={setSelectedWhatsapp}
        whatsapps={whatsapps}
        isConnectionUsed={isConnectionUsed}
        onSave={handleSaveFlow}
        onClose={handleCloseModal}
      />

      <PageHeader
        title={tr("uiText.flowManager.title")}
        description={tr("uiText.flowManager.subtitle")}
      >
        <FlowToolbar
          searchParam={searchParam}
          onSearchChange={setSearchParam}
          view={view}
          onViewChange={setView}
          onCreateNew={() => handleOpenModal()}
        />
      </PageHeader>

      <PageContent>
        <FlowGrid
          loading={loading}
          filteredFlows={filteredFlows}
          onCreateNew={() => handleOpenModal()}
          onEdit={handleOpenModal}
          onDelete={handleRequestDelete}
          onToggleActive={handleToggleActive}
        />
      </PageContent>
    </PageContainer>
  );
};

export default FlowManager;
