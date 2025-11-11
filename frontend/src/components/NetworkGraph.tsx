import React, { useEffect, useRef, useState, useCallback } from "react";
import cytoscape from "cytoscape";
import type { Core } from "cytoscape";
// @ts-ignore - cytoscape-fcose doesn't have type definitions
import fcose from "cytoscape-fcose";
import { 
  Box, 
  Paper, 
  Typography, 
  Chip, 
  Stack,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  TextField,
  Button,
  Checkbox,
  FormControlLabel,
  SelectChangeEvent
} from "@mui/material";
import FilterListIcon from '@mui/icons-material/FilterList';
import ClearIcon from '@mui/icons-material/Clear';

// Register the fcose layout
// @ts-ignore - Type compatibility issue with cytoscape versions
cytoscape.use(fcose);

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
  
  // Filter states
  const [filters, setFilters] = useState({
    nodeType: 'all',
    healthStatus: 'all',
    namespace: 'all',
    searchQuery: '',
    showOnlyIssues: false,
    minLatency: 0,
    showPacketLoss: false,
  });
  
  const [namespaces, setNamespaces] = useState<string[]>([]);
  const [showFilters, setShowFilters] = useState(false);

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
    const shapes: Record<GraphNode["type"], any> = {
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

  // Extract namespaces from topology
  useEffect(() => {
    if (topology) {
      const ns = new Set<string>();
      topology.nodes.forEach(node => {
        if (node.namespace) ns.add(node.namespace);
      });
      setNamespaces(Array.from(ns).sort());
    }
  }, [topology]);

  // Apply filters to graph
  const applyFilters = useCallback(() => {
    if (!cyRef.current) return;
    
    const cy = cyRef.current;
    
    // Reset visibility
    cy.elements().style('display', 'element');
    
    // Apply node filters
    cy.nodes().forEach((node: any) => {
      let shouldHide = false;
      
      // Filter by node type
      if (filters.nodeType !== 'all' && node.data('nodeType') !== filters.nodeType) {
        shouldHide = true;
      }
      
      // Filter by health status
      if (filters.healthStatus !== 'all' && node.data('health') !== filters.healthStatus) {
        shouldHide = true;
      }
      
      // Filter by namespace
      if (filters.namespace !== 'all' && node.data('namespace') !== filters.namespace) {
        shouldHide = true;
      }
      
      // Filter by search query
      if (filters.searchQuery && !node.data('label').toLowerCase().includes(filters.searchQuery.toLowerCase())) {
        shouldHide = true;
      }
      
      // Show only issues
      if (filters.showOnlyIssues && node.data('health') === 'healthy') {
        shouldHide = true;
      }
      
      if (shouldHide) {
        node.style('display', 'none');
      }
    });
    
    // Apply edge filters
    cy.edges().forEach((edge: any) => {
      let shouldHide = false;
      
      // Hide edges connected to hidden nodes
      if (edge.source().style('display') === 'none' || edge.target().style('display') === 'none') {
        shouldHide = true;
      }
      
      // Filter by minimum latency
      if (filters.minLatency > 0 && (!edge.data('latency') || edge.data('latency') < filters.minLatency)) {
        shouldHide = true;
      }
      
      // Show only packet loss
      if (filters.showPacketLoss && !edge.data('packet_loss')) {
        shouldHide = true;
      }
      
      if (shouldHide) {
        edge.style('display', 'none');
      }
    });
  }, [filters]);

  // Apply filters when they change
  useEffect(() => {
    applyFilters();
  }, [filters, applyFilters]);

  const handleFilterChange = (key: string, value: any) => {
    setFilters(prev => ({ ...prev, [key]: value }));
  };

  const resetFilters = () => {
    setFilters({
      nodeType: 'all',
      healthStatus: 'all',
      namespace: 'all',
      searchQuery: '',
      showOnlyIssues: false,
      minLatency: 0,
      showPacketLoss: false,
    });
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
            "background-color": (ele: any) => getNodeColor(ele.data()),
            label: "data(label)",
            "text-valign": "center",
            "text-halign": "center",
            shape: (ele: any) => getNodeShape(ele.data("nodeType")),
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
            "line-color": (ele: any) => getEdgeColor(ele.data()),
            "target-arrow-color": (ele: any) => getEdgeColor(ele.data()),
            "target-arrow-shape": "triangle",
            "curve-style": "bezier",
            "line-style": (ele: any) => getEdgeStyle(ele.data("edgeType")),
            label: (ele: any) =>
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

        {/* Legend */}
        <Paper
          sx={{
            position: "absolute",
            top: 16,
            right: 16,
            p: 2,
            minWidth: 200,
            backgroundColor: "rgba(255, 255, 255, 0.95)",
          }}
          elevation={2}
        >
          <Typography variant="subtitle2" gutterBottom sx={{ fontWeight: "bold" }}>
            Legend
          </Typography>
          
          {/* Node Shapes */}
          <Typography variant="caption" color="text.secondary" sx={{ display: "block", mt: 1, mb: 0.5 }}>
            Node Types:
          </Typography>
          <Stack spacing={0.5}>
            <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
              <Box
                sx={{
                  width: 30,
                  height: 20,
                  borderRadius: "50%",
                  border: "2px solid #9e9e9e",
                  backgroundColor: "#f5f5f5",
                }}
              />
              <Typography variant="caption">Pod (Circle)</Typography>
            </Box>
            <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
              <Box
                sx={{
                  width: 0,
                  height: 0,
                  borderLeft: "10px solid transparent",
                  borderRight: "10px solid transparent",
                  borderBottom: "20px solid #9e9e9e",
                  transform: "rotate(45deg)",
                }}
              />
              <Typography variant="caption" sx={{ ml: 1 }}>Service (Diamond)</Typography>
            </Box>
            <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
              <Box
                sx={{
                  width: 30,
                  height: 20,
                  border: "2px solid #9e9e9e",
                  backgroundColor: "#f5f5f5",
                }}
              />
              <Typography variant="caption">Node (Rectangle)</Typography>
            </Box>
            <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
              <Box
                sx={{
                  width: 0,
                  height: 0,
                  borderLeft: "15px solid transparent",
                  borderRight: "15px solid transparent",
                  borderBottom: "20px solid #9e9e9e",
                }}
              />
              <Typography variant="caption">Namespace (Triangle)</Typography>
            </Box>
          </Stack>

          {/* Health Status */}
          <Typography variant="caption" color="text.secondary" sx={{ display: "block", mt: 2, mb: 0.5 }}>
            Health Status:
          </Typography>
          <Stack spacing={0.5}>
            <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
              <Box
                sx={{
                  width: 16,
                  height: 16,
                  borderRadius: "50%",
                  backgroundColor: "#4caf50",
                }}
              />
              <Typography variant="caption">Healthy</Typography>
            </Box>
            <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
              <Box
                sx={{
                  width: 16,
                  height: 16,
                  borderRadius: "50%",
                  backgroundColor: "#ff9800",
                }}
              />
              <Typography variant="caption">Degraded</Typography>
            </Box>
            <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
              <Box
                sx={{
                  width: 16,
                  height: 16,
                  borderRadius: "50%",
                  backgroundColor: "#f44336",
                }}
              />
              <Typography variant="caption">Failed</Typography>
            </Box>
            <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
              <Box
                sx={{
                  width: 16,
                  height: 16,
                  borderRadius: "50%",
                  backgroundColor: "#9e9e9e",
                }}
              />
              <Typography variant="caption">Unknown</Typography>
            </Box>
          </Stack>
        </Paper>

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
