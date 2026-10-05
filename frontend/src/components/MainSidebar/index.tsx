/* @jsxImportSource react */
import React from "react";
import { cn } from "@/lib/utils";
import VersionFooter from "../VersionFooter";
import { useMainSidebar } from "./hooks/useMainSidebar";
import SidebarHeader from "./components/SidebarHeader";
import SidebarNav from "./components/SidebarNav";
import type { MainSidebarProps } from "./mainSidebarTypes";

// Cores vêm dos tokens da paleta/modo (--bg-sidebar, --border-sidebar), então a
// sidebar acompanha o tema claro/escuro do usuário em qualquer paleta.
const getSidebarClass = (collapsed: boolean): string =>
  cn(
    "flex flex-col h-full transition-all duration-300 relative z-20",
    "bg-[hsl(var(--bg-sidebar))] border-r border-[hsl(var(--border-sidebar))]",
    collapsed ? "w-[70px]" : "w-[200px]"
  );

const getFooterClass = (): string =>
  "mt-auto border-t border-[hsl(var(--border-sidebar))] flex items-center justify-center px-3 py-2 min-h-[44px]";

const MainSidebar: React.FC<MainSidebarProps> = ({ collapsed, onToggle }) => {
  const { activePlugins, systemLogo, systemTitle, logoEnabled } = useMainSidebar();

  return (
    <aside className={getSidebarClass(collapsed)}>
      <SidebarHeader
        collapsed={collapsed}
        logoEnabled={logoEnabled}
        systemLogo={systemLogo}
        systemTitle={systemTitle}
        onToggle={onToggle}
      />

      <SidebarNav
        collapsed={collapsed}
        activePlugins={activePlugins}
      />

      <div className={getFooterClass()}>
        <VersionFooter collapsed={collapsed} />
      </div>
    </aside>
  );
};

export default MainSidebar;
