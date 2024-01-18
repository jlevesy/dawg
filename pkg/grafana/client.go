package grafana

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
)

type Client struct {
	httpClient *http.Client

	host string
}

type ClientOpt func(*Client)

func NewClient(host string, opts ...ClientOpt) *Client {
	cl := Client{httpClient: http.DefaultClient, host: host}

	for _, opt := range opts {
		opt(&cl)
	}

	return &cl
}

func WithRoundTripper(t http.RoundTripper) ClientOpt {
	return func(cl *Client) {
		cl.httpClient.Transport = t
	}
}

func WithAuthToken(tok string) ClientOpt {
	return func(cl *Client) {
		next := cl.httpClient.Transport
		if next == nil {
			next = http.DefaultTransport
		}

		cl.httpClient.Transport = &authorizedRoundTripper{
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

type APIError struct {
	Message    string `json:"message"`
	MessageID  string `json:"messageId"`
	StatusCode int    `json:"statusCode"`
	TraceID    string `json:"traceID"`
}

func (a *APIError) Error() string {
	return fmt.Sprintf("[%s] %s (code=%d, traceID=%q)", a.MessageID, a.Message, a.StatusCode, a.TraceID)
}

const createDashboardEndpoint = "/api/dashboards/db"

type CreateDashboardRequest struct {
	Dashboard json.RawMessage `json:"dashboard"`
	FolderUID string          `json:"folderUid"`
	Message   string          `json:"message"`
	Overwrite bool            `json:"overwrite"`
}

type CreateDashboardResponse struct {
	ID      int    `json:"id"`
	UID     string `json:"uid"`
	Status  string `json:"status"`
	Version int    `json:"version"`
	URL     string `json:"url"`
	Slug    string `json:"slug"`
}

func (c *Client) CreateDashboard(ctx context.Context, req *CreateDashboardRequest) (*CreateDashboardResponse, error) {
	var resp CreateDashboardResponse

	return &resp, c.do(ctx, http.MethodPost, createDashboardEndpoint, req, &resp)
}

type DeleteDashboardRequest struct {
	UID string
}

type DeleteDashboardResponse struct {
	Title   string `json:"title"`
	Message string `json:"message"`
	ID      int    `json:"id"`
}

const deleteDashboardEndpoint = "/api/dashboards/uid"

func (c *Client) DeleteDashboard(ctx context.Context, req *DeleteDashboardRequest) (*DeleteDashboardResponse, error) {
	var resp DeleteDashboardResponse

	return &resp, c.do(
		ctx,
		http.MethodDelete,
		path.Join(deleteDashboardEndpoint, req.UID),
		nil,
		&resp,
	)
}

func (c *Client) do(ctx context.Context, method, path string, reqPayload, respPayload any) error {
	var body io.Reader = http.NoBody

	if reqPayload != nil {
		var buf bytes.Buffer

		if err := json.NewEncoder(&buf).Encode(reqPayload); err != nil {
			return err
		}

		body = &buf
	}

	req, err := http.NewRequestWithContext(ctx, method, c.host+path, body)
	if err != nil {
		return err
	}

	if reqPayload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			return fmt.Errorf("could not decode response body: %w", err)
		}

		return &apiErr
	}

	return json.NewDecoder(resp.Body).Decode(&respPayload)
}
