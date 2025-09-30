import React, { useEffect, useRef, useState } from "react";
import cytoscape from "cytoscape";
import type { Core } from "cytoscape";
import fcose from "cytoscape-fcose";
import { Box, Paper, Typography, Chip, Stack } from "@mui/material";

// Register the fcose layout
cytoscape.use(fcose);

// Fallback type declaration if @types/cytoscape-fcose is missing
// Remove if you install @types/cytoscape-fcose
declare module "cytoscape-fcose";

interface GraphNode {
  id: string;
  name: string;
  type: "pod" | "service" | "node" | "namespace" | "external";
  namespace?: string;
  health: "healthy" | "degraded" | "failed" | "unknown";
  pod_ip?: string;
  node_name?: string;
  labels?: Record<string, string>;
  properties?: Record<string, string>;
}

interface GraphEdge {
  id: string;
  source: string;
  target: string;
  type: "connection" | "service" | "policy";
  health: "healthy" | "degraded" | "failed" | "unknown";
  latency_ms?: number;
  packet_loss?: number;
  properties?: Record<string, string>;
}

interface NetworkTopology {
  nodes: GraphNode[];
  edges: GraphEdge[];
  timestamp: string;
}

interface NetworkGraphProps {
  topology: NetworkTopology | null;
  onNodeClick?: (node: GraphNode) => void;
  onEdgeClick?: (edge: GraphEdge) => void;
}

