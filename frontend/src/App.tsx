import { useState, useEffect } from 'react';
import { 
  ThemeProvider, 
  createTheme, 
  CssBaseline, 
  AppBar, 
  Toolbar, 
  Typography, 
  Container, 
  Box,
  CircularProgress,
  Alert,
  Chip,
  Button
} from '@mui/material';
import ClusterView from './components/ClusterView';
import axios from 'axios';
import './App.css';

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
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

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
            <Alert severity="error" sx={{ mb: 2 }}>
              {error}
            </Alert>
          )}
          
          <ClusterView
            pods={pods}
            services={services}
            nodes={nodes}
            onRefresh={fetchData}
          />
        </Container>
      </Box>
    </ThemeProvider>
  );
}

export default App;
