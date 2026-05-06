package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-silver/steps-extend-pipeline/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T, srv *httptest.Server) api.Client {
	t.Helper()
	return api.NewClient(srv.URL, "test-slug", "test-token", log.NewLogger())
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func TestClient_Extend_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "test-token", r.Header.Get("X-HTTP-BUILD-API-TOKEN"))

		var body api.ExtendRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "workflows:\n  build:\n    steps: []", body.BitriseYML)
		assert.Equal(t, "my-pipeline", body.PipelineName)

		writeJSON(w, http.StatusOK, api.ExtendResponse{
			Workflows: []api.Workflow{
				{ID: "abc", Name: "build", DependsOn: []string{"setup"}},
			},
		})
	}))
	defer srv.Close()

	resp, err := newTestClient(t, srv).Extend("workflows:\n  build:\n    steps: []", "my-pipeline")

	require.NoError(t, err)
	require.Len(t, resp.Workflows, 1)
	assert.Equal(t, "build", resp.Workflows[0].Name)
}

func TestClient_Extend_PipelineNameOmittedWhenEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		_, hasPipelineName := body["pipeline_name"]
		assert.False(t, hasPipelineName, "pipeline_name should be omitted when empty")
		writeJSON(w, http.StatusOK, api.ExtendResponse{})
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv).Extend("yaml", "")

	require.NoError(t, err)
}

func TestClient_Extend_SurfacesErrorMsg(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error_msg": "forbidden top-level section: app"})
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv).Extend("yaml", "")

	require.Error(t, err)
	assert.Equal(t, "forbidden top-level section: app", err.Error())
}

func TestClient_Extend_RetriesOn409UntilSuccess(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			w.WriteHeader(http.StatusConflict) // retry triggered by HTTP status, not body content
			return
		}
		writeJSON(w, http.StatusOK, api.ExtendResponse{
			Workflows: []api.Workflow{{ID: "x", Name: "build", DependsOn: []string{}}},
		})
	}))
	defer srv.Close()

	resp, err := newTestClient(t, srv).Extend("yaml", "")

	require.NoError(t, err)
	assert.Equal(t, 3, callCount)
	assert.Len(t, resp.Workflows, 1)
}

func TestClient_Extend_FailsAfterMaxRetries(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusConflict) // retry triggered by HTTP status, not body content
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv).Extend("yaml", "")

	require.Error(t, err)
	assert.Equal(t, 4, callCount, "expected 1 initial attempt + 3 retries")
	assert.Contains(t, err.Error(), "409")
}

func TestClient_Extend_DoesNotRetryOn412(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		writeJSON(w, http.StatusPreconditionFailed, map[string]string{"error_msg": "already extended"})
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv).Extend("yaml", "")

	require.Error(t, err)
	assert.Equal(t, 1, callCount, "412 must not be retried")
	assert.Equal(t, "already extended", err.Error())
}
