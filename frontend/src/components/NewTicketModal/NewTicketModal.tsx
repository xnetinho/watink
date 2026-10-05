import React, { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Loader2 } from "lucide-react";
import { useStartChat } from "../../hooks/useStartChat";
import { i18n } from "../../translate/i18n";

interface NewTicketModalProps {
  modalOpen: boolean;
  onClose: () => void;
}

export const NewTicketModal: React.FC<NewTicketModalProps> = ({ modalOpen, onClose }) => {
  const [searchParam, setSearchParam] = useState("");
  const { startingChat: loading, connectionDialog } = useStartChat(onClose);

  return (
    <Dialog open={modalOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>{i18n.t("newTicketModal.title")}</DialogTitle>
        </DialogHeader>
        <div className="grid gap-4 py-4">
          <Input
            placeholder={i18n.t("newTicketModal.fieldLabel") as string}
            value={searchParam}
            onChange={(e) => setSearchParam(e.target.value)}
            autoFocus
          />
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={loading}>
            {i18n.t("newTicketModal.buttons.cancel")}
          </Button>
          <Button onClick={() => {}} disabled={loading}>
            {loading ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : i18n.t("newTicketModal.buttons.ok")}
          </Button>
        </DialogFooter>
      </DialogContent>
      {connectionDialog}
    </Dialog>
  );
};

export default NewTicketModal;
