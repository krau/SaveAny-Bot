//go:build !no_jsparser

package js

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MoeForge/moejs"
)

func TestPostJSON(t *testing.T) {
	var gotMethod, gotContentType string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"name":"plugin"}`))
	}))
	defer server.Close()

	rt := moejs.NewRuntime(moejs.Options{})
	if err := rt.SetGlobal("ghttp", jsGhttp(rt)); err != nil {
		t.Fatalf("SetGlobal: %v", err)
	}
	module, err := moejs.Compile("post_json.js", `
export function run(url) {
	return ghttp.postJSON(url, {query: "value"});
}`)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if err := rt.Load(module); err != nil {
		t.Fatalf("Load: %v", err)
	}
	hook, err := module.Hook("run")
	if err != nil {
		t.Fatalf("Hook: %v", err)
	}
	urlArg, err := rt.FromGo(server.URL)
	if err != nil {
		t.Fatalf("FromGo: %v", err)
	}
	result, err := rt.Call(hook, urlArg)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	resultGo, err := rt.ToGo(result)
	if err != nil {
		t.Fatalf("ToGo: %v", err)
	}
	resultMap, ok := resultGo.(map[string]any)
	if !ok {
		t.Fatalf("result type = %T, want map[string]any", resultGo)
	}
	data, ok := resultMap["data"].(map[string]any)
	if !ok {
		t.Fatalf("data type = %T, want map[string]any", resultMap["data"])
	}
	if data["ok"] != true || data["name"] != "plugin" {
		t.Fatalf("response data = %#v", data)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q, want POST", gotMethod)
	}
	if gotContentType != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", gotContentType)
	}
	if gotBody["query"] != "value" {
		t.Fatalf("request body = %#v", gotBody)
	}
}

func TestPostJSONRejectsInvalidResponseJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	defer server.Close()

	rt := moejs.NewRuntime(moejs.Options{})
	result, err := postJSON(rt, server.Client(), server.URL, moejs.Undefined())
	if err != nil {
		t.Fatalf("postJSON: %v", err)
	}
	resultGo, err := rt.ToGo(result)
	if err != nil {
		t.Fatalf("ToGo: %v", err)
	}
	resultMap, ok := resultGo.(map[string]any)
	if !ok {
		t.Fatalf("result type = %T, want map[string]any", resultGo)
	}
	if resultMap["error"] == nil {
		t.Fatalf("result = %#v, want an error", resultMap)
	}
}
