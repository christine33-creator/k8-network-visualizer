import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Card,
  CardContent,
  Chip,
  Alert,
  AlertTitle,
  LinearProgress,
  IconButton,
  Tooltip,
  Badge,
  Stack
} from '@mui/material';
import {
  Refresh,
  Warning,
  Error as ErrorIcon,
  CheckCircle,
  TrendingDown,
  TrendingUp,
  Minimize
} from '@mui/icons-material';
import axios from 'axios';

interface Correlation {
  id: string;
  primary_metric: { type: string; name: string; value: number };
  related_metrics: Array<{ type: string; name: string; value: number }>;
  correlation_strength: number;
  correlation_type: string;
  time_detected: string;
  impact: {
    severity: string;
    business_impact: string;
    affected_services: string[];
    availability: number;
    error_rate: number;
    latency: number;
  };
  root_cause?: {
    cause: string;
    evidence: string[];
    propagation_path: Array<{ layer: string; component: string }>;
  };
}

interface HealthScore {
  overall: number;
  network: number;
  application: number;
  infrastructure: number;
  trend_direction: string;
  details: { error_rate: number; latency_impact: number; availability: number };
}

interface DashboardData {
  timestamp: string;
  overall_status: string;
  all_correlations: Correlation[];
  packet_loss_correlations: Correlation[];
  node_health_impact: Correlation[];
  cni_impact: Correlation[];
  critical_alerts: Array<{
    severity: string;
    message: string;
    source: string;
    timestamp: string;
    correlation_id: string;
    impact: string;
  }>;
  health_score: HealthScore;
}

