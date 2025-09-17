package visualizer

import (
	"fmt"
	"strings"

	"github.com/christine33-creator/k8-network-visualizer/pkg/models"
)

// Visualizer handles the generation of visual representations
type Visualizer struct{}

// New creates a new visualizer instance
func New() *Visualizer {
	return &Visualizer{}
}

// GenerateHTML creates an HTML page with network topology visualization
func (v *Visualizer) GenerateHTML(topology *models.NetworkTopology) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Kubernetes Network Visualizer</title>
    <script src="https://d3js.org/d3.v7.min.js"></script>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            padding: 20px;
        }
        .header {
            text-align: center;
            margin-bottom: 30px;
            border-bottom: 2px solid #e0e0e0;
            padding-bottom: 20px;
        }
        .stats {
            display: flex;
            justify-content: space-around;
            margin-bottom: 30px;
            flex-wrap: wrap;
        }
        .stat-box {
            background: #f8f9fa;
            border: 1px solid #dee2e6;
            border-radius: 6px;
            padding: 15px;
            margin: 5px;
            text-align: center;
            min-width: 120px;
        }
        .stat-number {
            font-size: 24px;
            font-weight: bold;
            color: #007bff;
        }
        .stat-label {
            color: #6c757d;
            font-size: 14px;
        }
        .section {
            margin-bottom: 30px;
        }
        .section-title {
            font-size: 18px;
            font-weight: bold;
            margin-bottom: 15px;
            color: #333;
            border-left: 4px solid #007bff;
            padding-left: 10px;
        }
        .grid {
            display: grid;
            gap: 15px;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
        }
        .card {
            background: #fff;
            border: 1px solid #e0e0e0;
            border-radius: 6px;
            padding: 15px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
        }
        .card-title {
            font-weight: bold;
            margin-bottom: 10px;
            color: #333;
        }
        .card-content {
            font-size: 14px;
            color: #666;
        }
        .status-ready { color: #28a745; }
        .status-not-ready { color: #dc3545; }
        .status-running { color: #28a745; }
        .status-pending { color: #ffc107; }
        .status-failed { color: #dc3545; }
        .network-graph {
            width: 100%%;
            height: 500px;
            border: 1px solid #ddd;
            border-radius: 6px;
        }
        .node {
            fill: #69b3a2;
            stroke: #333;
            stroke-width: 2px;
        }
        .node.pod {
            fill: #4285f4;
        }
        .node.service {
            fill: #ff9800;
        }
        .link {
            stroke: #999;
            stroke-opacity: 0.6;
            stroke-width: 2px;
        }
        .node-label {
            font-size: 12px;
            fill: #333;
            text-anchor: middle;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Kubernetes Network Visualizer</h1>
            <p>Network topology discovered at %s</p>
        </div>

        <div class="stats">
            <div class="stat-box">
                <div class="stat-number">%d</div>
                <div class="stat-label">Nodes</div>
            </div>
            <div class="stat-box">
                <div class="stat-number">%d</div>
                <div class="stat-label">Pods</div>
            </div>
            <div class="stat-box">
                <div class="stat-number">%d</div>
                <div class="stat-label">Services</div>
            </div>
            <div class="stat-box">
                <div class="stat-number">%d</div>
                <div class="stat-label">Connections</div>
            </div>
            <div class="stat-box">
                <div class="stat-number">%d</div>
                <div class="stat-label">Policies</div>
            </div>
        </div>

        <div class="section">
            <div class="section-title">Network Graph</div>
            <svg class="network-graph" id="network-graph"></svg>
        </div>

        <div class="section">
            <div class="section-title">Nodes</div>
            <div class="grid">
                %s
            </div>
        </div>

        <div class="section">
            <div class="section-title">Pods</div>
            <div class="grid">
                %s
            </div>
        </div>

        <div class="section">
            <div class="section-title">Services</div>
            <div class="grid">
                %s
            </div>
        </div>

        <div class="section">
            <div class="section-title">Network Policies</div>
            <div class="grid">
                %s
            </div>
        </div>
    </div>

    <script>
        const topology = %s;
        
        // Create network graph
        const svg = d3.select("#network-graph");
        const width = svg.node().clientWidth;
        const height = 500;
        
        svg.attr("width", width).attr("height", height);
        
        // Prepare data for D3
        const nodes = [];
        const links = [];
        
        // Add nodes
        topology.nodes.forEach(node => {
            nodes.push({
                id: node.name,
                type: 'node',
                label: node.name,
                ready: node.ready
            });
        });
        
        topology.pods.forEach(pod => {
            nodes.push({
                id: pod.namespace + '/' + pod.name,
                type: 'pod',
                label: pod.name,
                namespace: pod.namespace,
                node: pod.node
            });
            
            // Link pod to node
            links.push({
                source: pod.node,
                target: pod.namespace + '/' + pod.name
            });
        });
        
        topology.services.forEach(service => {
            nodes.push({
                id: service.namespace + '/' + service.name + '-svc',
                type: 'service',
                label: service.name,
                namespace: service.namespace
            });
            
            // Link service to endpoints
            service.endpoints.forEach(endpoint => {
                const podNode = nodes.find(n => n.type === 'pod' && n.id.includes(endpoint));
                if (podNode) {
                    links.push({
                        source: service.namespace + '/' + service.name + '-svc',
                        target: podNode.id
                    });
                }
            });
        });
        
        // Create force simulation
        const simulation = d3.forceSimulation(nodes)
            .force("link", d3.forceLink(links).id(d => d.id).distance(100))
            .force("charge", d3.forceManyBody().strength(-300))
            .force("center", d3.forceCenter(width / 2, height / 2));
        
        // Add links
        const link = svg.append("g")
            .selectAll("line")
            .data(links)
            .join("line")
            .classed("link", true);
        
        // Add nodes
        const node = svg.append("g")
            .selectAll("circle")
            .data(nodes)
            .join("circle")
            .attr("r", d => d.type === 'node' ? 15 : d.type === 'service' ? 12 : 8)
            .classed("node", true)
            .classed("pod", d => d.type === 'pod')
            .classed("service", d => d.type === 'service')
            .call(d3.drag()
                .on("start", dragstarted)
                .on("drag", dragged)
                .on("end", dragended));
        
        // Add labels
        const label = svg.append("g")
            .selectAll("text")
            .data(nodes)
            .join("text")
            .classed("node-label", true)
            .text(d => d.label)
            .attr("dy", -20);
        
        // Update positions
        simulation.on("tick", () => {
            link
                .attr("x1", d => d.source.x)
                .attr("y1", d => d.source.y)
                .attr("x2", d => d.target.x)
                .attr("y2", d => d.target.y);
            
            node
                .attr("cx", d => d.x)
                .attr("cy", d => d.y);
            
            label
                .attr("x", d => d.x)
                .attr("y", d => d.y);
        });
        
        function dragstarted(event, d) {
            if (!event.active) simulation.alphaTarget(0.3).restart();
            d.fx = d.x;
            d.fy = d.y;
        }
        
        function dragged(event, d) {
            d.fx = event.x;
            d.fy = event.y;
        }
        
        function dragended(event, d) {
            if (!event.active) simulation.alphaTarget(0);
            d.fx = null;
            d.fy = null;
        }
    </script>
</body>
</html>`,
		topology.Timestamp.Format("2006-01-02 15:04:05"),
		len(topology.Nodes),
		len(topology.Pods),
		len(topology.Services),
		len(topology.Connections),
		len(topology.Policies),
		v.generateNodesHTML(topology.Nodes),
		v.generatePodsHTML(topology.Pods),
		v.generateServicesHTML(topology.Services),
		v.generatePoliciesHTML(topology.Policies),
		v.topologyToJSON(topology),
	)
}

func (v *Visualizer) generateNodesHTML(nodes []models.Node) string {
	var html strings.Builder
	for _, node := range nodes {
		status := "status-not-ready"
		statusText := "Not Ready"
		if node.Ready {
			status = "status-ready"
			statusText = "Ready"
		}

		html.WriteString(fmt.Sprintf(`
		<div class="card">
			<div class="card-title">%s</div>
			<div class="card-content">
				<div>IP: %s</div>
				<div class="%s">Status: %s</div>
				<div>CIDR: %s</div>
			</div>
		</div>`,
			node.Name,
			node.IP,
			status,
			statusText,
			strings.Join(node.CIDRs, ", "),
		))
	}
	return html.String()
}

func (v *Visualizer) generatePodsHTML(pods []models.Pod) string {
	var html strings.Builder
	for _, pod := range pods {
		statusClass := "status-" + strings.ToLower(pod.Status)

		var ports []string
		for _, port := range pod.Ports {
			ports = append(ports, fmt.Sprintf("%d/%s", port.Port, port.Protocol))
		}

		html.WriteString(fmt.Sprintf(`
		<div class="card">
			<div class="card-title">%s/%s</div>
			<div class="card-content">
				<div>IP: %s</div>
				<div>Node: %s</div>
				<div class="%s">Status: %s</div>
				<div>Ports: %s</div>
			</div>
		</div>`,
			pod.Namespace,
			pod.Name,
			pod.IP,
			pod.Node,
			statusClass,
			pod.Status,
			strings.Join(ports, ", "),
		))
	}
	return html.String()
}

func (v *Visualizer) generateServicesHTML(services []models.Service) string {
	var html strings.Builder
	for _, service := range services {
		var ports []string
		for _, port := range service.Ports {
			ports = append(ports, fmt.Sprintf("%d/%s", port.Port, port.Protocol))
		}

		html.WriteString(fmt.Sprintf(`
		<div class="card">
			<div class="card-title">%s/%s</div>
			<div class="card-content">
				<div>Type: %s</div>
				<div>Cluster IP: %s</div>
				<div>Ports: %s</div>
				<div>Endpoints: %s</div>
			</div>
		</div>`,
			service.Namespace,
			service.Name,
			service.Type,
			service.ClusterIP,
			strings.Join(ports, ", "),
			strings.Join(service.Endpoints, ", "),
		))
	}
	return html.String()
}

func (v *Visualizer) generatePoliciesHTML(policies []models.NetworkPolicy) string {
	var html strings.Builder
	for _, policy := range policies {
		html.WriteString(fmt.Sprintf(`
		<div class="card">
			<div class="card-title">%s/%s</div>
			<div class="card-content">
				<div>Ingress Rules: %d</div>
				<div>Egress Rules: %d</div>
			</div>
		</div>`,
			policy.Namespace,
			policy.Name,
			len(policy.Ingress),
			len(policy.Egress),
		))
	}
	return html.String()
}

func (v *Visualizer) topologyToJSON(topology *models.NetworkTopology) string {
	// Simplified JSON for JavaScript consumption
	return fmt.Sprintf(`{
		"nodes": %s,
		"pods": %s,
		"services": %s,
		"connections": %s
	}`,
		v.nodesToJSON(topology.Nodes),
		v.podsToJSON(topology.Pods),
		v.servicesToJSON(topology.Services),
		v.connectionsToJSON(topology.Connections),
	)
}

func (v *Visualizer) nodesToJSON(nodes []models.Node) string {
	var items []string
	for _, node := range nodes {
		items = append(items, fmt.Sprintf(`{
			"name": "%s",
			"ip": "%s",
			"ready": %t,
			"cidrs": ["%s"]
		}`, node.Name, node.IP, node.Ready, strings.Join(node.CIDRs, `", "`)))
	}
	return "[" + strings.Join(items, ",") + "]"
}

func (v *Visualizer) podsToJSON(pods []models.Pod) string {
	var items []string
	for _, pod := range pods {
		items = append(items, fmt.Sprintf(`{
			"name": "%s",
			"namespace": "%s",
			"ip": "%s",
			"node": "%s",
			"status": "%s"
		}`, pod.Name, pod.Namespace, pod.IP, pod.Node, pod.Status))
	}
	return "[" + strings.Join(items, ",") + "]"
}

func (v *Visualizer) servicesToJSON(services []models.Service) string {
	var items []string
	for _, service := range services {
		items = append(items, fmt.Sprintf(`{
			"name": "%s",
			"namespace": "%s",
			"cluster_ip": "%s",
			"type": "%s",
			"endpoints": ["%s"]
		}`, service.Name, service.Namespace, service.ClusterIP, service.Type, strings.Join(service.Endpoints, `", "`)))
	}
	return "[" + strings.Join(items, ",") + "]"
}

func (v *Visualizer) connectionsToJSON(connections []models.Connection) string {
	var items []string
	for _, conn := range connections {
		items = append(items, fmt.Sprintf(`{
			"source": "%s",
			"destination": "%s",
			"port": %d,
			"protocol": "%s",
			"status": "%s"
		}`, conn.Source, conn.Destination, conn.Port, conn.Protocol, conn.Status))
	}
	return "[" + strings.Join(items, ",") + "]"
}
