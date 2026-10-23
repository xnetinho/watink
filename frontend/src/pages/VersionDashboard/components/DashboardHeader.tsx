import React from "react";
import { Link as RouterLink } from "react-router";
import { UploadCloud, BookOpen } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";

interface DashboardHeaderProps {
  hasUpdateAvailable: boolean;
  availableVersion: string;
  onUpdateClick: () => void;
}

const DashboardHeader: React.FC<DashboardHeaderProps> = ({
  hasUpdateAvailable,
  availableVersion,
  onUpdateClick,
}) => (
  <div className="flex flex-wrap items-center justify-between gap-3">
    <h1 className="text-xl font-semibold">Monitor do Sistema (Business)</h1>
    <div className="flex items-center gap-2">
      <Button variant="outline" size="sm" asChild>
        <RouterLink to="/swagger">
          <BookOpen className="mr-2 h-4 w-4" />
          Swagger
        </RouterLink>
      </Button>
      {hasUpdateAvailable && (
        <Badge variant="destructive">Nova versão v{availableVersion}</Badge>
      )}
      <Button size="sm" onClick={onUpdateClick}>
        <UploadCloud className="mr-2 h-4 w-4" />
        Verificar Atualização
      </Button>
    </div>
  </div>
);

export default DashboardHeader;
