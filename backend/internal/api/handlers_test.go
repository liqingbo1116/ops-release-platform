package api

import "testing"

func TestFallbackJenkinsPipelineViewURLBuildsURLFromViewName(t *testing.T) {
	viewSet := map[string]string{
		"release-view": "release-view",
	}

	got := fallbackJenkinsPipelineViewURL("http://jenkins.local/", "release-view", viewSet)
	want := "http://jenkins.local/view/release-view/"

	if got != want {
		t.Fatalf("expected fallback view URL %s, got %s", want, got)
	}
}

func TestFallbackJenkinsPipelineViewURLPrefersBoundViewURL(t *testing.T) {
	viewSet := map[string]string{
		"http://jenkins.local/view/release-view/": "release-view",
		"release-view":                           "release-view",
	}

	got := fallbackJenkinsPipelineViewURL("http://jenkins.local", "release-view", viewSet)
	want := "http://jenkins.local/view/release-view/"

	if got != want {
		t.Fatalf("expected bound view URL %s, got %s", want, got)
	}
}
