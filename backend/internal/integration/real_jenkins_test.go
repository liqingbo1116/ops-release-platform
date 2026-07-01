package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRealJenkinsAdapterTriggerAndReadBuildStatus(t *testing.T) {
	var server *httptest.Server
	var triggerPath string
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/crumbIssuer/api/json":
			if username, token, ok := r.BasicAuth(); !ok || username != "builder" || token != "secret" {
				t.Fatalf("missing basic auth on crumb request")
			}
			_ = json.NewEncoder(w).Encode(map[string]string{
				"crumbRequestField": "Jenkins-Crumb",
				"crumb":             "crumb-value",
			})
		case "/job/folder/job/build-app/buildWithParameters":
			triggerPath = r.URL.Path
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST trigger, got %s", r.Method)
			}
			if r.Header.Get("Jenkins-Crumb") != "crumb-value" {
				t.Fatalf("expected crumb header")
			}
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse form: %v", err)
			}
			if r.Form.Get("BRANCH") != "dev" {
				t.Fatalf("expected BRANCH=dev, got %q", r.Form.Get("BRANCH"))
			}
			w.Header().Set("Location", server.URL+"/queue/item/123/")
			w.WriteHeader(http.StatusCreated)
		case "/queue/item/123/api/json":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"executable": map[string]any{
					"number": 42,
					"url":    server.URL + "/job/folder/job/build-app/42/",
				},
			})
		case "/job/folder/job/build-app/42/api/json":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":        "42",
				"number":    42,
				"building":  false,
				"result":    "SUCCESS",
				"timestamp": time.Date(2026, 6, 26, 10, 0, 0, 0, time.UTC).UnixMilli(),
				"duration":  int64(120000),
				"url":       server.URL + "/job/folder/job/build-app/42/",
			})
		case "/job/folder/job/build-app/42/consoleText":
			_, _ = w.Write([]byte("line 1\nline 2\nline 3\n"))
		default:
			t.Fatalf("unexpected Jenkins path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	adapter := RealJenkinsAdapter{client: server.Client()}
	build, err := adapter.TriggerBuild(context.Background(), BuildRequest{
		JenkinsURL: server.URL,
		Username:   "builder",
		Token:      "secret",
		JobName:    "folder/build-app",
		Parameters: map[string]string{
			"BRANCH": "dev",
		},
	})
	if err != nil {
		t.Fatalf("trigger build: %v", err)
	}
	if triggerPath != "/job/folder/job/build-app/buildWithParameters" {
		t.Fatalf("expected parameterized build endpoint, got %s", triggerPath)
	}
	if build.BuildID != "42" || build.Status != "BUILDING" || !strings.HasSuffix(build.URL, "/42/") {
		t.Fatalf("unexpected build result: %+v", build)
	}

	status, err := adapter.GetBuildStatus(context.Background(), BuildStatusRequest{
		JenkinsURL:   server.URL,
		Username:     "builder",
		Token:        "secret",
		JobName:      "folder/build-app",
		BuildID:      build.BuildID,
		BuildURL:     build.URL,
		LogLineLimit: 2,
	})
	if err != nil {
		t.Fatalf("get build status: %v", err)
	}
	if status.BuildID != "42" || status.Status != "SUCCESS" {
		t.Fatalf("unexpected build status: %+v", status)
	}
	if len(status.Logs) != 2 || status.Logs[0] != "line 2" || status.Logs[1] != "line 3" {
		t.Fatalf("expected tailed console logs, got %#v", status.Logs)
	}
	if status.StartedAt == "" || status.FinishedAt == "" || status.LogURL == "" {
		t.Fatalf("expected timing and log url, got %+v", status)
	}
}

func TestRealJenkinsAdapterTriggerWithoutParametersUsesPlainBuildEndpoint(t *testing.T) {
	var triggerPath string
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/crumbIssuer/api/json":
			w.WriteHeader(http.StatusNotFound)
		case "/job/build-app/build":
			triggerPath = r.URL.Path
			if r.URL.RawQuery != "" {
				t.Fatalf("expected no build parameters, got %q", r.URL.RawQuery)
			}
			w.Header().Set("Location", server.URL+"/queue/item/321/")
			w.WriteHeader(http.StatusCreated)
		case "/queue/item/321/api/json":
			_ = json.NewEncoder(w).Encode(map[string]any{})
		default:
			t.Fatalf("unexpected Jenkins path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	adapter := RealJenkinsAdapter{client: server.Client()}
	build, err := adapter.TriggerBuild(context.Background(), BuildRequest{
		JenkinsURL: server.URL,
		JobName:    "build-app",
		Branch:     "release/2026.06",
	})
	if err != nil {
		t.Fatalf("trigger build: %v", err)
	}
	if triggerPath != "/job/build-app/build" {
		t.Fatalf("expected plain build endpoint, got %s", triggerPath)
	}
	if build.BuildID != "queue:321" {
		t.Fatalf("expected queued build id, got %+v", build)
	}
}

func TestRealJenkinsAdapterTriggerParameterizedWithoutValuesUsesParameterizedEndpoint(t *testing.T) {
	var triggerPath string
	var contentType string
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/crumbIssuer/api/json":
			w.WriteHeader(http.StatusNotFound)
		case "/job/build-app/buildWithParameters":
			triggerPath = r.URL.Path
			contentType = r.Header.Get("Content-Type")
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse form: %v", err)
			}
			if len(r.Form) != 0 {
				t.Fatalf("expected empty parameter form, got %#v", r.Form)
			}
			w.Header().Set("Location", server.URL+"/queue/item/654/")
			w.WriteHeader(http.StatusCreated)
		case "/queue/item/654/api/json":
			_ = json.NewEncoder(w).Encode(map[string]any{})
		default:
			t.Fatalf("unexpected Jenkins path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	adapter := RealJenkinsAdapter{client: server.Client()}
	build, err := adapter.TriggerBuild(context.Background(), BuildRequest{
		JenkinsURL:    server.URL,
		JobName:       "build-app",
		Parameterized: true,
	})
	if err != nil {
		t.Fatalf("trigger build: %v", err)
	}
	if triggerPath != "/job/build-app/buildWithParameters" {
		t.Fatalf("expected parameterized build endpoint, got %s", triggerPath)
	}
	if !strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		t.Fatalf("expected form content type, got %q", contentType)
	}
	if build.BuildID != "queue:654" {
		t.Fatalf("expected queued build id, got %+v", build)
	}
}

