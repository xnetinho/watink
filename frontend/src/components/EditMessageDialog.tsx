import React, { useEffect, useState } from "react";
import api from "../services/api";
import { i18n } from "../translate/i18n";
import toastError from "../errors/toastError";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "./ui/dialog";
import { Button } from "./ui/button";
import { Textarea } from "./ui/textarea";

const tr = (key: string): string => String(i18n.t(key) ?? "");

interface Props {
  messageId: string | number;
  initialBody: string;
  /** Legenda de mídia em vez de texto: muda só o rótulo do título. */
  isCaption?: boolean;
  onClose: () => void;
}

/** Edita uma mensagem já enviada. O servidor valida o prazo e as regras; a tela avisa se ele recusar. */
const EditMessageDialog: React.FC<Props> = ({ messageId, initialBody, isCaption = false, onClose }) => {
  const [body, setBody] = useState(initialBody);
  const [saving, setSaving] = useState(false);
  useEffect(() => setBody(initialBody), [initialBody]);

  const unchanged = body.trim() === initialBody.trim();
  const canSave = !saving && body.trim() !== "" && !unchanged;

  const save = async () => {
    setSaving(true);
    try {
      await api.post(`/message/${messageId}/edit`, { body: body.trim() });
      onClose();
    } catch (err) {
      toastError(err);
      setSaving(false);
    }
  };

  return (
    <Dialog open onOpenChange={(o) => !o && !saving && onClose()}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{tr(isCaption ? "messageOptionsMenu.editDialog.titleCaption" : "messageOptionsMenu.editDialog.title")}</DialogTitle>
        </DialogHeader>
        <Textarea
          autoFocus
          value={body}
          rows={5}
          onChange={(e) => setBody(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && (e.ctrlKey || e.metaKey) && canSave) void save();
          }}
          aria-label={tr("messageOptionsMenu.editDialog.field")}
          data-testid="edit-message-text"
        />
        <p className="text-xs text-[hsl(var(--text-muted))]">{tr("messageOptionsMenu.editDialog.hint")}</p>
        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={saving}>
            {tr("messageOptionsMenu.editDialog.cancel")}
          </Button>
          <Button onClick={() => void save()} disabled={!canSave} data-testid="edit-message-save">
            {tr("messageOptionsMenu.editDialog.save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default EditMessageDialog;
