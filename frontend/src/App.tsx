import { useState, useEffect } from 'react';
import { 
  ThemeProvider, 
  createTheme, 
  CssBaseline, 
  Box, 
  Container, 
  Typography, 
  Alert,
  CircularProgress,
  Button,
  Tabs,
  Tab,
  Paper,
  AppBar,
  Toolbar,
  Chip
} from '@mui/material';
import { Refresh } from '@mui/icons-material';
import axios from 'axios';
import ClusterView from './components/ClusterView';
import NetworkGraph from './components/NetworkGraph';
import IssuesPanel from './components/IssuesPanel';
import IntelligentInsights from './components/IntelligentInsights';
import WhatIfSimulations from './components/WhatIfSimulations';
import UnifiedDashboard from './components/UnifiedDashboard';

// Define types
interface Pod {
  name: string;
  namespace: string;
  status: string;
  node_name: string;
  pod_ip: string;
  labels: Record<string, string>;
  ready?: boolean;
  restarts?: number;
  age?: string;
}

interface Service {
  name: string;
  namespace: string;
  type: string;
  cluster_ip: string;
  ports: Array<{
    port: number;
    target_port: number;
    protocol: string;
  }>;
  labels?: Record<string, string>;
}

interface Node {
  name: string;
  status: string;
  roles: string[];
  version: string;
  internal_ip: string;
  labels: Record<string, string>;
}

interface NetworkTopology {
  nodes: Array<{
    id: string;
    name: string;
    type: 'pod' | 'service' | 'node' | 'namespace' | 'external';
    namespace?: string;
    health: 'healthy' | 'degraded' | 'failed' | 'unknown';
    pod_ip?: string;
    node_name?: string;
    labels?: Record<string, string>;
    properties?: Record<string, string>;
  }>;
  edges: Array<{
    id: string;
    source: string;
    target: string;
    type: 'connection' | 'service' | 'policy';
    health: 'healthy' | 'degraded' | 'failed' | 'unknown';
    latency_ms?: number;
    packet_loss?: number;
    properties?: Record<string, string>;
  }>;
  timestamp: string;
}

interface NetworkIssue {
  id: string;
  type: 'connectivity' | 'latency' | 'policy' | 'dns' | 'configuration' | 'cidr_overlap' | 'resource_health';
  severity: 'critical' | 'high' | 'medium' | 'low';
  title: string;
  description: string;
  affected_resources: string[];
  suggestions: string[];
  details: Record<string, any>;
  timestamp: string;
}

interface IntelligentInsight {
  id: string;
  category: 'optimization' | 'security' | 'reliability' | 'cost';
  priority: 'high' | 'medium' | 'low';
  title: string;
  description: string;
  impact: string;
  actions: Array<{
    description: string;
    command?: string;
    risk: 'low' | 'medium' | 'high';
    automated: boolean;
  }>;
  metrics: Record<string, any>;
  confidence: number;
  timestamp: string;
}

const theme = createTheme({
  palette: {
    mode: 'dark',
    primary: {
      main: '#1976d2',
    },
    secondary: {
      main: '#dc004e',
    },
    background: {
      default: '#0d1117',
      paper: '#161b22',
    },
  },
});

