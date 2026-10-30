import { useCallback } from 'react';
import type { Node, Edge } from '@xyflow/react';
import { toast } from 'react-toastify';

import api from '../../../services/api';
import { stripLegacyDimensions } from '../flowBuilderTypes';

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface UseFlowIOOptions {
    flowId: string | undefined;
    nodes: Node[];
    edges: Edge[];
    setNodes: (nodes: Node[] | ((nds: Node[]) => Node[])) => void;
    setEdges: (edges: Edge[] | ((eds: Edge[]) => Edge[])) => void;
    fileInputRef: React.RefObject<HTMLInputElement>;
}

export function useFlowIO({ flowId, nodes, edges, setNodes, setEdges, fileInputRef }: UseFlowIOOptions) {
    const handleExport = useCallback(() => {
        const flowData = { nodes, edges };
        const jsonString = `data:text/json;charset=utf-8,${encodeURIComponent(JSON.stringify(flowData))}`;
        const link = document.createElement('a');
        link.href = jsonString;
        link.download = `flow_${flowId}.json`;
        link.click();
    }, [nodes, edges, flowId]);

    const handleImport = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        const file = event.target.files?.[0];
        if (!file) return;
        const fileReader = new FileReader();
        fileReader.readAsText(file, 'UTF-8');
        fileReader.onload = (e) => {
            try {
                const parsed = JSON.parse(e.target!.result as string) as { nodes?: Node[]; edges?: Edge[] };
                if (parsed.nodes && parsed.edges) {
                    setNodes(stripLegacyDimensions(parsed.nodes));
                    setEdges(parsed.edges);
                    toast.success(tr("uiText.flow.io.imported"));
                } else {
                    toast.error(tr("uiText.flow.io.invalid"));
                }
            } catch {
                toast.error(tr("uiText.flow.io.readError"));
            }
        };
        // reset so the same file can be re-imported
        if (fileInputRef.current) fileInputRef.current.value = '';
    }, [setNodes, setEdges, fileInputRef]);

    const handleSimulate = useCallback(async (id: string | undefined, message: string) => {
        try {
            const { data } = await api.post(`/flows/${id}/simulate`, { message });
            return data;
        } catch (err) {
            const status = (err as { response?: { status?: number } })?.response?.status;
            if (status === 501) {
                const friendly = tr("uiText.flow.io.unavailable");
                toast.info(friendly, { toastId: 'flow-simulate-501' });
                throw new Error(friendly);
            }
            throw err;
        }
    }, []);

    const handleAIResponse = useCallback((newNodes: Node[], newEdges: Edge[]) => {
        if (newNodes && newEdges) {
            setNodes(newNodes);
            setEdges(newEdges);
        }
    }, [setNodes, setEdges]);

    return {
        handleExport,
        handleImport,
        handleSimulate,
        handleAIResponse,
    };
}
