import React from "react";
import DropZone from "../MessageInput/DropZone";
import { useMediaDrop } from "../../context/MediaDrop/MediaDropContext";

/** Torna a conversa inteira (lista + campo de mensagem) uma área de soltar arquivos. */
const TicketDropZone: React.FC<{ canSend: boolean; children: React.ReactNode }> = ({ canSend, children }) => {
  const { deliver } = useMediaDrop();
  return (
    <DropZone onFiles={deliver} disabled={!canSend} className="relative flex min-h-0 flex-1 flex-col">
      {children}
    </DropZone>
  );
};

export default TicketDropZone;
