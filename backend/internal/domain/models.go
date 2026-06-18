package domain

type PageResult[T any] struct {
	Items    []T `json:"items"`
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
	Total    int `json:"total"`
}

type Environment struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Code            string `json:"code"`
	Type            string `json:"type"`
	NetworkMode     string `json:"networkMode"`
	ClusterID       string `json:"clusterId"`
	Namespace       string `json:"namespace"`
	RegistryID      string `json:"registryId"`
	RegistryProject string `json:"registryProject"`
	JenkinsID       string `json:"jenkinsId"`
	JenkinsView     string `json:"jenkinsView"`
	Status          string `json:"status"`
	AgentStatus     string `json:"agentStatus"`
	LastCheckAt     string `json:"lastCheckAt"`

	ClusterAPIServer      string `json:"-"`
	ClusterCredentialRef  string `json:"-"`
	RegistryURL           string `json:"-"`
	RegistryCredentialRef string `json:"-"`
	JenkinsURL            string `json:"-"`
	JenkinsCredentialRef  string `json:"-"`
}

type KubernetesCluster struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	APIServer     string `json:"apiServer"`
	CredentialRef string `json:"credentialRef"`
	Status        string `json:"status"`
	LastCheckAt   string `json:"lastCheckAt"`
}

type HarborRegistry struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	URL           string `json:"url"`
	CredentialRef string `json:"credentialRef"`
	Status        string `json:"status"`
	LastCheckAt   string `json:"lastCheckAt"`
}

type JenkinsInstance struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	URL           string `json:"url"`
	CredentialRef string `json:"credentialRef"`
	Status        string `json:"status"`
	LastCheckAt   string `json:"lastCheckAt"`
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
	SourceEnvironmentID   string `json:"sourceEnvironmentId"`
	SourceEnvironmentName string `json:"sourceEnvironmentName"`
	ServiceCount          int    `json:"serviceCount"`
	CreatedBy             string `json:"createdBy"`
	CreatedAt             string `json:"createdAt"`
	Status                string `json:"status"`
	Purpose               string `json:"purpose"`
	LockedAt              string `json:"lockedAt,omitempty"`
	SnapshotSource        string `json:"snapshotSource,omitempty"`
	SnapshotCollectedAt   string `json:"snapshotCollectedAt,omitempty"`
	SnapshotMode          string `json:"snapshotMode,omitempty"`
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
	SourceEnvironmentID   string         `json:"sourceEnvironmentId"`
	SourceEnvironmentName string         `json:"sourceEnvironmentName"`
	ServiceCount          int            `json:"serviceCount"`
	Status                string         `json:"status"`
	CreatedBy             string         `json:"createdBy,omitempty"`
	CreatedAt             string         `json:"createdAt,omitempty"`
	Purpose               string         `json:"purpose,omitempty"`
	LockedAt              string         `json:"lockedAt,omitempty"`
	SnapshotSource        string         `json:"snapshotSource,omitempty"`
	SnapshotCollectedAt   string         `json:"snapshotCollectedAt,omitempty"`
	SnapshotMode          string         `json:"snapshotMode,omitempty"`
	SnapshotTaskID        string         `json:"snapshotTaskId,omitempty"`
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

type ReleaseImageTag struct {
	Tag       string `json:"tag"`
	Digest    string `json:"digest,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

type ReleaseSourceService struct {
	ServiceID       string            `json:"serviceId"`
	ServiceName     string            `json:"serviceName"`
	Namespace       string            `json:"namespace"`
	WorkloadName    string            `json:"workloadName"`
	WorkloadType    string            `json:"workloadType"`
	ImageRepository string            `json:"imageRepository"`
	Tags            []ReleaseImageTag `json:"tags"`
	Publishable     bool              `json:"publishable"`
	Message         string            `json:"message,omitempty"`
}

type ReleaseSource struct {
	EnvironmentID string                 `json:"environmentId"`
	Services      []ReleaseSourceService `json:"services"`
	JenkinsJobs   []string               `json:"jenkinsJobs"`
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

type ActionRecord struct {
	Action     string `json:"action"`
	Operator   string `json:"operator"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	OccurredAt string `json:"occurredAt"`
}

type ReleaseReport struct {
	GeneratedAt         string `json:"generatedAt"`
	Operator            string `json:"operator"`
	SuccessServiceCount int    `json:"successServiceCount"`
	FailedServiceCount  int    `json:"failedServiceCount"`
	ManualConfirmCount  int    `json:"manualConfirmCount"`
	RollbackRecommended bool   `json:"rollbackRecommended"`
	Summary             string `json:"summary"`
}

