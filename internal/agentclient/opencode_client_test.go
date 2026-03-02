package agentclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bytesmith/internal/acp"
)

func TestResolveModelID(t *testing.T) {
	providers := []openCodeProvider{
		{
			ID: "openai",
			Models: map[string]openCodeModel{
				"gpt-5": {ID: "gpt-5", Name: "GPT-5"},
			},
		},
		{
			ID: "anthropic",
			Models: map[string]openCodeModel{
				"claude-sonnet-4": {ID: "claude-sonnet-4", Name: "Claude Sonnet 4"},
			},
		},
	}

	model, err := resolveModelID("openai/gpt-5", providers)
	if err != nil {
		t.Fatalf("expected full model id to resolve: %v", err)
	}
	if model.ProviderID != "openai" || model.ModelID != "gpt-5" {
		t.Fatalf("unexpected model resolved: %#v", model)
	}

	model, err = resolveModelID("claude-sonnet-4", providers)
	if err != nil {
		t.Fatalf("expected short model id to resolve: %v", err)
	}
	if model.ProviderID != "anthropic" || model.ModelID != "claude-sonnet-4" {
		t.Fatalf("unexpected short model resolved: %#v", model)
	}

	_, err = resolveModelID("openai/", providers)
	if err == nil || !strings.Contains(err.Error(), "invalid model id format") {
		t.Fatalf("expected invalid format error, got: %v", err)
	}

	_, err = resolveModelID("openai/missing", providers)
	if err == nil || !strings.Contains(err.Error(), "model not available") {
		t.Fatalf("expected unavailable model error, got: %v", err)
	}
}

func TestPromptIncludesModelOverride(t *testing.T) {
	var (
		captured map[string]any
		client   *OpenCodeClient
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/session/s1/message" && r.Method == http.MethodPost {
			defer r.Body.Close()
			if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
				t.Fatalf("decode prompt body: %v", err)
			}
			go client.signalPromptDone("s1", "end_turn")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	client = newTestOpenCodeClient(srv.URL)
	client.trackSession("s1", "/repo")
	client.setSessionModel("s1", openCodeModelRef{
		ProviderID: "openai",
		ModelID:    "gpt-5",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := client.Prompt(ctx, "s1", []acp.ContentBlock{{Type: "text", Text: "hello"}})
	if err != nil {
		t.Fatalf("prompt should succeed: %v", err)
	}

	modelAny, ok := captured["model"]
	if !ok {
		t.Fatalf("expected model override in prompt payload")
	}
	modelMap, ok := modelAny.(map[string]any)
	if !ok {
		t.Fatalf("unexpected model payload type: %T", modelAny)
	}
	if modelMap["providerID"] != "openai" || modelMap["modelID"] != "gpt-5" {
		t.Fatalf("unexpected model payload: %#v", modelMap)
	}
}

func TestPromptIncludesModeAsAgent(t *testing.T) {
	var (
		captured map[string]any
		client   *OpenCodeClient
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/session/s1/message" && r.Method == http.MethodPost {
			defer r.Body.Close()
			if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
				t.Fatalf("decode prompt body: %v", err)
			}
			go client.signalPromptDone("s1", "end_turn")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	client = newTestOpenCodeClient(srv.URL)
	client.trackSession("s1", "/repo")
	client.setSessionMode("s1", "plan")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := client.Prompt(ctx, "s1", []acp.ContentBlock{{Type: "text", Text: "hello"}})
	if err != nil {
		t.Fatalf("prompt should succeed: %v", err)
	}

	agentAny, ok := captured["agent"]
	if !ok {
		t.Fatalf("expected agent in prompt payload")
	}
	agent, ok := agentAny.(string)
	if !ok {
		t.Fatalf("unexpected agent payload type: %T", agentAny)
	}
	if agent != "plan" {
		t.Fatalf("unexpected agent payload: %q", agent)
	}
}

func TestSetModelStoresOverride(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/config/providers" && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"providers":[
					{"id":"openai","models":{"gpt-5":{"id":"gpt-5","name":"GPT-5"}}}
				],
				"default":{"openai":"gpt-5"}
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	client := newTestOpenCodeClient(srv.URL)
	client.trackSession("s1", "/repo")

	if err := client.SetModel(context.Background(), "s1", "openai/gpt-5"); err != nil {
		t.Fatalf("set model should succeed: %v", err)
	}

	model, ok := client.getSessionModel("s1")
	if !ok {
		t.Fatalf("expected session model override to be stored")
	}
	if model.ProviderID != "openai" || model.ModelID != "gpt-5" {
		t.Fatalf("unexpected stored model: %#v", model)
	}

	if err := client.SetModel(context.Background(), "s1", "openai/missing"); err == nil {
		t.Fatalf("expected unavailable model error")
	}

	if err := client.SetModel(context.Background(), "s1", "openai/"); err == nil {
		t.Fatalf("expected invalid model format error")
	}
}

func TestNewSessionReturnsModesFromAgents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/session" && r.Method == http.MethodPost:
			_, _ = w.Write([]byte(`{
				"id":"s1",
				"directory":"/repo",
				"title":"session",
				"time":{"updated":1}
			}`))
		case r.URL.Path == "/agent" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`[
				{"name":"plan","mode":"primary"},
				{"name":"build","mode":"primary"},
				{"name":"explore","mode":"subagent"},
				{"name":"summary","mode":"primary","hidden":true}
			]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := newTestOpenCodeClient(srv.URL)
	result, err := client.NewSession(context.Background(), "/repo", nil)
	if err != nil {
		t.Fatalf("new session should succeed: %v", err)
	}
	if result == nil || result.Modes == nil {
		t.Fatalf("expected modes on new session")
	}
	if result.Modes.CurrentModeID != "plan" {
		t.Fatalf("expected current mode plan, got: %s", result.Modes.CurrentModeID)
	}
	if len(result.Modes.AvailableModes) != 2 {
		t.Fatalf("expected only primary visible modes, got %#v", result.Modes.AvailableModes)
	}
	if result.Modes.AvailableModes[0].ID != "plan" || result.Modes.AvailableModes[1].ID != "build" {
		t.Fatalf("unexpected mode order: %#v", result.Modes.AvailableModes)
	}

	modeID, ok := client.getSessionMode("s1")
	if !ok || modeID != "plan" {
		t.Fatalf("expected session mode to be stored, got: %q, ok=%v", modeID, ok)
	}
}

func TestSetModeStoresSelection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/agent" && r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`[
				{"name":"build","mode":"primary"},
				{"name":"plan","mode":"primary"}
			]`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	client := newTestOpenCodeClient(srv.URL)
	client.trackSession("s1", "/repo")

	if err := client.SetMode(context.Background(), "s1", "PLAN"); err != nil {
		t.Fatalf("set mode should succeed: %v", err)
	}
	modeID, ok := client.getSessionMode("s1")
	if !ok || modeID != "plan" {
		t.Fatalf("expected canonical mode id to be stored, got: %q, ok=%v", modeID, ok)
	}

	if err := client.SetMode(context.Background(), "s1", "missing"); err == nil || !strings.Contains(err.Error(), "mode not available") {
		t.Fatalf("expected mode unavailable error, got: %v", err)
	}
	if err := client.SetMode(context.Background(), "s1", ""); err == nil || !strings.Contains(err.Error(), "mode id is required") {
		t.Fatalf("expected mode required error, got: %v", err)
	}
}

func TestLoadSessionReadsLastUserModel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/session/s1" && r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{
				"id":"s1",
				"directory":"/repo",
				"title":"session",
				"time":{"updated":1}
			}`))
			return
		}
		if r.URL.Path == "/session/s1/message" && r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`[
				{"info":{"role":"assistant"}},
				{"info":{"role":"user","model":{"providerID":"openai","modelID":"gpt-5"}}}
			]`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	client := newTestOpenCodeClient(srv.URL)
	if err := client.LoadSession(context.Background(), "s1", "/repo", nil); err != nil {
		t.Fatalf("load session should succeed: %v", err)
	}

	model, ok := client.getSessionModel("s1")
	if !ok {
		t.Fatalf("expected session model to be loaded from history")
	}
	if model.ProviderID != "openai" || model.ModelID != "gpt-5" {
		t.Fatalf("unexpected loaded model: %#v", model)
	}
}

