package main

import (
	"strings"
	"testing"
)

func TestBuildOfflineAPIDocsHTMLUsesOnlyLocalAssets(t *testing.T) {
	html := buildOfflineAPIDocsHTML("openapi: 3.0.3\ninfo:\n  title: Test & Docs\n")

	if strings.Contains(html, "unpkg.com") || strings.Contains(html, "cdn.jsdelivr.net") {
		t.Fatalf("expected offline docs HTML without external CDN references, got %q", html)
	}
	if !strings.Contains(html, "/api/openapi.yaml") {
		t.Fatalf("expected offline docs HTML to link local schema endpoint")
	}
	if !strings.Contains(html, "Test &amp; Docs") {
		t.Fatalf("expected OpenAPI payload to be HTML-escaped in offline docs")
	}
	if !strings.Contains(html, "Offline API Docs") {
		t.Fatalf("expected offline docs header to be present")
	}
}
