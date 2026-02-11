package query

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type PrometheusClient struct {
	BaseURL    string
	Timeout    time.Duration
	HTTPClient *http.Client
}

func NewPrometheusClient(baseURL string) *PrometheusClient {
	return &PrometheusClient{
		BaseURL:    baseURL,
		Timeout:    30 * time.Second,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type QueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
	ErrorType string `json:"errorType,omitempty"`
	Error     string `json:"error,omitempty"`
}

func (c *PrometheusClient) Query(ctx context.Context, expr string) (*QueryResult, error) {
	u, _ := url.Parse(c.BaseURL + "/api/v1/query")
	q := u.Query()
	q.Set("query", expr)
	q.Set("time", fmt.Sprintf("%d", time.Now().Unix()))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prometheus query failed: %s", string(body))
	}

	var result QueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("prometheus error: %s", result.Error)
	}

	return &result, nil
}

func (c *PrometheusClient) QueryRange(ctx context.Context, expr string, start, end time.Time, step time.Duration) (*QueryResult, error) {
	u, _ := url.Parse(c.BaseURL + "/api/v1/query_range")
	q := u.Query()
	q.Set("query", expr)
	q.Set("start", fmt.Sprintf("%d", start.Unix()))
	q.Set("end", fmt.Sprintf("%d", end.Unix()))
	q.Set("step", fmt.Sprintf("%d", int(step.Seconds())))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prometheus query failed: %s", string(body))
	}

	var result QueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func GetValue(val []interface{}) float64 {
	if len(val) < 2 {
		return 0
	}
	if s, ok := val[1].(string); ok {
		var f float64
		fmt.Sscanf(s, "%f", &f)
		return f
	}
	return 0
}