func TestRealJenkinsAdapterTriggerKeepsDiscoveredBranchParameterName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/crumbIssuer/api/json":
			w.WriteHeader(http.StatusNotFound)
		case "/job/build-app/buildWithParameters":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse form: %v", err)
			}
			if r.Form.Get("git_branch") != "feature/demo" {
				t.Fatalf("expected git_branch=feature/demo, got %q", r.Form.Get("git_branch"))
			}
			if r.Form.Get("BRANCH") != "" {
				t.Fatalf("expected no extra BRANCH parameter, got %q", r.Form.Get("BRANCH"))
			}
			w.WriteHeader(http.StatusCreated)
		default:
			t.Fatalf("unexpected Jenkins path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	adapter := RealJenkinsAdapter{client: server.Client()}
	_, err := adapter.TriggerBuild(context.Background(), BuildRequest{
		JenkinsURL: server.URL,
		JobName:    "build-app",
		Branch:     "main",
		Parameters: map[string]string{"git_branch": "feature/demo"},
	})
	if err != nil {
		t.Fatalf("trigger build: %v", err)
	}
}

func TestRealJenkinsAdapterTriggerUsesJobURLRootForCrumb(t *testing.T) {
	var crumbPath string
	var triggerPath string
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/jenkins/crumbIssuer/api/json":
			t.Fatalf("crumb should be requested from job URL root, not configured Jenkins sub path")
		case "/crumbIssuer/api/json":
			crumbPath = r.URL.Path
			_ = json.NewEncoder(w).Encode(map[string]string{
				"crumbRequestField": "Jenkins-Crumb",
				"crumb":             "root-crumb",
			})
		case "/job/team/job/build-app/build":
			triggerPath = r.URL.Path
			if r.Header.Get("Jenkins-Crumb") != "root-crumb" {
				t.Fatalf("expected root crumb header")
			}
			if r.Header.Get("Referer") != server.URL+"/job/team/job/build-app" {
				t.Fatalf("expected referer to job url, got %q", r.Header.Get("Referer"))
			}
			w.WriteHeader(http.StatusCreated)
		default:
			t.Fatalf("unexpected Jenkins path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	adapter := RealJenkinsAdapter{client: server.Client()}
	_, err := adapter.TriggerBuild(context.Background(), BuildRequest{
		JenkinsURL: server.URL + "/jenkins",
		JobName:    "build-app",
		JobURL:     server.URL + "/job/team/job/build-app",
	})
	if err != nil {
		t.Fatalf("trigger build: %v", err)
	}
	if crumbPath != "/crumbIssuer/api/json" {
		t.Fatalf("expected root crumb path, got %q", crumbPath)
	}
	if triggerPath != "/job/team/job/build-app/build" {
		t.Fatalf("expected job url build path, got %q", triggerPath)
	}
}

func TestRealJenkinsAdapterGetJobParameters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/crumbIssuer/api/json":
			w.WriteHeader(http.StatusNotFound)
		case "/view/release-view/job/build-app/api/json":
			if username, token, ok := r.BasicAuth(); !ok || username != "builder" || token != "secret" {
				t.Fatalf("missing basic auth on parameter request")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"property": []map[string]any{
					{
						"parameterDefinitions": []map[string]any{
							{
								"name":        "git_branch",
								"type":        "StringParameterDefinition",
								"description": "代码分支",
							},
							{
								"name":        "DEPLOY_ENV",
								"type":        "ChoiceParameterDefinition",
								"description": "部署环境",
								"defaultParameterValue": map[string]any{
									"value": "test",
								},
							},
						},
					},
				},
			})
		default:
			t.Fatalf("unexpected Jenkins path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	adapter := RealJenkinsAdapter{client: server.Client()}
	parameters, err := adapter.GetJobParameters(context.Background(), JobParametersRequest{
		JenkinsURL: server.URL,
		Username:   "builder",
		Token:      "secret",
		JobURL:     server.URL + "/view/release-view/job/build-app/",
	})
	if err != nil {
		t.Fatalf("get job parameters: %v", err)
	}
	if len(parameters) != 2 {
		t.Fatalf("expected 2 parameters, got %+v", parameters)
	}
	if parameters[0].Name != "git_branch" || !parameters[0].Required || parameters[0].DefaultValue != "" {
		t.Fatalf("expected git_branch to be required without default, got %+v", parameters[0])
	}
	if parameters[1].Name != "DEPLOY_ENV" || parameters[1].Required || parameters[1].DefaultValue != "test" {
		t.Fatalf("expected DEPLOY_ENV default, got %+v", parameters[1])
	}
}
