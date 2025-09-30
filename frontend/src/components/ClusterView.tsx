import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Chip,
  Card,
  CardContent,
  IconButton,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  Divider,
  Stack,
  Button
} from '@mui/material';
import {
  ExpandMore,
  Storage,
  Hub,
  Computer,
  CloudQueue,
  Info,
  Refresh
} from '@mui/icons-material';

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

interface ClusterViewProps {
  pods: Pod[];
  services: Service[];
  nodes: Node[];
  onRefresh: () => void;
}

const ClusterView: React.FC<ClusterViewProps> = ({ pods, services, nodes, onRefresh }) => {
  const [expandedNamespaces, setExpandedNamespaces] = useState<string[]>(['default', 'kube-system']);
  const [selectedPod, setSelectedPod] = useState<Pod | null>(null);
  const [selectedService, setSelectedService] = useState<Service | null>(null);

  // Group pods by namespace
  const podsByNamespace = pods.reduce((acc, pod) => {
    if (!acc[pod.namespace]) {
      acc[pod.namespace] = [];
    }
    acc[pod.namespace].push(pod);
    return acc;
  }, {} as Record<string, Pod[]>);

  // Group services by namespace
  const servicesByNamespace = services.reduce((acc, service) => {
    if (!acc[service.namespace]) {
      acc[service.namespace] = [];
    }
    acc[service.namespace].push(service);
    return acc;
  }, {} as Record<string, Service[]>);

  const toggleNamespace = (namespace: string) => {
    setExpandedNamespaces(prev =>
      prev.includes(namespace)
        ? prev.filter(ns => ns !== namespace)
        : [...prev, namespace]
    );
  };

  const getStatusColor = (status: string) => {
    if (!status || typeof status !== 'string') {
      return 'default';
    }
    switch (status.toLowerCase()) {
      case 'running':
        return 'success';
      case 'pending':
        return 'warning';
      case 'failed':
      case 'error':
        return 'error';
      default:
        return 'default';
    }
  };

  const getServiceTypeColor = (type: string) => {
    switch (type) {
      case 'LoadBalancer':
        return 'primary';
      case 'NodePort':
        return 'secondary';
      case 'ClusterIP':
        return 'default';
      default:
        return 'default';
    }
  };

  const getPodPurpose = (pod: Pod): string => {
    const { labels, name } = pod;
    
    // Check common app patterns
    if (labels?.app) {
      return `Application: ${labels.app}`;
    }
    if (labels?.component) {
      return `System Component: ${labels.component}`;
    }
    if (labels?.['k8s-app']) {
      return `Kubernetes App: ${labels['k8s-app']}`;
    }
    
    // Check for common system components (with safe name check)
    if (name && typeof name === 'string') {
      if (name.includes('coredns')) return 'DNS Resolution Service';
      if (name.includes('kube-proxy')) return 'Network Proxy Service';
      if (name.includes('kube-scheduler')) return 'Pod Scheduling Service';
      if (name.includes('kube-controller')) return 'Cluster Controller Service';
      if (name.includes('kube-apiserver')) return 'API Server Service';
      if (name.includes('etcd')) return 'Cluster State Database';
      if (name.includes('dashboard')) return 'Kubernetes Dashboard UI';
      if (name.includes('storage-provisioner')) return 'Storage Management Service';
    }
    
    return 'Unknown Purpose - Check labels for more info';
  };

  const getServicePurpose = (service: Service): string => {
    const { name, labels } = service;
    
    if (labels?.app) {
      return `Exposes ${labels.app} application`;
    }
    if (labels?.['k8s-app']) {
      return `Exposes ${labels['k8s-app']} service`;
    }
    
    // Common service patterns (with safe name check)
    if (name && typeof name === 'string') {
      if (name === 'kubernetes') return 'Kubernetes API Server endpoint';
      if (name.includes('dns')) return 'DNS service for cluster';
      if (name.includes('dashboard')) return 'Web UI access for Kubernetes Dashboard';
      if (name.includes('frontend')) return 'Frontend application access';
      if (name.includes('backend') || name.includes('api')) return 'Backend API access';
      if (name.includes('database') || name.includes('db')) return 'Database access';
    }
    
    return 'Service endpoint - check configuration for purpose';
  };

  const namespaces = Object.keys(podsByNamespace).sort();

  return (
    <Box sx={{ height: '100%', overflow: 'auto' }}>
      {/* Header */}
      <Box sx={{ mb: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Typography variant="h5">Cluster Overview</Typography>
        <Button
          variant="outlined"
          startIcon={<Refresh />}
          onClick={onRefresh}
          size="small"
        >
          Refresh
        </Button>
      </Box>

      {/* Cluster Summary */}
      <Box sx={{ display: 'flex', gap: 2, mb: 3, flexWrap: 'wrap' }}>
        <Card sx={{ minWidth: 200 }}>
          <CardContent>
            <Stack direction="row" alignItems="center" spacing={2}>
              <Computer color="primary" />
              <Box>
                <Typography variant="h4">{nodes.length}</Typography>
                <Typography variant="body2" color="text.secondary">Nodes</Typography>
              </Box>
            </Stack>
          </CardContent>
        </Card>
        <Card sx={{ minWidth: 200 }}>
          <CardContent>
            <Stack direction="row" alignItems="center" spacing={2}>
              <CloudQueue color="secondary" />
              <Box>
                <Typography variant="h4">{pods.length}</Typography>
                <Typography variant="body2" color="text.secondary">Pods</Typography>
              </Box>
            </Stack>
          </CardContent>
        </Card>
        <Card sx={{ minWidth: 200 }}>
          <CardContent>
            <Stack direction="row" alignItems="center" spacing={2}>
              <Hub color="success" />
              <Box>
                <Typography variant="h4">{services.length}</Typography>
                <Typography variant="body2" color="text.secondary">Services</Typography>
              </Box>
            </Stack>
          </CardContent>
        </Card>
        <Card sx={{ minWidth: 200 }}>
          <CardContent>
            <Stack direction="row" alignItems="center" spacing={2}>
              <Storage color="warning" />
              <Box>
                <Typography variant="h4">{namespaces.length}</Typography>
                <Typography variant="body2" color="text.secondary">Namespaces</Typography>
              </Box>
            </Stack>
          </CardContent>
        </Card>
      </Box>

      {/* Namespace View */}
      {namespaces.map(namespace => (
        <Accordion
          key={namespace}
          expanded={expandedNamespaces.includes(namespace)}
          onChange={() => toggleNamespace(namespace)}
          sx={{ mb: 1 }}
        >
          <AccordionSummary expandIcon={<ExpandMore />}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
              <Storage />
              <Typography variant="h6">{namespace}</Typography>
              <Chip
                label={`${podsByNamespace[namespace]?.length || 0} pods`}
                size="small"
                color="primary"
              />
              <Chip
                label={`${servicesByNamespace[namespace]?.length || 0} services`}
                size="small"
                color="secondary"
              />
            </Box>
          </AccordionSummary>
          <AccordionDetails>
            <Box sx={{ display: 'flex', gap: 2 }}>
              {/* Pods */}
              <Box sx={{ flex: 1 }}>
                <Typography variant="subtitle1" gutterBottom>
                  <CloudQueue sx={{ mr: 1, verticalAlign: 'middle' }} />
                  Pods
                </Typography>
                <List dense>
                  {podsByNamespace[namespace]?.map(pod => (
                    <ListItem
                      key={pod.name}
                      onClick={() => setSelectedPod(pod)}
                      sx={{
                        border: 1,
                        borderColor: 'divider',
                        borderRadius: 1,
                        mb: 1,
                        cursor: 'pointer',
                        '&:hover': {
                          backgroundColor: 'action.hover'
                        }
                      }}
                    >
                      <ListItemIcon>
                        <Chip
                          label={pod.status}
                          size="small"
                          color={getStatusColor(pod.status) as any}
                        />
                      </ListItemIcon>
                      <ListItemText
                        primary={pod.name}
                        secondary={
                          <Box>
                            <Typography variant="caption" display="block">
                              {getPodPurpose(pod)}
                            </Typography>
                            <Typography variant="caption" color="text.secondary">
                              IP: {pod.pod_ip} | Node: {pod.node_name}
                            </Typography>
                          </Box>
                        }
                      />
                      <IconButton size="small">
                        <Info />
                      </IconButton>
                    </ListItem>
                  )) || <Typography variant="body2" color="text.secondary">No pods</Typography>}
                </List>
              </Box>

              {/* Services */}
              <Box sx={{ flex: 1 }}>
                <Typography variant="subtitle1" gutterBottom>
                  <Hub sx={{ mr: 1, verticalAlign: 'middle' }} />
                  Services
                </Typography>
                <List dense>
                  {servicesByNamespace[namespace]?.map(service => (
                    <ListItem
                      key={service.name}
                      onClick={() => setSelectedService(service)}
                      sx={{
                        border: 1,
                        borderColor: 'divider',
                        borderRadius: 1,
                        mb: 1,
                        cursor: 'pointer',
                        '&:hover': {
                          backgroundColor: 'action.hover'
                        }
                      }}
                    >
                      <ListItemIcon>
                        <Chip
                          label={service.type}
                          size="small"
                          color={getServiceTypeColor(service.type) as any}
                        />
                      </ListItemIcon>
                      <ListItemText
                        primary={service.name}
                        secondary={
                          <Box>
                            <Typography variant="caption" display="block">
                              {getServicePurpose(service)}
                            </Typography>
                            <Typography variant="caption" color="text.secondary">
                              Cluster IP: {service.cluster_ip}
                            </Typography>
                            <Typography variant="caption" color="text.secondary" display="block">
                              Ports: {service.ports?.map(p => `${p.port}:${p.target_port}/${p.protocol}`).join(', ')}
                            </Typography>
                          </Box>
                        }
                      />
                      <IconButton size="small">
                        <Info />
                      </IconButton>
                    </ListItem>
                  )) || <Typography variant="body2" color="text.secondary">No services</Typography>}
                </List>
              </Box>
            </Box>
          </AccordionDetails>
        </Accordion>
      ))}

      {/* Pod Details Modal */}
      {selectedPod && (
        <Paper
          sx={{
            position: 'fixed',
            top: '50%',
            left: '50%',
            transform: 'translate(-50%, -50%)',
            width: '80%',
            maxWidth: 600,
            maxHeight: '80%',
            overflow: 'auto',
            p: 3,
            zIndex: 1000
          }}
        >
          <Typography variant="h6" gutterBottom>
            Pod Details: {selectedPod.name}
          </Typography>
          <Divider sx={{ mb: 2 }} />
          
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            <Box sx={{ display: 'flex', gap: 4 }}>
              <Box>
                <Typography variant="subtitle2">Namespace:</Typography>
                <Typography variant="body2">{selectedPod.namespace}</Typography>
              </Box>
              <Box>
                <Typography variant="subtitle2">Status:</Typography>
                <Chip label={selectedPod.status} color={getStatusColor(selectedPod.status) as any} size="small" />
              </Box>
            </Box>
            <Box sx={{ display: 'flex', gap: 4 }}>
              <Box>
                <Typography variant="subtitle2">Pod IP:</Typography>
                <Typography variant="body2">{selectedPod.pod_ip}</Typography>
              </Box>
              <Box>
                <Typography variant="subtitle2">Node:</Typography>
                <Typography variant="body2">{selectedPod.node_name}</Typography>
              </Box>
            </Box>
            <Box>
              <Typography variant="subtitle2">Purpose:</Typography>
              <Typography variant="body2">{getPodPurpose(selectedPod)}</Typography>
            </Box>
            <Box>
              <Typography variant="subtitle2">Labels:</Typography>
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1, mt: 1 }}>
                {Object.entries(selectedPod.labels || {}).map(([key, value]) => (
                  <Chip key={key} label={`${key}: ${value}`} size="small" variant="outlined" />
                ))}
              </Box>
            </Box>
          </Box>
          
          <Box sx={{ mt: 3, display: 'flex', justifyContent: 'flex-end' }}>
            <Button onClick={() => setSelectedPod(null)}>Close</Button>
          </Box>
        </Paper>
      )}

      {/* Service Details Modal */}
      {selectedService && (
        <Paper
          sx={{
            position: 'fixed',
            top: '50%',
            left: '50%',
            transform: 'translate(-50%, -50%)',
            width: '80%',
            maxWidth: 600,
            maxHeight: '80%',
            overflow: 'auto',
            p: 3,
            zIndex: 1000
          }}
        >
          <Typography variant="h6" gutterBottom>
            Service Details: {selectedService.name}
          </Typography>
          <Divider sx={{ mb: 2 }} />
          
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            <Box sx={{ display: 'flex', gap: 4 }}>
              <Box>
                <Typography variant="subtitle2">Namespace:</Typography>
                <Typography variant="body2">{selectedService.namespace}</Typography>
              </Box>
              <Box>
                <Typography variant="subtitle2">Type:</Typography>
                <Chip label={selectedService.type} color={getServiceTypeColor(selectedService.type) as any} size="small" />
              </Box>
            </Box>
            <Box>
              <Typography variant="subtitle2">Cluster IP:</Typography>
              <Typography variant="body2">{selectedService.cluster_ip}</Typography>
            </Box>
            <Box>
              <Typography variant="subtitle2">Purpose:</Typography>
              <Typography variant="body2">{getServicePurpose(selectedService)}</Typography>
            </Box>
            <Box>
              <Typography variant="subtitle2">Ports:</Typography>
              <List dense>
                {selectedService.ports?.map((port, index) => (
                  <ListItem key={index}>
                    <ListItemText
                      primary={`Port ${port.port} → Target ${port.target_port}`}
                      secondary={`Protocol: ${port.protocol}`}
                    />
                  </ListItem>
                ))}
              </List>
            </Box>
            {selectedService.labels && Object.keys(selectedService.labels).length > 0 && (
              <Box>
                <Typography variant="subtitle2">Labels:</Typography>
                <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1, mt: 1 }}>
                  {Object.entries(selectedService.labels || {}).map(([key, value]) => (
                    <Chip key={key} label={`${key}: ${value}`} size="small" variant="outlined" />
                  ))}
                </Box>
              </Box>
            )}
          </Box>
          
          <Box sx={{ mt: 3, display: 'flex', justifyContent: 'flex-end' }}>
            <Button onClick={() => setSelectedService(null)}>Close</Button>
          </Box>
        </Paper>
      )}

      {/* Backdrop for modals */}
      {(selectedPod || selectedService) && (
        <Box
          sx={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            backgroundColor: 'rgba(0, 0, 0, 0.5)',
            zIndex: 999
          }}
          onClick={() => {
            setSelectedPod(null);
            setSelectedService(null);
          }}
        />
      )}
    </Box>
  );
};

export default ClusterView;
