package grafana

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Client struct {
	cl *http.Client

	host string
}

type ClientOpt func(*Client)

func NewClient(host string, opts ...ClientOpt) *Client {
	cl := Client{cl: http.DefaultClient, host: host}

	for _, opt := range opts {
		opt(&cl)
	}

	return &cl
}

func WithAuthToken(tok string) ClientOpt {
	return func(cl *Client) {
		next := cl.cl.Transport
		if next == nil {
			next = http.DefaultTransport
		}

		cl.cl.Transport = &authorizedRoundTripper{
			next: next,
			tok:  tok,
		}

	}
}

type authorizedRoundTripper struct {
	next http.RoundTripper
	tok  string
}

func (a *authorizedRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Authorization", "Bearer "+a.tok)
	return a.next.RoundTrip(r)
}

const dashboardDbEndpoint = "/api/dashboards/db"

type APIError struct {
	Message    string `json:"message"`
	MessageID  string `json:"messageId"`
	StatusCode int    `json:"statusCode"`
	TraceID    string `json:"traceID"`
}

func (a *APIError) Error() string {
	return fmt.Sprintf("[%s] %s (code=%d, traceID=%q)", a.MessageID, a.Message, a.StatusCode, a.TraceID)
}

type CreateDashboardRequest struct {
	Dashboard json.RawMessage `json:"dashboard"`
	FolderUID string          `json:"folderUid"`
	Message   string          `json:"message"`
	Overwrite bool            `json:"overwrite"`
}

type CreateDashboardResponse struct {
	ID      int    `json:"id"`
	Status  string `json:"status"`
	Version int    `json:"version"`
	URL     string `json:"url"`
}

func (c *Client) CreateDashboard(ctx context.Context, req *CreateDashboardRequest) (*CreateDashboardResponse, error) {
	var resp CreateDashboardResponse

	return &resp, c.sendJSON(ctx, req, &resp)
}

func (c *Client) sendJSON(ctx context.Context, reqPayload, respPayload any) error {
	var buf bytes.Buffer

	if err := json.NewEncoder(&buf).Encode(reqPayload); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+dashboardDbEndpoint, &buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.cl.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			return fmt.Errorf("could not decode response body: %w", err)
		}

		return &apiErr
	}

	return json.NewDecoder(resp.Body).Decode(&respPayload)
}