const NetworkGraph: React.FC<NetworkGraphProps> = ({
  topology,
  onNodeClick,
  onEdgeClick,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const cyRef = useRef<Core | null>(null);
  const [selectedElement, setSelectedElement] = useState<any>(null);

  const getNodeColor = (node: GraphNode): string => {
    const healthColors: Record<GraphNode["health"], string> = {
      healthy: "#4caf50",
      degraded: "#ff9800",
      failed: "#f44336",
      unknown: "#9e9e9e",
    };
    return healthColors[node.health];
  };

  const getNodeShape = (type: GraphNode["type"]) => {
    const shapes: Record<GraphNode["type"], cytoscape.NodeShape> = {
      pod: "ellipse",
      service: "diamond",
      node: "rectangle",
      namespace: "round-rectangle",
      external: "hexagon",
    };
    return shapes[type];
  };

  const getEdgeColor = (edge: GraphEdge): string => {
    const healthColors: Record<GraphEdge["health"], string> = {
      healthy: "#4caf50",
      degraded: "#ff9800",
      failed: "#f44336",
      unknown: "#cccccc",
    };
    return healthColors[edge.health];
  };

  const getEdgeStyle = (type: GraphEdge["type"]) => {
    const styles: Record<GraphEdge["type"], cytoscape.Css.LineStyle> = {
      connection: "solid",
      service: "dashed",
      policy: "dotted",
    };
    return styles[type];
  };

  useEffect(() => {
    if (!containerRef.current || !topology) return;

    // Convert topology to Cytoscape elements
    const elements = [
      ...topology.nodes.map((node) => ({
        data: {
          id: node.id,
          label: node.name,
          nodeType: node.type,
          health: node.health,
          namespace: node.namespace,
          // spread carefully: avoid overwriting id/health
          ...node.properties,
        },
        classes: `node-${node.type} health-${node.health}`,
      })),
      ...topology.edges.map((edge) => ({
        data: {
          id: edge.id,
          source: edge.source,
          target: edge.target,
          edgeType: edge.type,
          health: edge.health,
          latency: edge.latency_ms,
          // spread carefully: avoid overwriting id/source/target
          ...edge.properties,
        },
        classes: `edge-${edge.type} health-${edge.health}`,
      })),
    ];

    const cy = cytoscape({
      container: containerRef.current,
      elements,
      style: [
        {
          selector: "node",
          style: {
            "background-color": (ele) => getNodeColor(ele.data()),
            label: "data(label)",
            "text-valign": "center",
            "text-halign": "center",
            shape: (ele) => getNodeShape(ele.data("nodeType")),
            width: 40,
            height: 40,
            "font-size": "10px",
            "text-wrap": "wrap",
            "text-max-width": "60px",
            "overlay-padding": "6px",
            "z-index": 10,
            "border-width": 2,
            "border-color": "#fff",
          },
        },
        {
          selector: "edge",
          style: {
            width: 2,
            "line-color": (ele) => getEdgeColor(ele.data()),
            "target-arrow-color": (ele) => getEdgeColor(ele.data()),
            "target-arrow-shape": "triangle",
            "curve-style": "bezier",
            "line-style": (ele) => getEdgeStyle(ele.data("edgeType")),
            label: (ele) =>
              ele.data("latency") ? `${ele.data("latency")}ms` : "",
            "font-size": "8px",
            "text-rotation": "autorotate",
            "text-margin-y": -10,
          },
        },
      ],
      layout: { name: "fcose" } as any,
      minZoom: 0.1,
      maxZoom: 3,
      wheelSensitivity: 0.2,
    });

    cy.on("tap", "node", (evt) => {
      const node = evt.target;
      setSelectedElement(node.data());
      onNodeClick?.(node.data());
    });

    cy.on("tap", "edge", (evt) => {
      const edge = evt.target;
      setSelectedElement(edge.data());
      onEdgeClick?.(edge.data());
    });

    cy.on("tap", (evt) => {
      if (evt.target === cy) setSelectedElement(null);
    });

    cyRef.current = cy;

    return () => {
      cy.destroy();
    };
  }, [topology, onNodeClick, onEdgeClick]);

  const getHealthChipColor = (health: string) => {
    switch (health) {
      case "healthy":
        return "success";
      case "degraded":
        return "warning";
      case "failed":
        return "error";
      default:
        return "default";
    }
  };

  return (
    <Box sx={{ height: "100%", display: "flex", flexDirection: "column" }}>
      {/* Top bar */}
      <Paper sx={{ p: 2, mb: 2 }}>
        <Stack direction="row" spacing={2} alignItems="center">
          <Typography variant="h6">Network Topology</Typography>
          {topology && (
            <>
              <Chip label={`${topology.nodes.length} Nodes`} size="small" />
              <Chip label={`${topology.edges.length} Connections`} size="small" />
              <Typography variant="caption" color="text.secondary">
                Last updated:{" "}
                {new Date(topology.timestamp).toLocaleTimeString()}
              </Typography>
            </>
          )}
        </Stack>
      </Paper>

      {/* Graph */}
      <Paper sx={{ flex: 1, position: "relative", overflow: "hidden" }}>
        <Box ref={containerRef} sx={{ width: "100%", height: "100%" }} />

        {selectedElement && (
          <Paper
            sx={{
              position: "absolute",
              bottom: 16,
              right: 16,
              p: 2,
              maxWidth: 300,
              maxHeight: 400,
              overflow: "auto",
            }}
            elevation={3}
          >
            <Typography variant="subtitle2" gutterBottom>
              {selectedElement.nodeType ? "Node" : "Edge"} Details
            </Typography>
            <Stack spacing={1}>
              <Typography variant="body2">
                <strong>ID:</strong> {selectedElement.id}
              </Typography>
              {selectedElement.label && (
                <Typography variant="body2">
                  <strong>Name:</strong> {selectedElement.label}
                </Typography>
              )}
              <Chip
                label={selectedElement.health}
                color={getHealthChipColor(selectedElement.health) as any}
                size="small"
              />
              {selectedElement.namespace && (
                <Typography variant="body2">
                  <strong>Namespace:</strong> {selectedElement.namespace}
                </Typography>
              )}
              {selectedElement.pod_ip && (
                <Typography variant="body2">
                  <strong>Pod IP:</strong> {selectedElement.pod_ip}
                </Typography>
              )}
              {selectedElement.latency && (
                <Typography variant="body2">
                  <strong>Latency:</strong> {selectedElement.latency}ms
                </Typography>
              )}
              {selectedElement.properties && (
                <>
                  <Typography variant="subtitle2">Properties:</Typography>
                  {Object.entries(selectedElement.properties).map(
                    ([key, value]) => (
                      <Typography key={key} variant="body2">
                        <strong>{key}:</strong> {value as string}
                      </Typography>
                    )
                  )}
                </>
              )}
            </Stack>
          </Paper>
        )}
      </Paper>
    </Box>
  );
};

export default NetworkGraph;
