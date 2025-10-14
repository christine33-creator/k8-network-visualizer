import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  TextField,
  Button,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Grid,
  Alert,
  CircularProgress,
  Chip,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  Divider,
  Card,
  CardContent
} from '@mui/material';
import {
  ExpandMore as ExpandMoreIcon,
  PlayArrow as PlayIcon,
  Security as SecurityIcon,
  Speed as SpeedIcon,
  Warning as WarningIcon,
  CheckCircle as CheckCircleIcon,
  Error as ErrorIcon,
  Info as InfoIcon,
  NetworkCheck as NetworkIcon
} from '@mui/icons-material';

interface SimulationPanelProps {
  topology: any;
  onSimulationComplete: (result: any) => void;
}

interface SimulationRequest {
  type: 'network_policy' | 'node_failure' | 'pod_scaling' | 'traffic_surge';
  parameters: Record<string, any>;
}

interface SimulationResult {
  id: string;
  request: SimulationRequest;
  connectivity_impact: Array<{
    area: string;
    description: string;
    risk: 'low' | 'medium' | 'high';
    likelihood: number;
  }>;
  security_impact: Array<{
    area: string;
    description: string;
    risk: 'low' | 'medium' | 'high';
    likelihood: number;
  }>;
  performance_impact: Array<{
    area: string;
    description: string;
    risk: 'low' | 'medium' | 'high';
    likelihood: number;
  }>;
  recommendations: string[];
  overall_risk: 'low' | 'medium' | 'high';
  confidence: number;
  timestamp: string;
}

