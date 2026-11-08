import React, { createContext, useCallback, useContext, useState } from "react";

interface DockValue {
  /** O elemento do rodapé da lista de tickets, quando a lista está na tela. */
  host: HTMLElement | null;
  setHost: (el: HTMLElement | null) => void;
}

const DockContext = createContext<DockValue>({ host: null, setHost: () => undefined });

/** Guarda onde os cartões de chamada se encaixam: no rodapé da lista de tickets quando ela existe, senão flutuam. */
export const CallCardDockProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [host, setHostState] = useState<HTMLElement | null>(null);
  const setHost = useCallback((el: HTMLElement | null) => setHostState(el), []);
  return <DockContext.Provider value={{ host, setHost }}>{children}</DockContext.Provider>;
};

export const useCallCardDock = () => useContext(DockContext);

/** Rodapé da lista de tickets: marca o lugar onde os cartões de chamada se encaixam (a lista encolhe para abrir espaço). */
export const CallCardDockSlot: React.FC = () => {
  const { setHost } = useCallCardDock();
  return <div ref={setHost} data-testid="call-card-dock" className="shrink-0 empty:hidden space-y-2 border-t border-border bg-card/60 p-2" />;
};
