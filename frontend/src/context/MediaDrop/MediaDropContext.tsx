import React, { createContext, useCallback, useContext, useRef } from "react";

type Receiver = (files: File[]) => void;

interface MediaDropValue {
  /** O campo de mensagem registra aqui como receber arquivos soltos. Devolve a função que cancela o registro. */
  register: (receiver: Receiver) => () => void;
  /** Quem recebe o arraste (a área da conversa) entrega os arquivos por aqui. */
  deliver: (files: File[]) => void;
}

const noop: MediaDropValue = { register: () => () => undefined, deliver: () => undefined };
const MediaDropContext = createContext<MediaDropValue>(noop);

/**
 * Liga a área onde se solta o arquivo (a conversa inteira) ao campo de mensagem, que guarda os anexos. Os dois são
 * irmãos na árvore, então o campo se registra e a área entrega, sem subir o estado dos anexos para o pai.
 */
export const MediaDropProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const receiver = useRef<Receiver | null>(null);
  const register = useCallback((r: Receiver) => {
    receiver.current = r;
    return () => {
      if (receiver.current === r) receiver.current = null;
    };
  }, []);
  const deliver = useCallback((files: File[]) => receiver.current?.(files), []);
  const value = useRef({ register, deliver }).current;
  return <MediaDropContext.Provider value={value}>{children}</MediaDropContext.Provider>;
};

export const useMediaDrop = (): MediaDropValue => useContext(MediaDropContext);