function App() {
  const [pods, setPods] = useState<Pod[]>([]);
  const [services, setServices] = useState<Service[]>([]);
  const [nodes, setNodes] = useState<Node[]>([]);
  const [topology, setTopology] = useState<NetworkTopology | null>(null);
  const [issues, setIssues] = useState<NetworkIssue[]>([]);
  const [insights, setInsights] = useState<IntelligentInsight[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [currentTab, setCurrentTab] = useState(0);

  const fetchData = async () => {
    try {
      console.log('Starting to fetch data...');
      setLoading(true);
      setError(null);

      // Fetch pods
      console.log('Fetching pods...');
      const podsResponse = await axios.get('/api/pods');
      console.log('Pods response:', podsResponse.data?.length || 0, 'pods');
      setPods(podsResponse.data || []);

      // Fetch services
      console.log('Fetching services...');
      const servicesResponse = await axios.get('/api/services');
      console.log('Services response:', servicesResponse.data?.length || 0, 'services');
      setServices(servicesResponse.data || []);

      // Fetch nodes
      console.log('Fetching nodes...');
      const nodesResponse = await axios.get('/api/nodes');
      console.log('Nodes response:', nodesResponse.data?.length || 0, 'nodes');
      setNodes(nodesResponse.data || []);

      // Fetch topology
      console.log('Fetching topology...');
      const topologyResponse = await axios.get('/api/topology');
      console.log('Topology response:', topologyResponse.data?.nodes?.length || 0, 'nodes,', topologyResponse.data?.edges?.length || 0, 'edges');
      setTopology(topologyResponse.data || null);

      // Fetch issues
      console.log('Fetching issues...');
      const issuesResponse = await axios.get('/api/issues');
      console.log('Issues response:', issuesResponse.data?.length || 0, 'issues');
      setIssues(issuesResponse.data || []);

      // Fetch intelligent insights
      console.log('Fetching insights...');
      try {
        const insightsResponse = await axios.get('/api/insights');
        console.log('Insights response:', insightsResponse.data?.length || 0, 'insights');
        setInsights(insightsResponse.data || []);
      } catch (insightsErr) {
        console.log('Insights API not available yet, using mock data');
        // Mock insights for demo
        setInsights([
          {
            id: 'insight-1',
            category: 'security',
            priority: 'high',
            title: 'Unprotected Default Namespace',
            description: 'The default namespace has no network policies, leaving pods vulnerable to lateral movement attacks.',
            impact: 'Implementing network policies will reduce attack surface by 85%',
            actions: [
              {
                description: 'Create default deny-all network policy',
                command: 'kubectl apply -f default-deny-policy.yaml',
                risk: 'low',
                automated: true
              },
              {
                description: 'Allow only necessary pod-to-pod communication',
                risk: 'medium',
                automated: false
              }
            ],
            metrics: {
              'Exposed Pods': 8,
              'Risk Score': 'High'
            },
            confidence: 0.92,
            timestamp: new Date().toISOString()
          },
          {
            id: 'insight-2',
            category: 'optimization',
            priority: 'medium',
            title: 'Resource Optimization Opportunity',
            description: 'Several pods are over-provisioned with CPU and memory, leading to cluster inefficiency.',
            impact: 'Right-sizing resources could free up 30% cluster capacity',
            actions: [
              {
                description: 'Analyze actual resource usage patterns',
                command: 'kubectl top pods --all-namespaces',
                risk: 'low',
                automated: true
              },
              {
                description: 'Update resource requests and limits',
                risk: 'medium',
                automated: false
              }
            ],
            metrics: {
              'Over-provisioned Pods': 5,
              'Potential Savings': '30%'
            },
            confidence: 0.78,
            timestamp: new Date().toISOString()
          }
        ]);
      }

      console.log('Data fetch completed successfully');
    } catch (err) {
      console.error('Error fetching data:', err);
      setError('Failed to fetch cluster data. Make sure the backend is running.');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
    
    // Refresh data every 30 seconds
    const interval = setInterval(fetchData, 30000);
    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return (
      <ThemeProvider theme={theme}>
        <CssBaseline />
        <Box 
          display="flex" 
          justifyContent="center" 
          alignItems="center" 
          minHeight="100vh"
          flexDirection="column"
        >
          <CircularProgress />
          <Typography variant="body2" sx={{ mt: 2 }}>
            Loading cluster data...
          </Typography>
        </Box>
      </ThemeProvider>
    );
  }

  if (error) {
    return (
      <ThemeProvider theme={theme}>
        <CssBaseline />
        <Box 
          display="flex" 
          justifyContent="center" 
          alignItems="center" 
          minHeight="100vh"
          flexDirection="column"
          p={2}
        >
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
          <Button variant="contained" onClick={fetchData}>
            Retry
          </Button>
        </Box>
      </ThemeProvider>
    );
  }

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <Box sx={{ flexGrow: 1 }}>
        <AppBar position="static">
          <Toolbar>
            <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
              Kubernetes Network Visualizer
            </Typography>
            <Chip 
              label={`${pods.length} Pods`} 
              color="primary" 
              variant="outlined" 
              sx={{ mr: 1 }} 
            />
            <Chip 
              label={`${services.length} Services`} 
              color="secondary" 
              variant="outlined" 
            />
          </Toolbar>
        </AppBar>

        <Container maxWidth={false} sx={{ mt: 2, height: 'calc(100vh - 100px)' }}>
          {error && (
            <Alert severity="error" sx={{ mb: 2 }} action={
              <Button color="inherit" size="small" onClick={fetchData} startIcon={<Refresh />}>
                Retry
              </Button>
            }>
              {error}
            </Alert>
          )}
          
          <Paper sx={{ width: '100%', mb: 2 }}>
            <Tabs 
              value={currentTab} 
              onChange={(_, newValue) => setCurrentTab(newValue)}
              indicatorColor="primary"
              textColor="primary"
              variant="scrollable"
              scrollButtons="auto"
            >
              <Tab label="📊 Cluster Overview" />
              <Tab label="🔍 Network Topology" />
              <Tab label="⚠️ Issues & Security" />
              <Tab label="🤖 AI Insights" />
              <Tab label="🎯 What-If Analysis" />
              <Tab label="🔗 Unified Observability" />
            </Tabs>
          </Paper>

          {currentTab === 0 && (
            <ClusterView
              pods={pods}
              services={services}
              nodes={nodes}
              onRefresh={fetchData}
            />
          )}

          {currentTab === 1 && (
            <NetworkGraph
              topology={topology}
              onNodeClick={(node) => console.log('Node clicked:', node)}
              onEdgeClick={(edge) => console.log('Edge clicked:', edge)}
            />
          )}

          {currentTab === 2 && (
            <IssuesPanel issues={issues} />
          )}

          {currentTab === 3 && (
            <IntelligentInsights insights={insights} />
          )}

          {currentTab === 4 && (
            <WhatIfSimulations 
              onRunSimulation={async (request) => {
                try {
                  const response = await axios.post('/api/simulations', request);
                  return response.data;
                } catch (err) {
                  console.log('Simulation API not available, using mock result');
                  // Return mock result for demo
                  return {
                    id: `sim-${Date.now()}`,
                    request,
                    connectivity_impact: [
                      {
                        area: 'Pod-to-Pod Communication',
                        description: 'May affect communication between pods in different namespaces',
                        risk: 'medium',
                        likelihood: 0.7
                      }
                    ],
                    security_impact: [
                      {
                        area: 'Attack Surface',
                        description: 'Reduces attack surface by blocking unnecessary traffic',
                        risk: 'low',
                        likelihood: 0.9
                      }
                    ],
                    performance_impact: [
                      {
                        area: 'Response Time',
                        description: 'Minor impact on applications requiring external dependencies',
                        risk: 'low',
                        likelihood: 0.4
                      }
                    ],
                    overall_risk: 'medium',
                    confidence: 0.85,
                    timestamp: new Date().toISOString()
                  };
                }
              }}
            />
          )}

          {currentTab === 5 && (
            <UnifiedDashboard />
          )}
        </Container>
      </Box>
    </ThemeProvider>
  );
}

export default App;
