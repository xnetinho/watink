/* @jsxImportSource react */
import React from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "../../ui/button";
import { cn } from "@/lib/utils";
import { getBackendUrl } from "../../../helpers/urlUtils";
import type { SidebarHeaderProps } from "../mainSidebarTypes";

const HEADER_CLASS = "flex items-center h-16 min-h-16 border-b border-[hsl(var(--border-sidebar))] px-3 gap-2";

const TOGGLE_CLASS =
  "shrink-0 rounded-full h-8 w-8 text-[hsl(var(--text-sidebar))] hover:bg-[hsl(var(--text-sidebar)/0.1)]";

const SidebarHeader: React.FC<SidebarHeaderProps> = ({
  collapsed,
  logoEnabled,
  systemLogo,
  systemTitle,
  onToggle,
}) => {
  return (
    <div className={HEADER_CLASS}>
      <div className={cn("flex flex-1 items-center overflow-hidden", collapsed ? "justify-center" : "justify-start pl-1")}>
        {collapsed ? (
          <div className="w-8 h-8 bg-primary rounded-full shrink-0" />
        ) : logoEnabled && systemLogo ? (
          <img
            src={getBackendUrl(systemLogo)}
            alt={systemTitle}
            className="h-8 w-auto object-contain opacity-90 dark:brightness-0 dark:invert"
          />
        ) : (
          <h1 className="text-lg font-bold truncate text-[hsl(var(--text-sidebar))]">
            {systemTitle}
          </h1>
        )}
      </div>

      <Button
        variant="ghost"
        size="icon"
        onClick={onToggle}
        className={TOGGLE_CLASS}
        aria-label={collapsed ? "Expandir menu" : "Recolher menu"}
      >
        {collapsed ? <ChevronRight size={18} /> : <ChevronLeft size={18} />}
      </Button>
    </div>
  );
};

export default SidebarHeader;
