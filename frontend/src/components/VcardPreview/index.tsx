/* @jsxImportSource react */
import React, { useEffect, useState } from "react";
import toastError from "../../errors/toastError";
import api from "../../services/api";
import { useStartChat } from "../../hooks/useStartChat";
import { Button } from "../ui/button";
import { Avatar } from "../ui/avatar";
import { Separator } from "../ui/separator";
import { MessageCircle } from "lucide-react";

interface VcardPreviewProps {
  contact: string;
  numbers: string | undefined;
}

interface SelectedContact {
  id?: number;
  name: string;
  number: number | string;
  profilePicUrl: string;
}

const VcardPreview: React.FC<VcardPreviewProps> = ({ contact, numbers }) => {
  const { startChat, connectionDialog } = useStartChat();

  const [selectedContact, setContact] = useState<SelectedContact>({
    name: "",
    number: 0,
    profilePicUrl: "",
  });

  useEffect(() => {
    const delayDebounceFn = setTimeout(() => {
      const fetchContacts = async () => {
        try {
          const contactObj = {
            name: contact,
            number: numbers !== undefined ? numbers.replace(/\D/g, "") : "",
            email: "",
          };
          const { data } = await api.post("/contact", contactObj);
          setContact(data);
        } catch (err) {
          toastError(err);
        }
      };
      fetchContacts();
    }, 500);
    return () => clearTimeout(delayDebounceFn);
  }, [contact, numbers]);

  const handleNewChat = () => {
    if (selectedContact.id !== undefined) startChat(selectedContact.id);
  };

  return (
    <div className="min-w-[250px]">
      <div className="flex items-center gap-3 p-2">
        <Avatar className="h-8 w-8 shrink-0" src={selectedContact.profilePicUrl} name={selectedContact.name} />
        <span className="text-sm font-medium text-[var(--action-primary)] mt-1 ml-1 truncate">
          {selectedContact.name}
        </span>
      </div>
      <Separator />
      <Button
        variant="ghost"
        className="w-full gap-2 text-[var(--action-primary)] hover:text-[var(--action-primary-hover)]"
        onClick={handleNewChat}
        disabled={!selectedContact.number}
      >
        <MessageCircle className="h-4 w-4" />
        Conversar
      </Button>
      {connectionDialog}
    </div>
  );
};

export default VcardPreview;
