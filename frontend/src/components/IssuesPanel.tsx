import React from 'react';
import {
  Box,
  Paper,
  Typography,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  Chip,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Stack
} from '@mui/material';
import {
  ExpandMore,
  Error,
  Warning,
  Info,
  Security,
  NetworkCheck,
  BugReport,
  Dns,
  Settings
} from '@mui/icons-material';

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

interface IssuesPanelProps {
  issues: NetworkIssue[];
}

const IssuesPanel: React.FC<IssuesPanelProps> = ({ issues }) => {
  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical':
        return 'error';
      case 'high':
        return 'warning';
      case 'medium':
        return 'info';
      case 'low':
        return 'success';
      default:
        return 'default';
    }
  };

  const getSeverityIcon = (severity: string) => {
    switch (severity) {
      case 'critical':
        return <Error color="error" />;
      case 'high':
        return <Warning color="warning" />;
      case 'medium':
        return <Info color="info" />;
      case 'low':
        return <Info color="action" />;
      default:
        return <Info />;
    }
  };

  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'policy':
        return <Security />;
      case 'connectivity':
        return <NetworkCheck />;
      case 'resource_health':
        return <BugReport />;
      case 'dns':
        return <Dns />;
      case 'configuration':
        return <Settings />;
      default:
        return <Info />;
    }
  };

  const groupedIssues = issues.reduce((acc, issue) => {
    if (!acc[issue.severity]) {
      acc[issue.severity] = [];
    }
    acc[issue.severity].push(issue);
    return acc;
  }, {} as Record<string, NetworkIssue[]>);

  const severityOrder = ['critical', 'high', 'medium', 'low'];

  if (issues.length === 0) {
    return (
      <Paper sx={{ p: 3 }}>
        <Typography variant="h6" gutterBottom>
          🎉 No Issues Detected
        </Typography>
        <Typography variant="body2" color="text.secondary">
          Your cluster appears to be healthy with no detected issues.
        </Typography>
      </Paper>
    );
  }

  return (
    <Box>
      <Typography variant="h6" gutterBottom>
        🔍 Intelligent Issue Detection
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        {issues.length} issue(s) detected across your cluster
      </Typography>

      {severityOrder.map(severity => {
        const severityIssues = groupedIssues[severity] || [];
        if (severityIssues.length === 0) return null;

        return (
          <Accordion key={severity} defaultExpanded={severity === 'critical' || severity === 'high'}>
            <AccordionSummary expandIcon={<ExpandMore />}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                {getSeverityIcon(severity)}
                <Typography variant="subtitle1" sx={{ textTransform: 'capitalize' }}>
                  {severity} Issues
                </Typography>
                <Chip 
                  label={severityIssues.length} 
                  size="small" 
                  color={getSeverityColor(severity) as any}
                />
              </Box>
            </AccordionSummary>
            <AccordionDetails>
              <List disablePadding>
                {severityIssues.map((issue) => (
                  <Paper key={issue.id} variant="outlined" sx={{ mb: 2 }}>
                    <ListItem>
                      <ListItemIcon>
                        {getTypeIcon(issue.type)}
                      </ListItemIcon>
                      <ListItemText
                        primary={
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                            <Typography variant="subtitle2">{issue.title}</Typography>
                            <Chip 
                              label={issue.type.replace('_', ' ')} 
                              size="small" 
                              variant="outlined"
                            />
                          </Box>
                        }
                        secondary={
                          <Box sx={{ mt: 1 }}>
                            <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                              {issue.description}
                            </Typography>
                            
                            {issue.affected_resources.length > 0 && (
                              <Box sx={{ mb: 1 }}>
                                <Typography variant="caption" fontWeight="bold">
                                  Affected Resources:
                                </Typography>
                                <Stack direction="row" spacing={0.5} sx={{ mt: 0.5 }}>
                                  {issue.affected_resources.map((resource, idx) => (
                                    <Chip 
                                      key={idx} 
                                      label={resource} 
                                      size="small" 
                                      variant="outlined"
                                      color="warning"
                                    />
                                  ))}
                                </Stack>
                              </Box>
                            )}

                            {issue.suggestions.length > 0 && (
                              <Box>
                                <Typography variant="caption" fontWeight="bold">
                                  Suggestions:
                                </Typography>
                                <ul style={{ margin: '4px 0', paddingLeft: '16px' }}>
                                  {issue.suggestions.map((suggestion, idx) => (
                                    <li key={idx}>
                                      <Typography variant="caption" color="text.secondary">
                                        {suggestion}
                                      </Typography>
                                    </li>
                                  ))}
                                </ul>
                              </Box>
                            )}

                            <Typography variant="caption" color="text.disabled">
                              Detected: {new Date(issue.timestamp).toLocaleString()}
                            </Typography>
                          </Box>
                        }
                      />
                    </ListItem>
                  </Paper>
                ))}
              </List>
            </AccordionDetails>
          </Accordion>
        );
      })}
    </Box>
  );
};

export default IssuesPanel;