func TestResumeSessionPrefersHistoryModelForCurrent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/session/s1" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{
				"id":"s1",
				"directory":"/repo",
				"title":"session",
				"time":{"updated":1}
			}`))
		case r.URL.Path == "/session/s1/message" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`[
					{"info":{"role":"user","agent":"plan","model":{"providerID":"openai","modelID":"gpt-5"}}}
				]`))
		case r.URL.Path == "/config/providers" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{
					"providers":[
						{"id":"openai","models":{"gpt-5":{"id":"gpt-5","name":"GPT-5"}}},
						{"id":"anthropic","models":{"claude-sonnet-4":{"id":"claude-sonnet-4","name":"Claude Sonnet 4"}}}
					],
					"default":{"openai":"gpt-5","anthropic":"claude-sonnet-4"}
				}`))
		case r.URL.Path == "/config" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"model":"anthropic/claude-sonnet-4"}`))
		case r.URL.Path == "/agent" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`[
					{"name":"build","mode":"primary"},
					{"name":"plan","mode":"primary"}
				]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := newTestOpenCodeClient(srv.URL)
	result, err := client.ResumeSession(context.Background(), "s1", "/repo", nil)
	if err != nil {
		t.Fatalf("resume session should succeed: %v", err)
	}
	if result == nil || result.Models == nil {
		t.Fatalf("expected models on resume")
	}
	if result.Models.CurrentModelID != "openai/gpt-5" {
		t.Fatalf("expected history model to be current, got: %s", result.Models.CurrentModelID)
	}
	if !containsModel(result.Models.AvailableModels, "openai/gpt-5") {
		t.Fatalf("expected canonical provider/model ids in available models")
	}
	if result.Modes == nil {
		t.Fatalf("expected modes on resume")
	}
	if result.Modes.CurrentModeID != "plan" {
		t.Fatalf("expected history mode to be current, got: %s", result.Modes.CurrentModeID)
	}
}

func newTestOpenCodeClient(baseURL string) *OpenCodeClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &OpenCodeClient{
		baseURL:       strings.TrimRight(baseURL, "/"),
		defaultCWD:    "/repo",
		httpClient:    &http.Client{Timeout: 2 * time.Second},
		eventHTTP:     &http.Client{Timeout: 2 * time.Second},
		stderrCh:      closedStringChannel(),
		ctx:           ctx,
		cancel:        cancel,
		sessionCWD:    make(map[string]string),
		toolCallSeen:  make(map[string]map[string]bool),
		sessionModel:  make(map[string]openCodeModelRef),
		sessionMode:   make(map[string]string),
		promptWaiters: make(map[string][]chan string),
	}
}
