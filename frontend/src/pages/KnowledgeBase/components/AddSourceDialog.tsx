/* @jsxImportSource react */
import React from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Loader2 } from "lucide-react";

interface AddSourceDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  tab: "text" | "file" | "url";
  onTabChange: (tab: "text" | "file" | "url") => void;
  textName: string;
  onTextNameChange: (val: string) => void;
  textContent: string;
  onTextContentChange: (val: string) => void;
  urlName: string;
  onUrlNameChange: (val: string) => void;
  urlValue: string;
  onUrlValueChange: (val: string) => void;
  file: File | null;
  onFileChange: (file: File | null) => void;
  canSubmit: boolean;
  submitting: boolean;
  onSubmit: () => Promise<void>;
}

export const AddSourceDialog: React.FC<AddSourceDialogProps> = ({
  open,
  onOpenChange,
  tab,
  onTabChange,
  textName,
  onTextNameChange,
  textContent,
  onTextContentChange,
  urlName,
  onUrlNameChange,
  urlValue,
  onUrlValueChange,
  file,
  onFileChange,
  canSubmit,
  submitting,
  onSubmit,
}) => {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Adicionar fonte</DialogTitle>
        </DialogHeader>

        <Tabs
          value={tab}
          onValueChange={(v) => onTabChange(v as "text" | "file" | "url")}
          className="mt-2"
        >
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="text">Texto</TabsTrigger>
            <TabsTrigger value="url">URL</TabsTrigger>
            <TabsTrigger value="file">Arquivo</TabsTrigger>
          </TabsList>

          <TabsContent value="text" className="flex flex-col gap-4 pt-2">
            <div className="flex flex-col gap-2">
              <Label htmlFor="source-name">Nome</Label>
              <Input
                id="source-name"
                value={textName}
                onChange={(e) => onTextNameChange(e.target.value)}
                placeholder="Ex: Política de trocas"
              />
            </div>
            <div className="flex flex-col gap-2">
              <Label htmlFor="source-content">Conteúdo</Label>
              <Textarea
                id="source-content"
                value={textContent}
                onChange={(e) => onTextContentChange(e.target.value)}
                placeholder="Cole aqui o texto que a IA deve aprender"
                rows={8}
              />
            </div>
          </TabsContent>

          <TabsContent value="url" className="flex flex-col gap-4 pt-2">
            <div className="flex flex-col gap-2">
              <Label htmlFor="source-url">Endereço (URL)</Label>
              <Input
                id="source-url"
                type="url"
                value={urlValue}
                onChange={(e) => onUrlValueChange(e.target.value)}
                placeholder="https://exemplo.com/pagina"
              />
              <p className="text-xs text-muted-foreground">
                A página será raspada e convertida em texto para a IA aprender.
              </p>
            </div>
            <div className="flex flex-col gap-2">
              <Label htmlFor="source-url-name">Nome (opcional)</Label>
              <Input
                id="source-url-name"
                value={urlName}
                onChange={(e) => onUrlNameChange(e.target.value)}
                placeholder="Ex: Notícias do dia"
              />
            </div>
          </TabsContent>

          <TabsContent value="file" className="flex flex-col gap-4 pt-2">
            <div className="flex flex-col gap-2">
              <Label htmlFor="source-file">Arquivo</Label>
              <Input
                id="source-file"
                type="file"
                accept=".txt,.pdf,.docx,.csv,.xlsx,.md"
                onChange={(e) => onFileChange(e.target.files?.[0] ?? null)}
              />
              <p className="text-xs text-muted-foreground">
                Formatos: txt, pdf, docx, csv, xlsx, md
              </p>
              {file && (
                <p className="text-sm text-foreground truncate">{file.name}</p>
              )}
            </div>
          </TabsContent>
        </Tabs>

        <DialogFooter className="mt-2">
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={submitting}
          >
            Cancelar
          </Button>
          <Button onClick={onSubmit} disabled={!canSubmit || submitting}>
            {submitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Salvar e indexar
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
