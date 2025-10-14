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
  Stack,
  Button,
  LinearProgress,
  Card,
  CardContent,
  CardActions,
  Divider
} from '@mui/material';
import {
  ExpandMore,
  Lightbulb,
  Security,
  Speed,
  TrendingUp,
  Code,
  PlayArrow
} from '@mui/icons-material';

interface ActionableStep {
  description: string;
  command?: string;
  risk: 'low' | 'medium' | 'high';
  automated: boolean;
}

interface IntelligentInsight {
  id: string;
  category: 'optimization' | 'security' | 'reliability' | 'cost';
  priority: 'high' | 'medium' | 'low';
  title: string;
  description: string;
  impact: string;
  actions: ActionableStep[];
  metrics: Record<string, any>;
  confidence: number;
  timestamp: string;
}

interface IntelligentInsightsProps {
  insights: IntelligentInsight[];
}

const IntelligentInsights: React.FC<IntelligentInsightsProps> = ({ 
  insights
}) => {
  const getCategoryIcon = (category: string) => {
    switch (category) {
      case 'optimization':
        return <TrendingUp color="primary" />;
      case 'security':
        return <Security color="error" />;
      case 'reliability':
        return <Speed color="warning" />;
      case 'cost':
        return <TrendingUp color="success" />;
      default:
        return <Lightbulb color="info" />;
    }
  };

  const getCategoryColor = (category: string) => {
    switch (category) {
      case 'optimization':
        return 'primary';
      case 'security':
        return 'error';
      case 'reliability':
        return 'warning';
      case 'cost':
        return 'success';
      default:
        return 'info';
    }
  };

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case 'high':
        return 'error';
      case 'medium':
        return 'warning';
      case 'low':
        return 'info';
      default:
        return 'default';
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

  const groupedInsights = insights.reduce((acc, insight) => {
    if (!acc[insight.category]) {
      acc[insight.category] = [];
    }
    acc[insight.category].push(insight);
    return acc;
  }, {} as Record<string, IntelligentInsight[]>);

  const categoryOrder = ['security', 'reliability', 'optimization', 'cost'];

  if (insights.length === 0) {
    return (
      <Paper sx={{ p: 3 }}>
        <Typography variant="h6" gutterBottom>
          🤖 AI-Powered Insights
        </Typography>
        <Typography variant="body2" color="text.secondary">
          No intelligent insights available. The AI analyzer is collecting data and will provide recommendations shortly.
        </Typography>
      </Paper>
    );
  }

  return (
    <Box>
      <Typography variant="h5" gutterBottom>
        🤖 Intelligent Analysis & Recommendations
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
        AI-powered actionable insights to optimize your Kubernetes cluster
      </Typography>

      {categoryOrder.map(category => {
        const categoryInsights = groupedInsights[category] || [];
        if (categoryInsights.length === 0) return null;

        return (
          <Accordion key={category} defaultExpanded={category === 'security'}>
            <AccordionSummary expandIcon={<ExpandMore />}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                {getCategoryIcon(category)}
                <Typography variant="h6" sx={{ textTransform: 'capitalize' }}>
                  {category} Insights
                </Typography>
                <Chip 
                  label={categoryInsights.length} 
                  size="small" 
                  color={getCategoryColor(category) as any}
                />
              </Box>
            </AccordionSummary>
            <AccordionDetails>
              <Stack spacing={2}>
                {categoryInsights.map((insight) => (
                  <Card key={insight.id} variant="outlined">
                    <CardContent>
                      <Box sx={{ display: 'flex', justifyContent: 'between', alignItems: 'flex-start', mb: 2 }}>
                        <Box sx={{ flex: 1 }}>
                          <Typography variant="h6" gutterBottom>
                            {insight.title}
                          </Typography>
                          <Stack direction="row" spacing={1} sx={{ mb: 1 }}>
                            <Chip 
                              label={insight.priority} 
                              size="small" 
                              color={getPriorityColor(insight.priority) as any}
                            />
                            <Chip 
                              label={insight.category} 
                              size="small" 
                              variant="outlined"
                            />
                          </Stack>
                        </Box>
                        <Box sx={{ textAlign: 'right' }}>
                          <Typography variant="caption" color="text.secondary">
                            Confidence
                          </Typography>
                          <Typography variant="h6" color="primary">
                            {Math.round(insight.confidence * 100)}%
                          </Typography>
                          <LinearProgress 
                            variant="determinate" 
                            value={insight.confidence * 100} 
                            sx={{ width: 60, mt: 0.5 }}
                          />
                        </Box>
                      </Box>

                      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                        {insight.description}
                      </Typography>

                      <Box sx={{ mb: 2 }}>
                        <Typography variant="subtitle2" gutterBottom>
                          💡 Expected Impact
                        </Typography>
                        <Typography variant="body2" color="text.primary">
                          {insight.impact}
                        </Typography>
                      </Box>

                      {Object.keys(insight.metrics).length > 0 && (
                        <Box sx={{ mb: 2 }}>
                          <Typography variant="subtitle2" gutterBottom>
                            📊 Key Metrics
                          </Typography>
                          <Stack direction="row" spacing={1} flexWrap="wrap">
                            {Object.entries(insight.metrics).map(([key, value]) => (
                              <Chip 
                                key={key}
                                label={`${key}: ${value}`}
                                size="small"
                                variant="outlined"
                                color="info"
                              />
                            ))}
                          </Stack>
                        </Box>
                      )}

                      <Divider sx={{ my: 2 }} />

                      <Typography variant="subtitle2" gutterBottom>
                        🚀 Recommended Actions
                      </Typography>
                      <List dense>
                        {insight.actions.map((action, idx) => (
                          <ListItem 
                            key={idx}
                            sx={{ 
                              bgcolor: 'background.default', 
                              borderRadius: 1, 
                              mb: 1 
                            }}
                          >
                            <ListItemIcon>
                              {action.automated ? <PlayArrow color="success" /> : <Code />}
                            </ListItemIcon>
                            <ListItemText
                              primary={action.description}
                              secondary={
                                <Box sx={{ mt: 1 }}>
                                  {action.command && (
                                    <Typography 
                                      variant="caption" 
                                      sx={{ 
                                        fontFamily: 'monospace', 
                                        bgcolor: 'grey.100', 
                                        px: 1, 
                                        py: 0.5, 
                                        borderRadius: 0.5,
                                        display: 'block',
                                        mb: 0.5
                                      }}
                                    >
                                      {action.command}
                                    </Typography>
                                  )}
                                  <Stack direction="row" spacing={1}>
                                    <Chip 
                                      label={`Risk: ${action.risk}`}
                                      size="small"
                                      color={getRiskColor(action.risk) as any}
                                      variant="outlined"
                                    />
                                    <Chip 
                                      label={action.automated ? 'Automated' : 'Manual'}
                                      size="small"
                                      variant="outlined"
                                      color={action.automated ? 'success' : 'default'}
                                    />
                                  </Stack>
                                </Box>
                              }
                            />
                          </ListItem>
                        ))}
                      </List>
                    </CardContent>
                    
                    <CardActions>
                      <Button 
                        size="small" 
                        color="primary"
                        onClick={() => {/* TODO: Implement action execution */}}
                      >
                        Execute Actions
                      </Button>
                      <Button size="small" color="inherit">
                        Learn More
                      </Button>
                      <Box sx={{ ml: 'auto' }}>
                        <Typography variant="caption" color="text.disabled">
                          Generated: {new Date(insight.timestamp).toLocaleString()}
                        </Typography>
                      </Box>
                    </CardActions>
                  </Card>
                ))}
              </Stack>
            </AccordionDetails>
          </Accordion>
        );
      })}
    </Box>
  );
};

export default IntelligentInsights;
