/* @jsxImportSource react */
import React, { useState } from "react";
import { Loader2, MessageCircle } from "lucide-react";
import toastError from "../../errors/toastError";
import api from "../../services/api";
import { useStartChat } from "../../hooks/useStartChat";
import { Button } from "../ui/button";
import { Avatar } from "../ui/avatar";
import { Separator } from "../ui/separator";

interface VcardPreviewProps {
  contact: string;
  numbers: string | undefined;
}

interface ContactRef {
  id: number;
  number?: string;
}

const onlyDigits = (v: string | undefined): string => (v ?? "").replace(/\D/g, "");

// Procura o contato pelo número na agenda; só cria se não existir. Mostrar o
// cartão não cria nada: o contato nasce quando o atendente clica em "Conversar".
export const findOrCreateContact = async (name: string, number: string): Promise<number> => {
  const { data } = await api.get<{ contacts?: ContactRef[] }>("/contacts/", {
    params: { searchParam: number },
  });
  const existing = (data.contacts ?? []).find((c) => onlyDigits(c.number) === number);
  if (existing) return existing.id;

  try {
    const { data: created } = await api.post<ContactRef>("/contacts", { name: name || number, number, email: "" });
    return created.id;
  } catch (err) {
    // Outro atendente pode ter cadastrado entre a busca e o POST (409): busca de novo.
    const status = (err as { response?: { status?: number } }).response?.status;
    if (status !== 409) throw err;
    const { data: again } = await api.get<{ contacts?: ContactRef[] }>("/contacts/", {
      params: { searchParam: number },
    });
    const found = (again.contacts ?? []).find((c) => onlyDigits(c.number) === number);
    if (!found) throw err;
    return found.id;
  }
};

const VcardPreview: React.FC<VcardPreviewProps> = ({ contact, numbers }) => {
  const { startChat, connectionDialog } = useStartChat();
  const [busy, setBusy] = useState(false);
  const number = onlyDigits(numbers);

  const handleNewChat = async () => {
    if (!number) return;
    setBusy(true);
    try {
      const id = await findOrCreateContact(contact, number);
      await startChat(id);
    } catch (err) {
      toastError(err);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="min-w-[250px]">
      <div className="flex items-center gap-3 p-2">
        <Avatar className="h-8 w-8 shrink-0" src="" name={contact} />
        <span className="text-sm font-medium text-[var(--action-primary)] mt-1 ml-1 truncate">{contact}</span>
      </div>
      <Separator />
      <Button
        variant="ghost"
        className="w-full gap-2 text-[var(--action-primary)] hover:text-[var(--action-primary-hover)]"
        onClick={handleNewChat}
        disabled={!number || busy}
      >
        {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <MessageCircle className="h-4 w-4" />}
        Conversar
      </Button>
      {connectionDialog}
    </div>
  );
};

export default VcardPreview;
