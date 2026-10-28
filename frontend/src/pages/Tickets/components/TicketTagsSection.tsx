import React, { useEffect, useMemo, useState } from "react";
import { Tags as TagsIcon, Plus, Search } from "lucide-react";
import { toast } from "react-toastify";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import TagChip, { Tag } from "../../../components/TagChip";
import { TAG_COLOR_OPTIONS } from "../../../helpers/tagColors";
import api from "../../../services/api";
import { i18n } from "../../../translate/i18n";

// i18n.t() deste projeto pode devolver null; a tela só quer string.
const tr = (key: string, opts: Record<string, string | number> = {}): string => String(i18n.t(key, opts) ?? "");

interface TicketTagsSectionProps {
  ticketId: number;
}

const randomTagColor = () =>
  TAG_COLOR_OPTIONS[Math.floor(Math.random() * TAG_COLOR_OPTIONS.length)];

const TicketTagsSection: React.FC<TicketTagsSectionProps> = ({ ticketId }) => {
  const [allTags, setAllTags] = useState<Tag[]>([]);
  const [ticketTags, setTicketTags] = useState<Tag[]>([]);
  const [open, setOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [search, setSearch] = useState("");
  const [creating, setCreating] = useState(false);
  const queryClient = useQueryClient();

  const filteredTags = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return allTags;
    return allTags.filter((tag) => tag.name.toLowerCase().includes(q));
  }, [allTags, search]);

  const exactMatch = useMemo(
    () => allTags.some((tag) => tag.name.toLowerCase() === search.trim().toLowerCase()),
    [allTags, search],
  );

  useEffect(() => {
    api
      .get<Tag[]>(`/entities/ticket/${ticketId}/tags`)
      .then(({ data }) => setTicketTags(data))
      .catch(() => setTicketTags([]));
  }, [ticketId]);

  const openPopover = async (isOpen: boolean) => {
    setOpen(isOpen);
    if (!isOpen) {
      setSearch("");
      return;
    }
    if (allTags.length === 0) {
      try {
        const { data } = await api.get<Tag[]>("/tags");
        setAllTags(data);
      } catch {
        setAllTags([]);
      }
    }
  };

  const syncTags = async (nextTags: Tag[]) => {
    setSaving(true);
    try {
      await api.put(`/entities/ticket/${ticketId}/tags/sync`, {
        tagIds: nextTags.map((t) => t.id),
      });
      setTicketTags(nextTags);
      queryClient.invalidateQueries({ queryKey: ["tickets"] });
    } catch {
      toast.error(tr("tickets.details.tags.updateError"));
    } finally {
      setSaving(false);
    }
  };

  const toggleTag = (tag: Tag) => {
    const has = ticketTags.some((t) => t.id === tag.id);
    const next = has
      ? ticketTags.filter((t) => t.id !== tag.id)
      : [...ticketTags, tag];
    syncTags(next);
  };

  const removeTag = (tag: Tag) => {
    syncTags(ticketTags.filter((t) => t.id !== tag.id));
  };

  const createTag = async () => {
    const name = search.trim();
    if (!name || creating) return;
    setCreating(true);
    try {
      const { data } = await api.post<Tag>("/tags", { name, color: randomTagColor() });
      setAllTags((prev) => [...prev, data]);
      setSearch("");
      await syncTags([...ticketTags, data]);
    } catch {
      toast.error(tr("tickets.details.tags.createError"));
    } finally {
      setCreating(false);
    }
  };

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide flex items-center gap-1">
          <TagsIcon className="w-3 h-3" />
          {tr("tickets.details.tags.title")}
        </p>
        <Popover open={open} onOpenChange={openPopover}>
          <PopoverTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="h-5 w-5"
              title={tr("tickets.details.tags.add")}
              disabled={saving}
            >
              <Plus className="w-3.5 h-3.5" />
            </Button>
          </PopoverTrigger>
          <PopoverContent className="w-56 p-1" align="end">
            <div className="relative px-1 py-1">
              <Search className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
              <Input
                autoFocus
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && filteredTags.length === 0 && !exactMatch) {
                    createTag();
                  }
                }}
                placeholder={tr("tickets.details.tags.search")}
                className="h-8 pl-8 text-xs"
              />
            </div>

            <div className="max-h-56 overflow-y-auto">
              {allTags.length === 0 && (
                <p className="px-2 py-1.5 text-xs text-muted-foreground">{tr("tickets.details.tags.noneRegistered")}</p>
              )}
              {filteredTags.map((tag) => (
                <label
                  key={tag.id}
                  className="flex items-center gap-2 rounded px-2 py-1.5 text-sm cursor-pointer hover:bg-accent"
                >
                  <Checkbox
                    checked={ticketTags.some((t) => t.id === tag.id)}
                    onCheckedChange={() => toggleTag(tag)}
                    disabled={saving}
                  />
                  <span className="flex-1 truncate">{tag.name}</span>
                </label>
              ))}
            </div>

            {search.trim() && !exactMatch && (
              <button
                type="button"
                onClick={createTag}
                disabled={creating}
                className="flex w-full items-center gap-2 rounded px-2 py-1.5 text-sm text-primary hover:bg-accent disabled:opacity-50"
              >
                <Plus className="h-3.5 w-3.5 shrink-0" />
                <span className="truncate">
                  {creating ? tr("tickets.details.tags.creating") : tr("tickets.details.tags.create", { name: search.trim() })}
                </span>
              </button>
            )}
          </PopoverContent>
        </Popover>
      </div>

      {ticketTags.length === 0 ? (
        <p className="text-xs text-muted-foreground">{tr("tickets.details.tags.noneAdded")}</p>
      ) : (
        <div className="flex flex-wrap gap-1.5">
          {ticketTags.map((tag) => (
            <TagChip
              key={tag.id}
              tag={tag}
              size="small"
              onRemove={saving ? undefined : removeTag}
            />
          ))}
        </div>
      )}
    </div>
  );
};

export default TicketTagsSection;
