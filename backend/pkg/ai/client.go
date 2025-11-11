package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client handles AI API interactions
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new AI client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: "https://openrouter.ai/api/v1/chat/completions",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ChatRequest represents a chat API request
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse represents a chat API response
type ChatResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
}

// Choice represents a response choice
type Choice struct {
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// GenerateScenarioAnalysis generates AI-powered analysis for a what-if scenario
func (c *Client) GenerateScenarioAnalysis(scenarioType, description, context string) (string, error) {
	prompt := fmt.Sprintf(`You are a Kubernetes expert analyzing a what-if scenario.

Scenario Type: %s
Description: %s
Current Cluster Context: %s

Provide a detailed textual analysis covering:
1. Immediate Impact - What will happen right after this change?
2. Connectivity Impact - How will pod-to-pod, pod-to-service, and external connectivity be affected?
3. Security Implications - What are the security risks or improvements?
4. Performance Impact - Expected performance changes (latency, throughput, resource usage)?
5. Reliability & Availability - Impact on service availability and fault tolerance?
6. Potential Issues - What could go wrong? Edge cases to consider?
7. Recommendations - Step-by-step recommendations to safely implement this change
8. Rollback Strategy - How to quickly revert if issues occur?

Be specific, technical, and actionable. Format the response in clear sections.`, scenarioType, description, context)

	reqBody := ChatRequest{
		Model: "meta-llama/llama-3.1-8b-instruct:free",
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// GenerateImpactSummary generates a brief impact summary
func (c *Client) GenerateImpactSummary(changes []string, affectedResources map[string]int) (string, error) {
	changesStr := ""
	for _, change := range changes {
		changesStr += "- " + change + "\n"
	}

	resourcesStr := ""
	for resource, count := range affectedResources {
		resourcesStr += fmt.Sprintf("- %d %s\n", count, resource)
	}

	prompt := fmt.Sprintf(`Summarize the impact of these Kubernetes changes in 2-3 sentences:

Changes:
%s

Affected Resources:
%s

Provide a concise, technical summary focusing on the most critical impacts.`, changesStr, resourcesStr)

	reqBody := ChatRequest{
		Model: "meta-llama/llama-3.1-8b-instruct:free",
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	return chatResp.Choices[0].Message.Content, nil
}
