package controller_test

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	dawgv1 "github.com/jlevesy/dawg/api/v1"
	"github.com/jlevesy/dawg/generator"
	"github.com/jlevesy/dawg/internal/controller"
	"github.com/jlevesy/dawg/pkg/grafana"
	"github.com/jlevesy/dawg/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate tinygo build -o ./testdata/v1.wasm -scheduler=none --no-debug -target wasi ./testdata/v1
//go:embed testdata/v1.wasm
var v1Bin []byte

//go:generate tinygo build -o ./testdata/v2.wasm -scheduler=none --no-debug -target wasi ./testdata/v2
//go:embed testdata/v2.wasm
var v2Bin []byte

var store = fakeStore{
	"fake://foo/bar/biz:v1": {
		Bin: v1Bin,
	},
	"fake://foo/bar/biz:v2": {
		Bin: v2Bin,
	},
}

func TestDashboardController_CreatesUpdatesDeletesDashboard(t *testing.T) {
	ctx := context.Background()

	k8sCluster := testutil.RunContainer(t, testutil.KWOKContainerConfig)
	t.Cleanup(func() {
		require.NoError(t, k8sCluster.Shutdown(ctx))
	})

	genRuntime, shutdown, err := generator.DefaultRuntime(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, shutdown(ctx))
	})

	var (
		grafanaBackend = stubRoundtripper{
			reqReceived: make(chan struct{}),
			resps: map[string]func() *http.Response{
				"http://somegrafana.com/api/dashboards/db": func() *http.Response {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(
							strings.NewReader(
								`{"id": 345, "uid":"dashboard-uid","version":42,"slug":"slug","url":"/url"}`,
							),
						),
					}
				},
				"http://somegrafana.com/api/dashboards/uid/dashboard-uid": func() *http.Response {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(
							strings.NewReader(
								`{"title": "foo", "message":"bar","id":42}`,
							),
						),
					}
				},
			},
		}

		grafanaClient = grafana.NewClient(
			"http://somegrafana.com",
			grafana.WithRoundTripper(&grafanaBackend),
		)
		mgr = testutil.NewTestingManager(
			t,
			&rest.Config{Host: "http://localhost:" + k8sCluster.Port},
			controller.NewDashboardReconciller(store, genRuntime, grafanaClient),
		)
		k8sClient = mgr.GetClient()
	)

	// Create a dashboard resource.
	dashboard := dawgv1.Dashboard{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-dashboard",
			Namespace: "default",
		},
		Spec: dawgv1.DashboardSpec{
			Generator: "fake://foo/bar/biz:v1",
			Config:    "some: config",
		},
	}

	err = k8sClient.Create(ctx, &dashboard)
	require.NoError(t, err)

	// This should trigger a call to Grafana.
	testutil.WaitForSignal(t, time.Second, grafanaBackend.reqReceived)

	// Assert that our dashboard has been created on Grafana's side.
	var req grafana.CreateDashboardRequest
	err = json.NewDecoder(grafanaBackend.readRequestBody(t, 0)).Decode(&req)
	require.NoError(t, err)

	assert.True(t, req.Overwrite)
	assert.Equal(t, `{"version":"v1"}`, string(req.Dashboard))

	// Assert that the resource status has been updated.
	testutil.Retry(t, 10, time.Second, func() bool {
		err = k8sClient.Get(
			ctx,
			client.ObjectKey{
				Name:      dashboard.Name,
				Namespace: dashboard.Namespace,
			},
			&dashboard,
		)
		require.NoError(t, err)
		return dashboard.Status.SyncStatus == dawgv1.DashboardStatusOK
	})

	assert.Equal(t, string(dawgv1.DashboardStatusOK), dashboard.Status.SyncStatus)
	assert.Equal(t, 345, dashboard.Status.Grafana.ID)
	assert.Equal(t, "dashboard-uid", dashboard.Status.Grafana.UID)
	assert.Equal(t, 42, dashboard.Status.Grafana.Version)
	assert.Equal(t, "/url", dashboard.Status.Grafana.URL)

	// Update the resource to use a new generator.
	dashboard.Spec.Generator = "fake://foo/bar/biz:v2"

	err = k8sClient.Update(ctx, &dashboard)
	require.NoError(t, err)

	// This should trigger a call to Grafana.
	testutil.WaitForSignal(t, time.Second, grafanaBackend.reqReceived)

	// Assert that our dashboard has been updated on Grafana's side.
	err = json.NewDecoder(grafanaBackend.readRequestBody(t, 1)).Decode(&req)
	require.NoError(t, err)

	assert.True(t, req.Overwrite)
	assert.Equal(t, `{"version":"v2"}`, string(req.Dashboard))

	err = k8sClient.Delete(ctx, &dashboard)
	require.NoError(t, err)

	// This should trigger a call to Grafana.
	testutil.WaitForSignal(t, time.Second, grafanaBackend.reqReceived)

	//  This call deletes the dasbhoard based on its UID
	deleteRequest := grafanaBackend.readRequest(t, 2)
	assert.Equal(t, http.MethodDelete, deleteRequest.Method)
	assert.Equal(t, "/api/dashboards/uid/dashboard-uid", deleteRequest.URL.Path)
}

