/* @jsxImportSource react */
import React from "react";
import { MessageCircle, Send, ShoppingBag, Camera, Tag, Video, Globe } from "lucide-react";
import { CHANNELS_CATALOG, ChannelKind } from "../config/channels";
import { cn } from "../lib/utils";

interface ChannelBadgeProps {
  channel?: ChannelKind | string;
  size?: "sm" | "md";
  showLabel?: boolean;
  className?: string;
}

export const ChannelBadge: React.FC<ChannelBadgeProps> = ({
  channel = "whatsapp",
  size = "sm",
  showLabel = true,
  className = "",
}) => {
  const def = CHANNELS_CATALOG.find((c) => c.kind === channel) || {
    kind: "whatsapp" as ChannelKind,
    name: "WhatsApp",
    colorClass: "text-emerald-600 dark:text-emerald-400",
    bgClass: "bg-emerald-500/15",
    borderClass: "border-emerald-500/30",
    availability: "available",
    authKind: "qr",
    description: "",
    hasPhone: true,
    iconName: "MessageCircle",
  };

  const getIcon = () => {
    const iconSize = size === "sm" ? 12 : 14;
    switch (def.kind) {
      case "whatsapp":
        return <MessageCircle size={iconSize} />;
      case "telegram":
        return <Send size={iconSize} />;
      case "mercadolivre":
        return <ShoppingBag size={iconSize} />;
      case "instagram":
        return <Camera size={iconSize} />;
      case "olx":
        return <Tag size={iconSize} />;
      case "tiktok":
        return <Video size={iconSize} />;
      default:
        return <Globe size={iconSize} />;
    }
  };

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium border shadow-xs",
        def.bgClass,
        def.colorClass,
        def.borderClass,
        className
      )}
    >
      {getIcon()}
      {showLabel && <span>{def.name}</span>}
    </span>
  );
};