const SimulationPanel: React.FC<SimulationPanelProps> = ({ topology, onSimulationComplete }) => {
  const [simulationType, setSimulationType] = useState<string>('network_policy');
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<SimulationResult | null>(null);
  const [parameters, setParameters] = useState<Record<string, any>>({
    namespace: 'default',
    policy_type: 'deny_all',
    source: '',
    target: '',
    node_name: '',
    pod_count: 3,
    traffic_multiplier: 2
  });

  const runSimulation = async () => {
    setLoading(true);
    try {
      // Simulate API call with mock data
      await new Promise(resolve => setTimeout(resolve, 2000));
      
      const mockResult: SimulationResult = {
        id: `sim-${Date.now()}`,
        request: {
          type: simulationType as any,
          parameters
        },
        connectivity_impact: [
          {
            area: simulationType === 'network_policy' ? 'Pod-to-Pod Communication' : 'Service Availability',
            description: simulationType === 'network_policy' 
              ? 'May block communication between pods in different namespaces'
              : 'Services may experience temporary disruptions',
            risk: simulationType === 'node_failure' ? 'high' : 'medium',
            likelihood: simulationType === 'node_failure' ? 0.9 : 0.6
          },
          {
            area: 'External Traffic',
            description: 'Ingress traffic may be affected depending on controller placement',
            risk: 'low',
            likelihood: 0.3
          }
        ],
        security_impact: [
          {
            area: 'Attack Surface',
            description: simulationType === 'network_policy'
              ? 'Reduces attack surface by implementing zero-trust networking'
              : 'May expose additional attack vectors during failover',
            risk: simulationType === 'network_policy' ? 'low' : 'medium',
            likelihood: 0.8
          }
        ],
        performance_impact: [
          {
            area: 'Response Time',
            description: simulationType === 'traffic_surge'
              ? 'Services may experience increased latency under load'
              : 'Minimal impact on response times',
            risk: simulationType === 'traffic_surge' ? 'high' : 'low',
            likelihood: simulationType === 'traffic_surge' ? 0.8 : 0.2
          },
          {
            area: 'Throughput',
            description: 'Network throughput may be affected by policy enforcement',
            risk: 'low',
            likelihood: 0.4
          }
        ],
        recommendations: [
          'Test policy changes in a staging environment first',
          'Monitor network metrics closely after implementation',
          'Have a rollback plan ready',
          'Consider implementing changes during maintenance window'
        ],
        overall_risk: simulationType === 'node_failure' ? 'high' : 'medium',
        confidence: 0.85,
        timestamp: new Date().toISOString()
      };

      setResult(mockResult);
      onSimulationComplete(mockResult);
    } catch (error) {
      console.error('Simulation failed:', error);
    } finally {
      setLoading(false);
    }
  };

  const getRiskColor = (risk: string) => {
    switch (risk) {
      case 'high': return 'error';
      case 'medium': return 'warning';
      case 'low': return 'success';
      default: return 'default';
    }
  };

  const getRiskIcon = (risk: string) => {
    switch (risk) {
      case 'high': return <ErrorIcon />;
      case 'medium': return <WarningIcon />;
      case 'low': return <CheckCircleIcon />;
      default: return <InfoIcon />;
    }
  };

  return (
    <Box>
      <Grid container spacing={3}>
        {/* Configuration Panel */}
        <Grid item xs={12} md={4}>
          <Paper sx={{ p: 3, height: '100%' }}>
            <Typography variant="h6" gutterBottom>
              Simulation Configuration
            </Typography>
            
            <FormControl fullWidth sx={{ mt: 2, mb: 2 }}>
              <InputLabel>Simulation Type</InputLabel>
              <Select
                value={simulationType}
                onChange={(e) => setSimulationType(e.target.value)}
                label="Simulation Type"
              >
                <MenuItem value="network_policy">Network Policy Change</MenuItem>
                <MenuItem value="node_failure">Node Failure</MenuItem>
                <MenuItem value="pod_scaling">Pod Scaling</MenuItem>
                <MenuItem value="traffic_surge">Traffic Surge</MenuItem>
              </Select>
            </FormControl>

            {simulationType === 'network_policy' && (
              <>
                <TextField
                  fullWidth
                  label="Namespace"
                  value={parameters.namespace}
                  onChange={(e) => setParameters({ ...parameters, namespace: e.target.value })}
                  sx={{ mb: 2 }}
                />
                <FormControl fullWidth sx={{ mb: 2 }}>
                  <InputLabel>Policy Type</InputLabel>
                  <Select
                    value={parameters.policy_type}
                    onChange={(e) => setParameters({ ...parameters, policy_type: e.target.value })}
                    label="Policy Type"
                  >
                    <MenuItem value="deny_all">Deny All</MenuItem>
                    <MenuItem value="allow_specific">Allow Specific</MenuItem>
                    <MenuItem value="block_egress">Block Egress</MenuItem>
                  </Select>
                </FormControl>
              </>
            )}

            {simulationType === 'node_failure' && (
              <TextField
                fullWidth
                label="Node Name"
                value={parameters.node_name}
                onChange={(e) => setParameters({ ...parameters, node_name: e.target.value })}
                placeholder="e.g., worker-node-1"
                sx={{ mb: 2 }}
              />
            )}

            {simulationType === 'pod_scaling' && (
              <TextField
                fullWidth
                type="number"
                label="Target Pod Count"
                value={parameters.pod_count}
                onChange={(e) => setParameters({ ...parameters, pod_count: parseInt(e.target.value) })}
                sx={{ mb: 2 }}
              />
            )}

            {simulationType === 'traffic_surge' && (
              <TextField
                fullWidth
                type="number"
                label="Traffic Multiplier"
                value={parameters.traffic_multiplier}
                onChange={(e) => setParameters({ ...parameters, traffic_multiplier: parseFloat(e.target.value) })}
                sx={{ mb: 2 }}
              />
            )}

            <Button
              fullWidth
              variant="contained"
              color="primary"
              startIcon={loading ? <CircularProgress size={20} /> : <PlayIcon />}
              onClick={runSimulation}
              disabled={loading}
            >
              {loading ? 'Running Simulation...' : 'Run Simulation'}
            </Button>

            <Alert severity="info" sx={{ mt: 2 }}>
              Simulations run in a sandbox environment and don't affect your actual cluster.
            </Alert>
          </Paper>
        </Grid>

        {/* Results Panel */}
        <Grid item xs={12} md={8}>
          {!result ? (
            <Paper sx={{ p: 3, height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              <Box textAlign="center">
                <NetworkIcon sx={{ fontSize: 64, color: 'text.secondary', mb: 2 }} />
                <Typography variant="h6" color="text.secondary">
                  Configure and run a simulation to see the impact analysis
                </Typography>
              </Box>
            </Paper>
          ) : (
            <Paper sx={{ p: 3 }}>
              <Box display="flex" alignItems="center" justifyContent="space-between" mb={2}>
                <Typography variant="h6">
                  Simulation Results
                </Typography>
                <Box>
                  <Chip
                    label={`Overall Risk: ${result.overall_risk.toUpperCase()}`}
                    color={getRiskColor(result.overall_risk)}
                    icon={getRiskIcon(result.overall_risk)}
                    sx={{ mr: 1 }}
                  />
                  <Chip
                    label={`Confidence: ${(result.confidence * 100).toFixed(0)}%`}
                    variant="outlined"
                  />
                </Box>
              </Box>

              <Grid container spacing={2}>
                {/* Connectivity Impact */}
                <Grid item xs={12}>
                  <Accordion defaultExpanded>
                    <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                      <NetworkIcon sx={{ mr: 1 }} />
                      <Typography>Connectivity Impact</Typography>
                    </AccordionSummary>
                    <AccordionDetails>
                      <List>
                        {result.connectivity_impact.map((impact, index) => (
                          <ListItem key={index}>
                            <ListItemIcon>
                              {getRiskIcon(impact.risk)}
                            </ListItemIcon>
                            <ListItemText
                              primary={impact.area}
                              secondary={
                                <>
                                  {impact.description}
                                  <Box mt={1}>
                                    <Chip
                                      size="small"
                                      label={`Risk: ${impact.risk}`}
                                      color={getRiskColor(impact.risk)}
                                      sx={{ mr: 1 }}
                                    />
                                    <Chip
                                      size="small"
                                      label={`Likelihood: ${(impact.likelihood * 100).toFixed(0)}%`}
                                      variant="outlined"
                                    />
                                  </Box>
                                </>
                              }
                            />
                          </ListItem>
                        ))}
                      </List>
                    </AccordionDetails>
                  </Accordion>
                </Grid>

                {/* Security Impact */}
                <Grid item xs={12}>
                  <Accordion>
                    <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                      <SecurityIcon sx={{ mr: 1 }} />
                      <Typography>Security Impact</Typography>
                    </AccordionSummary>
                    <AccordionDetails>
                      <List>
                        {result.security_impact.map((impact, index) => (
                          <ListItem key={index}>
                            <ListItemIcon>
                              {getRiskIcon(impact.risk)}
                            </ListItemIcon>
                            <ListItemText
                              primary={impact.area}
                              secondary={
                                <>
                                  {impact.description}
                                  <Box mt={1}>
                                    <Chip
                                      size="small"
                                      label={`Risk: ${impact.risk}`}
                                      color={getRiskColor(impact.risk)}
                                      sx={{ mr: 1 }}
                                    />
                                    <Chip
                                      size="small"
                                      label={`Likelihood: ${(impact.likelihood * 100).toFixed(0)}%`}
                                      variant="outlined"
                                    />
                                  </Box>
                                </>
                              }
                            />
                          </ListItem>
                        ))}
                      </List>
                    </AccordionDetails>
                  </Accordion>
                </Grid>

                {/* Performance Impact */}
                <Grid item xs={12}>
                  <Accordion>
                    <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                      <SpeedIcon sx={{ mr: 1 }} />
                      <Typography>Performance Impact</Typography>
                    </AccordionSummary>
                    <AccordionDetails>
                      <List>
                        {result.performance_impact.map((impact, index) => (
                          <ListItem key={index}>
                            <ListItemIcon>
                              {getRiskIcon(impact.risk)}
                            </ListItemIcon>
                            <ListItemText
                              primary={impact.area}
                              secondary={
                                <>
                                  {impact.description}
                                  <Box mt={1}>
                                    <Chip
                                      size="small"
                                      label={`Risk: ${impact.risk}`}
                                      color={getRiskColor(impact.risk)}
                                      sx={{ mr: 1 }}
                                    />
                                    <Chip
                                      size="small"
                                      label={`Likelihood: ${(impact.likelihood * 100).toFixed(0)}%`}
                                      variant="outlined"
                                    />
                                  </Box>
                                </>
                              }
                            />
                          </ListItem>
                        ))}
                      </List>
                    </AccordionDetails>
                  </Accordion>
                </Grid>

                {/* Recommendations */}
                <Grid item xs={12}>
                  <Card>
                    <CardContent>
                      <Typography variant="h6" gutterBottom>
                        Recommendations
                      </Typography>
                      <List dense>
                        {result.recommendations.map((rec, index) => (
                          <ListItem key={index}>
                            <ListItemIcon>
                              <CheckCircleIcon color="primary" />
                            </ListItemIcon>
                            <ListItemText primary={rec} />
                          </ListItem>
                        ))}
                      </List>
                    </CardContent>
                  </Card>
                </Grid>
              </Grid>
            </Paper>
          )}
        </Grid>
      </Grid>
    </Box>
  );
};

export default SimulationPanel;