const UnifiedDashboard: React.FC = () => {
  const [dashboardData, setDashboardData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchDashboard = async () => {
    try {
      setLoading(true);
      const response = await axios.get('/api/dashboard/unified');
      setDashboardData(response.data);
      setError(null);
    } catch (err) {
      setError('Failed to fetch dashboard data');
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchDashboard();
    const interval = setInterval(fetchDashboard, 30000);
    return () => clearInterval(interval);
  }, []);

  const getStatusColor = (status: string) => {
    const colors: Record<string, 'success' | 'warning' | 'error' | 'default'> = {
      healthy: 'success',
      warning: 'warning',
      degraded: 'error',
      critical: 'error'
    };
    return colors[status] || 'default';
  };

  const getStatusIcon = (status: string) => {
    if (status === 'healthy') return <CheckCircle color="success" />;
    if (status === 'warning') return <Warning color="warning" />;
    if (status === 'degraded' || status === 'critical') return <ErrorIcon color="error" />;
    return <Minimize />;
  };

  if (loading && !dashboardData) {
    return (
      <Box p={3}>
        <Typography>Loading dashboard...</Typography>
        <LinearProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Box p={3}>
        <Alert severity="error">{error}</Alert>
      </Box>
    );
  }

  if (!dashboardData) return null;

  const score = dashboardData.health_score;
  const getScoreColor = (value: number): 'success' | 'warning' | 'error' => {
    if (value >= 85) return 'success';
    if (value >= 60) return 'warning';
    return 'error';
  };

  return (
    <Box p={3}>
      <Stack direction="row" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h4">Unified Observability Dashboard</Typography>
        <Stack direction="row" spacing={1} alignItems="center">
          <Chip
            icon={getStatusIcon(dashboardData.overall_status)}
            label={dashboardData.overall_status.toUpperCase()}
            color={getStatusColor(dashboardData.overall_status)}
          />
          <IconButton onClick={fetchDashboard}>
            <Refresh />
          </IconButton>
        </Stack>
      </Stack>

      {/* Health Score and Alerts */}
      <Box display="flex" gap={2} mb={3} flexWrap="wrap">
        <Box flex="1" minWidth="400px">
          <Card>
            <CardContent>
              <Stack direction="row" justifyContent="space-between" mb={2}>
                <Typography variant="h6">System Health Score</Typography>
                <Tooltip title={`Trend: ${score.trend_direction}`}>
                  {score.trend_direction === 'improving' ? <TrendingUp color="success" /> :
                   score.trend_direction === 'degrading' ? <TrendingDown color="error" /> :
                   <Minimize color="action" />}
                </Tooltip>
              </Stack>

              <Box mb={2}>
                <Stack direction="row" justifyContent="space-between" mb={0.5}>
                  <Typography variant="body2">Overall</Typography>
                  <Typography variant="body2" fontWeight="bold">{score.overall}%</Typography>
                </Stack>
                <LinearProgress variant="determinate" value={score.overall} color={getScoreColor(score.overall)} sx={{ height: 8, borderRadius: 1 }} />
              </Box>

              <Box mb={1}>
                <Stack direction="row" justifyContent="space-between" mb={0.5}>
                  <Typography variant="caption">Network</Typography>
                  <Typography variant="caption">{score.network}%</Typography>
                </Stack>
                <LinearProgress variant="determinate" value={score.network} color={getScoreColor(score.network)} sx={{ height: 6, borderRadius: 1 }} />
              </Box>

              <Box mb={1}>
                <Stack direction="row" justifyContent="space-between" mb={0.5}>
                  <Typography variant="caption">Application</Typography>
                  <Typography variant="caption">{score.application}%</Typography>
                </Stack>
                <LinearProgress variant="determinate" value={score.application} color={getScoreColor(score.application)} sx={{ height: 6, borderRadius: 1 }} />
              </Box>

              <Box>
                <Stack direction="row" justifyContent="space-between" mb={0.5}>
                  <Typography variant="caption">Infrastructure</Typography>
                  <Typography variant="caption">{score.infrastructure}%</Typography>
                </Stack>
                <LinearProgress variant="determinate" value={score.infrastructure} color={getScoreColor(score.infrastructure)} sx={{ height: 6, borderRadius: 1 }} />
              </Box>
            </CardContent>
          </Card>
        </Box>

        <Box flex="1" minWidth="400px">
          <Card>
            <CardContent>
              <Typography variant="h6" mb={2}>
                <Badge badgeContent={dashboardData.critical_alerts.length} color="error">
                  Critical Alerts
                </Badge>
              </Typography>
              {dashboardData.critical_alerts.length === 0 ? (
                <Alert severity="success">
                  <AlertTitle>No Critical Alerts</AlertTitle>
                  All systems operating normally
                </Alert>
              ) : (
                <Box sx={{ maxHeight: '300px', overflow: 'auto' }}>
                  {dashboardData.critical_alerts.map((alert, index) => (
                    <Alert key={index} severity={alert.severity === 'critical' || alert.severity === 'high' ? 'error' : 'warning'} sx={{ mb: 1 }}>
                      <AlertTitle>{alert.severity.toUpperCase()}: {alert.source}</AlertTitle>
                      {alert.message}
                      <Typography variant="caption" display="block" mt={1}>
                        Impact: {alert.impact}
                      </Typography>
                    </Alert>
                  ))}
                </Box>
              )}
            </CardContent>
          </Card>
        </Box>
      </Box>

      {/* Correlations */}
      {dashboardData.all_correlations.length > 0 && (
        <Paper sx={{ p: 2, mb: 2 }}>
          <Typography variant="h6" mb={2}>
            Detected Correlations ({dashboardData.all_correlations.length})
          </Typography>
          {dashboardData.all_correlations.map((corr) => (
            <Card key={corr.id} sx={{ mb: 2 }}>
              <CardContent>
                <Stack direction="row" justifyContent="space-between" mb={1}>
                  <Typography variant="subtitle2">
                    {corr.primary_metric.name}
                    {corr.related_metrics.length > 0 && <> ↔ {corr.related_metrics[0].name}</>}
                  </Typography>
                  <Stack direction="row" spacing={1}>
                    <Chip label={corr.correlation_type} size="small" variant="outlined" />
                    <Chip
                      label={`${(corr.correlation_strength * 100).toFixed(0)}%`}
                      size="small"
                      color={corr.correlation_strength > 0.8 ? 'error' : 'warning'}
                    />
                  </Stack>
                </Stack>
                <Typography variant="body2" color="text.secondary">
                  {corr.impact.business_impact}
                </Typography>
                {corr.root_cause && (
                  <Typography variant="caption" display="block" mt={1}>
                    Root Cause: {corr.root_cause.cause}
                  </Typography>
                )}
              </CardContent>
            </Card>
          ))}
        </Paper>
      )}
    </Box>
  );
};

export default UnifiedDashboard;
