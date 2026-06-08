package domain

type PageResult[T any] struct {
	Items    []T `json:"items"`
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
	Total    int `json:"total"`
}

type Environment struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	Type        string `json:"type"`
	NetworkMode string `json:"networkMode"`
	Status      string `json:"status"`
	AgentStatus string `json:"agentStatus"`
	LastCheckAt string `json:"lastCheckAt"`
}

type Agent struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	EnvironmentID   string   `json:"environmentId"`
	EnvironmentName string   `json:"environmentName"`
	Version         string   `json:"version"`
	Status          string   `json:"status"`
	Capabilities    []string `json:"capabilities"`
	LastHeartbeatAt string   `json:"lastHeartbeatAt"`
	CurrentTaskID   *string  `json:"currentTaskId"`
}

type Baseline struct {
	ID                    string `json:"id"`
	Name                  string `json:"name"`
	SourceEnvironmentName string `json:"sourceEnvironmentName"`
	ServiceCount          int    `json:"serviceCount"`
	CreatedBy             string `json:"createdBy"`
	CreatedAt             string `json:"createdAt"`
	Status                string `json:"status"`
	Purpose               string `json:"purpose"`
}

type BaselineItem struct {
	ServiceID     string `json:"serviceId"`
	ServiceName   string `json:"serviceName"`
	Namespace     string `json:"namespace"`
	WorkloadName  string `json:"workloadName"`
	WorkloadType  string `json:"workloadType"`
	Tag           string `json:"tag"`
	Digest        string `json:"digest"`
	Replicas      int    `json:"replicas"`
	ReadyReplicas int    `json:"readyReplicas"`
	HealthStatus  string `json:"healthStatus"`
}

type BaselineDetail struct {
	ID                    string         `json:"id"`
	Name                  string         `json:"name"`
	SourceEnvironmentName string         `json:"sourceEnvironmentName"`
	ServiceCount          int            `json:"serviceCount"`
	Status                string         `json:"status"`
	Items                 []BaselineItem `json:"items"`
}

type DiffSummary struct {
	Consistent      int `json:"consistent"`
	NeedUpdate      int `json:"needUpdate"`
	MissingInTarget int `json:"missingInTarget"`
	WorkloadError   int `json:"workloadError"`
	Publishable     int `json:"publishable"`
}

type DiffItem struct {
	ServiceID   string  `json:"serviceId"`
	ServiceName string  `json:"serviceName"`
	Namespace   string  `json:"namespace"`
	SourceTag   string  `json:"sourceTag"`
	TargetTag   *string `json:"targetTag"`
	DiffStatus  string  `json:"diffStatus"`
	Publishable bool    `json:"publishable"`
	Strategy    string  `json:"strategy"`
}

type DiffResult struct {
	SourceBaselineID    string      `json:"sourceBaselineId"`
	TargetEnvironmentID string      `json:"targetEnvironmentId"`
	Summary             DiffSummary `json:"summary"`
	Items               []DiffItem  `json:"items"`
}

type ReleaseStep struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type ReleaseFailure struct {
	ServiceName string `json:"serviceName"`
	Reason      string `json:"reason"`
	Suggestion  string `json:"suggestion"`
}

type ReleaseOrder struct {
	ID                    string `json:"id"`
	Type                  string `json:"type"`
	SourceBaselineID      string `json:"sourceBaselineId"`
	TargetEnvironmentName string `json:"targetEnvironmentName"`
	Status                string `json:"status"`
	Progress              int    `json:"progress"`
	AgentName             string `json:"agentName"`
}

type ReleaseDetail struct {
	ID                    string           `json:"id"`
	Type                  string           `json:"type"`
	SourceBaselineID      string           `json:"sourceBaselineId"`
	TargetEnvironmentName string           `json:"targetEnvironmentName"`
	Status                string           `json:"status"`
	Progress              int              `json:"progress"`
	AgentName             string           `json:"agentName"`
	Steps                 []ReleaseStep    `json:"steps"`
	Failures              []ReleaseFailure `json:"failures"`
	Logs                  []string         `json:"logs"`
}

type DeployTask struct {
	ID                    string `json:"id"`
	ProductName           string `json:"productName"`
	TargetEnvironmentName string `json:"targetEnvironmentName"`
	Source                string `json:"source"`
	CurrentStep           string `json:"currentStep"`
	Progress              int    `json:"progress"`
	Status                string `json:"status"`
}

type DeployStep struct {
	Order  int    `json:"order"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

type DeployDetail struct {
	ID                    string       `json:"id"`
	ProductName           string       `json:"productName"`
	TargetEnvironmentName string       `json:"targetEnvironmentName"`
	Source                string       `json:"source"`
	Status                string       `json:"status"`
	Progress              int          `json:"progress"`
	Steps                 []DeployStep `json:"steps"`
	Logs                  []string     `json:"logs"`
}

type CurrentUser struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	DisplayName string   `json:"displayName"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

type User struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	DisplayName string   `json:"displayName"`
	Roles       []string `json:"roles"`
	Status      string   `json:"status"`
	LastLoginAt string   `json:"lastLoginAt"`
}

type Role struct {
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

type EnvironmentPermission struct {
	EnvironmentID   string   `json:"environmentId"`
	EnvironmentName string   `json:"environmentName"`
	RoleCode        string   `json:"roleCode"`
	Scope           string   `json:"scope"`
	Actions         []string `json:"actions"`
}

type ChangelogEntry struct {
	ID          string   `json:"id"`
	Version     string   `json:"version"`
	ReleasedAt  string   `json:"releasedAt"`
	Title       string   `json:"title"`
	Type        string   `json:"type"`
	Operator    string   `json:"operator"`
	Features    []string `json:"features"`
	Fixes       []string `json:"fixes"`
	KnownIssues []string `json:"knownIssues"`
}