type AuditSummary struct {
	Operator              string   `json:"operator"`
	TargetEnvironmentName string   `json:"targetEnvironmentName"`
	AffectedServices      []string `json:"affectedServices"`
	Result                string   `json:"result"`
	FailedStep            string   `json:"failedStep,omitempty"`
	LastAction            string   `json:"lastAction"`
	LastActionAt          string   `json:"lastActionAt"`
}

type ReleaseOrder struct {
	ID                    string `json:"id"`
	Type                  string `json:"type"`
	SourceBaselineID      string `json:"sourceBaselineId,omitempty"`
	ReleaseSource         string `json:"releaseSource,omitempty"`
	ExecutionMode         string `json:"executionMode,omitempty"`
	BuildID               string `json:"buildId,omitempty"`
	BuildStatus           string `json:"buildStatus,omitempty"`
	BuildURL              string `json:"buildUrl,omitempty"`
	ImageRepository       string `json:"imageRepository,omitempty"`
	ImageTag              string `json:"imageTag,omitempty"`
	ImageDigest           string `json:"imageDigest,omitempty"`
	TargetEnvironmentName string `json:"targetEnvironmentName"`
	Status                string `json:"status"`
	Progress              int    `json:"progress"`
	AgentName             string `json:"agentName"`
}

type CreateReleaseOrderInput struct {
	ID                   string
	Type                 string
	SourceBaselineID     string
	ReleaseSource        string
	ExecutionMode        string
	BuildID              string
	BuildStatus          string
	BuildURL             string
	ImageRepository      string
	ImageTag             string
	ImageDigest          string
	TargetEnvironmentID  string
	AgentID              string
	Status               string
	Progress             int
	SelectedServiceCount int
}

type ReleaseDetail struct {
	ID                    string           `json:"id"`
	Type                  string           `json:"type"`
	SourceBaselineID      string           `json:"sourceBaselineId,omitempty"`
	ReleaseSource         string           `json:"releaseSource,omitempty"`
	ExecutionMode         string           `json:"executionMode,omitempty"`
	BuildID               string           `json:"buildId,omitempty"`
	BuildStatus           string           `json:"buildStatus,omitempty"`
	BuildURL              string           `json:"buildUrl,omitempty"`
	ImageRepository       string           `json:"imageRepository,omitempty"`
	ImageTag              string           `json:"imageTag,omitempty"`
	ImageDigest           string           `json:"imageDigest,omitempty"`
	TargetEnvironmentName string           `json:"targetEnvironmentName"`
	Status                string           `json:"status"`
	Progress              int              `json:"progress"`
	AgentName             string           `json:"agentName"`
	AgentTaskID           string           `json:"agentTaskId"`
	Steps                 []ReleaseStep    `json:"steps"`
	Failures              []ReleaseFailure `json:"failures"`
	ActionRecords         []ActionRecord   `json:"actionRecords"`
	Report                *ReleaseReport   `json:"report,omitempty"`
	AuditSummary          *AuditSummary    `json:"auditSummary,omitempty"`
	Logs                  []string         `json:"logs"`
}

type DeployTask struct {
	ID                    string   `json:"id"`
	Type                  string   `json:"type,omitempty"`
	ProductName           string   `json:"productName"`
	TargetEnvironmentName string   `json:"targetEnvironmentName"`
	SourceBaselineID      string   `json:"sourceBaselineId,omitempty"`
	Source                string   `json:"source"`
	MissingServiceCount   int      `json:"missingServiceCount,omitempty"`
	ServiceNames          []string `json:"serviceNames,omitempty"`
	CurrentStep           string   `json:"currentStep"`
	Progress              int      `json:"progress"`
	Status                string   `json:"status"`
	AgentName             string   `json:"agentName,omitempty"`
	AgentTaskID           string   `json:"agentTaskId,omitempty"`
	NextAction            string   `json:"nextAction,omitempty"`
}

type DeployStep struct {
	ID     string `json:"id,omitempty"`
	Order  int    `json:"order"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

type DeployDetail struct {
	ID                    string         `json:"id"`
	ProductName           string         `json:"productName"`
	TargetEnvironmentName string         `json:"targetEnvironmentName"`
	Source                string         `json:"source"`
	Status                string         `json:"status"`
	Progress              int            `json:"progress"`
	AgentTaskID           string         `json:"agentTaskId"`
	Steps                 []DeployStep   `json:"steps"`
	ActionRecords         []ActionRecord `json:"actionRecords"`
	AuditSummary          *AuditSummary  `json:"auditSummary,omitempty"`
	Logs                  []string       `json:"logs"`
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