func TestDashboardController_DeletesNOKDashboard(t *testing.T) {
	ctx := context.Background()

	k8sCluster := testutil.RunContainer(t, testutil.KWOKContainerConfig)
	t.Cleanup(func() {
		require.NoError(t, k8sCluster.Shutdown(ctx))
	})

	genRuntime, shutdown, err := generator.DefaultRuntime(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, shutdown(ctx))
	})

	var (
		grafanaBackend = stubRoundtripper{}
		grafanaClient  = grafana.NewClient(
			"http://somegrafana.com",
			grafana.WithRoundTripper(&grafanaBackend),
		)
		mgr = testutil.NewTestingManager(
			t,
			&rest.Config{Host: "http://localhost:" + k8sCluster.Port},
			controller.NewDashboardReconciller(store, genRuntime, grafanaClient),
		)
		k8sClient = mgr.GetClient()
	)

	// Create a dashboard resource with a unkowmn generator.
	dashboard := dawgv1.Dashboard{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-dashboard",
			Namespace: "default",
		},
		Spec: dawgv1.DashboardSpec{
			Generator: "fake://foo/bar/biz:vbad",
			Config:    "some: config",
		},
	}

	err = k8sClient.Create(ctx, &dashboard)
	require.NoError(t, err)

	// Assert that the dashboard is marked as NOK.
	testutil.Retry(t, 10, time.Second, func() bool {
		err = k8sClient.Get(
			ctx,
			client.ObjectKey{
				Name:      dashboard.Name,
				Namespace: dashboard.Namespace,
			},
			&dashboard,
		)
		if err != nil {
			return false
		}

		return dashboard.Status.SyncStatus == dawgv1.DashboardStatusError
	})
	assert.Equal(t, dawgv1.DashboardStatusError, dashboard.Status.SyncStatus)
	assert.Equal(t, dashboard.Status.Error, "generator not found")

	// Assert that we could delete the dashboard.
	err = k8sClient.Delete(ctx, &dashboard)
	require.NoError(t, err)
}

var errGenNotFound = errors.New("generator not found")

type fakeStore map[string]*generator.Generator

func (fs fakeStore) Load(_ context.Context, u *url.URL) (*generator.Generator, error) {
	gen, ok := fs[u.String()]
	if !ok {
		return nil, errGenNotFound
	}

	return gen, nil
}

var errRespNotFound = errors.New("resps not found")

type stubRoundtripper struct {
	mu          sync.Mutex
	reqs        []*http.Request
	resps       map[string]func() *http.Response
	reqReceived chan struct{}
}

func (c *stubRoundtripper) RoundTrip(r *http.Request) (*http.Response, error) {
	defer func() {
		go func() {
			if c.reqReceived == nil {
				panic("does not expect any calls")
			}

			c.reqReceived <- struct{}{}
		}()
	}()

	c.mu.Lock()
	defer c.mu.Unlock()

	c.reqs = append(c.reqs, r)

	respBuilder, ok := c.resps[r.URL.String()]
	if !ok {
		return nil, errRespNotFound
	}

	return respBuilder(), nil
}

func (c *stubRoundtripper) readRequest(t *testing.T, reqID int) *http.Request {
	t.Helper()

	c.mu.Lock()
	defer c.mu.Unlock()

	if reqID > len(c.reqs)-1 {
		t.Fatal("accesing an incorrect request")
	}

	return c.reqs[reqID]
}

func (c *stubRoundtripper) readRequestBody(t *testing.T, reqID int) io.Reader {
	t.Helper()

	req := c.readRequest(t, reqID)

	body := req.Body
	defer func() {
		require.NoError(t, body.Close())
	}()

	var buf bytes.Buffer
	_, err := io.Copy(&buf, body)
	require.NoError(t, err)

	req.Body = io.NopCloser(&buf)

	return &buf
}
