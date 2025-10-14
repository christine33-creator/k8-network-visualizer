import axios from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080/api';

// Configure axios defaults
axios.defaults.baseURL = API_BASE_URL;
axios.defaults.headers.common['Content-Type'] = 'application/json';

// Add request interceptor for debugging
axios.interceptors.request.use(
  (config) => {
    console.log('API Request:', config.method?.toUpperCase(), config.url);
    return config;
  },
  (error) => {
    console.error('Request Error:', error);
    return Promise.reject(error);
  }
);

// Add response interceptor for error handling
axios.interceptors.response.use(
  (response) => {
    console.log('API Response:', response.status, response.config.url);
    return response;
  },
  (error) => {
    console.error('Response Error:', error.response?.status, error.message);
    return Promise.reject(error);
  }
);

export interface NetworkTopology {
  nodes: Array<{
    id: string;
    name: string;
    type: 'pod' | 'service' | 'node' | 'namespace';
    namespace?: string;
    health: 'healthy' | 'degraded' | 'failed' | 'unknown';
    properties?: Record<string, any>;
  }>;
  edges: Array<{
    id: string;
    source: string;
    target: string;
    type: 'connection' | 'service' | 'policy';
    health: 'healthy' | 'degraded' | 'failed' | 'unknown';
    latency_ms?: number;
    packet_loss?: number;
  }>;
  timestamp: string;
}

export interface NetworkIssue {
  id: string;
  type: string;
  severity: 'critical' | 'high' | 'medium' | 'low';
  title: string;
  description: string;
  affected_resources: string[];
  suggestions: string[];
  timestamp: string;
}

export interface SimulationRequest {
  type: 'network_policy' | 'node_failure' | 'pod_scaling' | 'traffic_surge';
  parameters: Record<string, any>;
}

export interface SimulationResult {
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

class ApiClient {
  async getTopology(): Promise<NetworkTopology> {
    const response = await axios.get('/topology');
    return response.data;
  }

  async getIssues(): Promise<NetworkIssue[]> {
    const response = await axios.get('/issues');
    return response.data;
  }

  async getPods(): Promise<any[]> {
    const response = await axios.get('/pods');
    return response.data;
  }

  async getServices(): Promise<any[]> {
    const response = await axios.get('/services');
    return response.data;
  }

  async getNodes(): Promise<any[]> {
    const response = await axios.get('/nodes');
    return response.data;
  }

  async getInsights(): Promise<any[]> {
    const response = await axios.get('/insights');
    return response.data;
  }

  async runSimulation(request: SimulationRequest): Promise<SimulationResult> {
    const response = await axios.post('/simulations', request);
    return response.data;
  }

  async getProbeResults(namespace?: string): Promise<any[]> {
    const params = namespace ? { namespace } : {};
    const response = await axios.get('/probes', { params });
    return response.data;
  }

  async triggerProbe(source: string, target: string): Promise<any> {
    const response = await axios.post('/probes', { source, target });
    return response.data;
  }

  async getNetworkPolicies(namespace?: string): Promise<any[]> {
    const params = namespace ? { namespace } : {};
    const response = await axios.get('/network-policies', { params });
    return response.data;
  }

  async validateNetworkPolicy(policy: any): Promise<any> {
    const response = await axios.post('/network-policies/validate', policy);
    return response.data;
  }

  async getMetrics(resource: string, name: string): Promise<any> {
    const response = await axios.get(`/metrics/${resource}/${name}`);
    return response.data;
  }

  async exportConfiguration(format: 'json' | 'yaml' = 'json'): Promise<any> {
    const response = await axios.get(`/export?format=${format}`);
    return response.data;
  }

  async getHealthStatus(): Promise<any> {
    const response = await axios.get('/health');
    return response.data;
  }
}

export const apiClient = new ApiClient();

// Export mock functions for development
export const mockData = {
  getTopology: (): NetworkTopology => ({
    nodes: [
      {
        id: 'node-1',
        name: 'frontend-pod',
        type: 'pod',
        namespace: 'default',
        health: 'healthy',
        properties: { 
          ip: '10.244.0.5',
          node: 'worker-1'
        }
      },
      {
        id: 'node-2',
        name: 'backend-pod',
        type: 'pod',
        namespace: 'default',
        health: 'healthy',
        properties: {
          ip: '10.244.0.6',
          node: 'worker-1'
        }
      },
      {
        id: 'node-3',
        name: 'database-pod',
        type: 'pod',
        namespace: 'default',
        health: 'degraded',
        properties: {
          ip: '10.244.1.5',
          node: 'worker-2'
        }
      },
      {
        id: 'svc-1',
        name: 'backend-service',
        type: 'service',
        namespace: 'default',
        health: 'healthy',
        properties: {
          clusterIP: '10.96.0.10',
          port: 8080
        }
      }
    ],
    edges: [
      {
        id: 'edge-1',
        source: 'node-1',
        target: 'svc-1',
        type: 'connection',
        health: 'healthy',
        latency_ms: 5,
        packet_loss: 0
      },
      {
        id: 'edge-2',
        source: 'svc-1',
        target: 'node-2',
        type: 'service',
        health: 'healthy',
        latency_ms: 2,
        packet_loss: 0
      },
      {
        id: 'edge-3',
        source: 'node-2',
        target: 'node-3',
        type: 'connection',
        health: 'degraded',
        latency_ms: 45,
        packet_loss: 0.02
      }
    ],
    timestamp: new Date().toISOString()
  }),

  getIssues: (): NetworkIssue[] => [
    {
      id: 'issue-1',
      type: 'connectivity',
      severity: 'high',
      title: 'High Latency Detected',
      description: 'Connection between backend-pod and database-pod showing elevated latency (45ms)',
      affected_resources: ['backend-pod', 'database-pod'],
      suggestions: [
        'Check network congestion on worker-2',
        'Verify database pod resource limits',
        'Consider pod affinity rules'
      ],
      timestamp: new Date().toISOString()
    },
    {
      id: 'issue-2',
      type: 'policy',
      severity: 'medium',
      title: 'Missing Network Policy',
      description: 'No network policies defined for default namespace',
      affected_resources: ['default'],
      suggestions: [
        'Implement default deny-all policy',
        'Create specific ingress/egress rules',
        'Follow zero-trust networking principles'
      ],
      timestamp: new Date().toISOString()
    }
  ]
};

export default apiClient;