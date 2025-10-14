import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Button,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Card,
  CardContent,
  CardActions,
  List,
  ListItem,
  ListItemText,
  Chip,
  Stack,
  LinearProgress,
  Divider,
  IconButton,
  Tooltip
} from '@mui/material';
import {
  Security,
  Speed,
  NetworkCheck,
  Delete,
  PlayArrow,
  Stop
} from '@mui/icons-material';

interface ImpactPrediction {
  area: string;
  description: string;
  risk: 'low' | 'medium' | 'high';
  likelihood: number;
}

interface SimulationRequest {
  type: 'policy' | 'resource' | 'topology';
  name: string;
  change: string;
  namespace?: string;
  parameters?: Record<string, any>;
}

interface SimulationResult {
  id: string;
  request: SimulationRequest;
  connectivity_impact: ImpactPrediction[];
  security_impact: ImpactPrediction[];
  performance_impact: ImpactPrediction[];
  overall_risk: 'low' | 'medium' | 'high';
  confidence: number;
  timestamp: string;
}

interface WhatIfSimulationsProps {
  onRunSimulation?: (request: SimulationRequest) => Promise<SimulationResult>;
}

const WhatIfSimulations: React.FC<WhatIfSimulationsProps> = ({ onRunSimulation }) => {
  const [simulations, setSimulations] = useState<SimulationResult[]>([]);
  const [isRunning, setIsRunning] = useState(false);
  const [currentSimulation, setCurrentSimulation] = useState<SimulationRequest>({
    type: 'policy',
    name: '',
    change: '',
    namespace: 'default',
    parameters: {}
  });

  const simulationTemplates = {
    policy: [
      {
        name: 'Block External Traffic',
        change: 'deny-external-egress',
        description: 'Block all egress traffic to external networks'
      },
      {
        name: 'Allow Only DNS',
        change: 'dns-only-policy',
        description: 'Allow only DNS traffic from pods'
      },
      {
        name: 'Service Mesh Security',
        change: 'istio-mtls-strict',
        description: 'Enable strict mTLS for all service-to-service communication'
      }
    ],
    resource: [
      {
        name: 'Scale Deployment',
        change: 'scale-replicas',
        description: 'Change deployment replica count'
      },
      {
        name: 'Update Resource Limits',
        change: 'update-limits',
        description: 'Modify CPU/memory limits and requests'
      },
      {
        name: 'Add Node',
        change: 'add-node',
        description: 'Add a new worker node to the cluster'
      }
    ],
    topology: [
      {
        name: 'Service Type Change',
        change: 'loadbalancer-to-nodeport',
        description: 'Change service from LoadBalancer to NodePort'
      },
      {
        name: 'Move to Different Namespace',
        change: 'namespace-migration',
        description: 'Move workload to a different namespace'
      },
      {
        name: 'Enable Ingress',
        change: 'add-ingress',
        description: 'Add ingress controller for external access'
      }
    ]
  };

  const handleRunSimulation = async () => {
    if (!currentSimulation.name || !currentSimulation.change) {
      return;
    }

    setIsRunning(true);
    try {
      // Mock simulation for demo - in real app this would call the backend
      const mockResult: SimulationResult = {
        id: `sim-${Date.now()}`,
        request: { ...currentSimulation },
        connectivity_impact: [
          {
            area: 'Pod-to-Pod Communication',
            description: 'May affect communication between pods in different namespaces',
            risk: 'medium',
            likelihood: 0.7
          },
          {
            area: 'External Access',
            description: 'Will block access to external APIs and services',
            risk: 'high',
            likelihood: 0.9
          }
        ],
        security_impact: [
          {
            area: 'Attack Surface',
            description: 'Significantly reduces attack surface by blocking external traffic',
            risk: 'low',
            likelihood: 0.95
          }
        ],
        performance_impact: [
          {
            area: 'Response Time',
            description: 'Minor impact on applications requiring external dependencies',
            risk: 'medium',
            likelihood: 0.6
          }
        ],
        overall_risk: 'medium',
        confidence: 0.85,
        timestamp: new Date().toISOString()
      };

      if (onRunSimulation) {
        const result = await onRunSimulation(currentSimulation);
        setSimulations([result, ...simulations]);
      } else {
        setSimulations([mockResult, ...simulations]);
      }

      // Reset form
      setCurrentSimulation({
        type: 'policy',
        name: '',
        change: '',
        namespace: 'default',
        parameters: {}
      });
    } finally {
      setIsRunning(false);
    }
  };

  const getRiskColor = (risk: string) => {
    switch (risk) {
      case 'high':
        return 'error';
      case 'medium':
        return 'warning';
      case 'low':
        return 'success';
      default:
        return 'default';
    }
  };

  const applyTemplate = (template: any) => {
    setCurrentSimulation({
      ...currentSimulation,
      name: template.name,
      change: template.change
    });
  };

  return (
    <Box>
      <Typography variant="h5" gutterBottom>
        🎯 What-If Analysis & Impact Simulation
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
        Predict the impact of policy, resource, and topology changes before implementing them
      </Typography>

      {/* Simulation Setup */}
      <Paper sx={{ p: 3, mb: 3 }}>
        <Typography variant="h6" gutterBottom>
          Create New Simulation
        </Typography>
        
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
          <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap' }}>
            <FormControl sx={{ minWidth: 200 }}>
              <InputLabel>Simulation Type</InputLabel>
              <Select
                value={currentSimulation.type}
                label="Simulation Type"
                onChange={(e) => setCurrentSimulation({
                  ...currentSimulation,
                  type: e.target.value as any,
                  name: '',
                  change: ''
                })}
              >
                <MenuItem value="policy">Network Policy</MenuItem>
                <MenuItem value="resource">Resource Changes</MenuItem>
                <MenuItem value="topology">Topology Changes</MenuItem>
              </Select>
            </FormControl>
            
            <TextField
              sx={{ minWidth: 250 }}
              label="Simulation Name"
              value={currentSimulation.name}
              onChange={(e) => setCurrentSimulation({
                ...currentSimulation,
                name: e.target.value
              })}
              placeholder="e.g., Block External Traffic"
            />
            
            <TextField
              sx={{ minWidth: 150 }}
              label="Namespace"
              value={currentSimulation.namespace}
              onChange={(e) => setCurrentSimulation({
                ...currentSimulation,
                namespace: e.target.value
              })}
              placeholder="default"
            />
          </Box>
          
          <TextField
            fullWidth
            multiline
            rows={3}
            label="Change Description"
            value={currentSimulation.change}
            onChange={(e) => setCurrentSimulation({
              ...currentSimulation,
              change: e.target.value
            })}
            placeholder="Describe the change you want to simulate..."
          />
        </Box>

        {/* Templates */}
        <Box sx={{ mt: 3 }}>
          <Typography variant="subtitle2" gutterBottom>
            Quick Templates
          </Typography>
          <Stack direction="row" spacing={1} flexWrap="wrap">
            {simulationTemplates[currentSimulation.type].map((template, idx) => (
              <Tooltip key={idx} title={template.description}>
                <Chip
                  label={template.name}
                  onClick={() => applyTemplate(template)}
                  variant="outlined"
                  color="primary"
                  size="small"
                />
              </Tooltip>
            ))}
          </Stack>
        </Box>

        <Box sx={{ mt: 3, display: 'flex', gap: 2 }}>
          <Button
            variant="contained"
            onClick={handleRunSimulation}
            disabled={isRunning || !currentSimulation.name || !currentSimulation.change}
            startIcon={isRunning ? <Stop /> : <PlayArrow />}
          >
            {isRunning ? 'Running Simulation...' : 'Run Simulation'}
          </Button>
          <Button
            variant="outlined"
            onClick={() => setCurrentSimulation({
              type: 'policy',
              name: '',
              change: '',
              namespace: 'default',
              parameters: {}
            })}
          >
            Clear
          </Button>
        </Box>
      </Paper>

      {/* Simulation Results */}
      {simulations.length > 0 && (
        <Box>
          <Typography variant="h6" gutterBottom>
            Simulation Results
          </Typography>
          
          <Stack spacing={2}>
            {simulations.map((result) => (
              <Card key={result.id} variant="outlined">
                <CardContent>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 2 }}>
                    <Box>
                      <Typography variant="h6" gutterBottom>
                        {result.request.name}
                      </Typography>
                      <Stack direction="row" spacing={1}>
                        <Chip 
                          label={result.request.type} 
                          size="small" 
                          variant="outlined"
                        />
                        <Chip 
                          label={`Overall Risk: ${result.overall_risk}`}
                          size="small"
                          color={getRiskColor(result.overall_risk) as any}
                        />
                        <Chip 
                          label={result.request.namespace}
                          size="small"
                          variant="outlined"
                          color="info"
                        />
                      </Stack>
                    </Box>
                    <Box sx={{ textAlign: 'right' }}>
                      <Typography variant="caption" color="text.secondary">
                        Confidence
                      </Typography>
                      <Typography variant="h6" color="primary">
                        {Math.round(result.confidence * 100)}%
                      </Typography>
                      <LinearProgress 
                        variant="determinate" 
                        value={result.confidence * 100} 
                        sx={{ width: 60, mt: 0.5 }}
                      />
                    </Box>
                  </Box>

                  <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                    {result.request.change}
                  </Typography>

                  <Divider sx={{ my: 2 }} />

                  {/* Impact Categories */}
                  <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))', gap: 2 }}>
                    {[
                      { title: 'Connectivity Impact', impacts: result.connectivity_impact, icon: <NetworkCheck /> },
                      { title: 'Security Impact', impacts: result.security_impact, icon: <Security /> },
                      { title: 'Performance Impact', impacts: result.performance_impact, icon: <Speed /> }
                    ].map((category) => (
                      <Box key={category.title} sx={{ border: 1, borderColor: 'divider', borderRadius: 1, p: 2 }}>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                          {category.icon}
                          <Typography variant="subtitle2">
                            {category.title}
                          </Typography>
                        </Box>
                        
                        {category.impacts.length === 0 ? (
                          <Typography variant="body2" color="text.secondary">
                            No predicted impact
                          </Typography>
                        ) : (
                          <List dense>
                            {category.impacts.map((impact, idx) => (
                              <ListItem key={idx} sx={{ px: 0 }}>
                                <ListItemText
                                  primary={impact.area}
                                  secondary={
                                    <Box>
                                      <Typography variant="caption" display="block">
                                        {impact.description}
                                      </Typography>
                                      <Box sx={{ display: 'flex', gap: 1, mt: 0.5 }}>
                                        <Chip 
                                          label={`${impact.risk} risk`}
                                          size="small"
                                          color={getRiskColor(impact.risk) as any}
                                          variant="outlined"
                                        />
                                        <Chip 
                                          label={`${Math.round(impact.likelihood * 100)}% likely`}
                                          size="small"
                                          variant="outlined"
                                        />
                                      </Box>
                                    </Box>
                                  }
                                />
                              </ListItem>
                            ))}
                          </List>
                        )}
                      </Box>
                    ))}
                  </Box>
                </CardContent>
                
                <CardActions>
                  <Button size="small" color="primary">
                    Implement Changes
                  </Button>
                  <Button size="small" color="inherit">
                    Export Report
                  </Button>
                  <Box sx={{ ml: 'auto' }}>
                    <IconButton 
                      size="small" 
                      onClick={() => setSimulations(simulations.filter(s => s.id !== result.id))}
                    >
                      <Delete fontSize="small" />
                    </IconButton>
                  </Box>
                  <Typography variant="caption" color="text.disabled" sx={{ ml: 1 }}>
                    {new Date(result.timestamp).toLocaleString()}
                  </Typography>
                </CardActions>
              </Card>
            ))}
          </Stack>
        </Box>
      )}

      {simulations.length === 0 && (
        <Paper sx={{ p: 3, textAlign: 'center' }}>
          <Typography variant="h6" color="text.secondary" gutterBottom>
            No simulations run yet
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Create and run your first simulation to see predicted impacts of changes.
          </Typography>
        </Paper>
      )}
    </Box>
  );
};

export default WhatIfSimulations;
