/* @jsxImportSource react */
import React from "react";
import { NavLink, useLocation } from "react-router";
import { cn } from "@/lib/utils";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "../../components/ui/tooltip";

interface SidebarItemProps {
  to: string;
  icon: React.ReactNode;
  label: string;
  collapsed?: boolean;
  activeColor?: string;
}

// Pré-computa classes fora do JSX para evitar ambiguidade do parser TSX com hsl(var(--...))
// Cores vêm dos tokens da paleta/modo (--text-sidebar, --action-primary*), então o
// item acompanha o tema claro/escuro do usuário em qualquer paleta.
const getLinkClass = (isActive: boolean, collapsed: boolean): string => {
  const base = "flex items-center gap-3 px-3 py-2 rounded-lg transition-all duration-200 group relative select-none";
  const hover = "hover:bg-[hsl(var(--text-sidebar)/0.08)]";
  const state = isActive
    ? "bg-[hsl(var(--action-primary-bg))] text-[hsl(var(--action-primary))] font-semibold"
    : "text-[hsl(var(--text-sidebar))]";
  const layout = collapsed ? "justify-center px-2" : "";
  return cn(base, hover, state, layout);
};

const getIconClass = (isActive: boolean): string => {
  const base = "flex shrink-0 items-center justify-center transition-transform group-hover:scale-110";
  const color = isActive
    ? "text-[hsl(var(--action-primary))]"
    : "text-[hsl(var(--text-sidebar)/0.65)] group-hover:text-[hsl(var(--text-sidebar))]";
  return cn(base, color);
};

const SidebarItem: React.FC<SidebarItemProps> = ({
  to,
  icon,
  label,
  collapsed = false,
  activeColor = "var(--primary)",
}) => {
  const location = useLocation();
  const isActive = to === "/" ? location.pathname === "/" : location.pathname.startsWith(to);

  const linkClass = getLinkClass(isActive, collapsed);
  const iconClass = getIconClass(isActive);

  const content = (
    <NavLink to={to} className={linkClass}>
      {/* Active Indicator */}
      {isActive && (
        <div
          className="absolute left-0 w-[3px] h-6 rounded-r-full"
          style={{ backgroundColor: activeColor || "var(--color-info)" }}
        />
      )}

      <div className={iconClass}>
        {icon}
      </div>

      {!collapsed && (
        <span className="text-sm truncate animate-in fade-in slide-in-from-left-2 duration-300">
          {label}
        </span>
      )}
    </NavLink>
  );

  if (collapsed) {
    return (
      <Tooltip delayDuration={0}>
        <TooltipTrigger asChild>
          {content}
        </TooltipTrigger>
        <TooltipContent side="right" className="font-semibold">
          {label}
        </TooltipContent>
      </Tooltip>
    );
  }

  return content;
};

export default SidebarItem;
