package domain

type PageResult[T any] struct {
	Items    []T `json:"items"`
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
	Total    int `json:"total"`
}

type OperationLog struct {
	ID            string `json:"id"`
	OperatorID    string `json:"operatorId"`
	OperatorName  string `json:"operatorName"`
	Action        string `json:"action"`
	ResourceType  string `json:"resourceType"`
	ResourceID    string `json:"resourceId"`
	ResourceName  string `json:"resourceName,omitempty"`
	ProjectID     string `json:"projectId,omitempty"`
	ProjectName   string `json:"projectName,omitempty"`
	EnvironmentID string `json:"environmentId,omitempty"`
	ProductName   string `json:"productName,omitempty"`
	TaskID        string `json:"taskId,omitempty"`
	Namespace     string `json:"namespace,omitempty"`
	WorkloadType  string `json:"workloadType,omitempty"`
	WorkloadName  string `json:"workloadName,omitempty"`
	ContainerName string `json:"containerName,omitempty"`
	ContainerType string `json:"containerType,omitempty"`
	Result        string `json:"result"`
	Detail        string `json:"detail"`
	CreatedAt     string `json:"createdAt"`
}

type Project struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Code         string `json:"code"`
	Description  string `json:"description"`
	Status       string `json:"status"`
	ProductCount int    `json:"productCount"`
	CreatedAt    string `json:"createdAt"`
}

type Environment struct {
	ID                  string                       `json:"id"`
	Name                string                       `json:"name"`
	Code                string                       `json:"code"`
	ProjectID           string                       `json:"projectId"`
	ProjectName         string                       `json:"projectName"`
	ProductStatus       string                       `json:"productStatus"`
	Type                string                       `json:"type"`
	DeployTargetType    string                       `json:"deployTargetType"`
	NetworkMode         string                       `json:"networkMode"`
	ClusterID           string                       `json:"clusterId"`
	Namespace           string                       `json:"namespace"`
	RegistryID          string                       `json:"registryId"`
	RegistryProject     string                       `json:"registryProject"`
	PrivateRegistryHost string                       `json:"privateRegistryHost"`
	JenkinsID           string                       `json:"jenkinsId"`
	JenkinsView         string                       `json:"jenkinsView"`
	Bindings            []EnvironmentResourceBinding `json:"bindings"`
	Status              string                       `json:"status"`
	AgentStatus         string                       `json:"agentStatus"`
	LastCheckAt         string                       `json:"lastCheckAt"`

	ClusterAPIServer      string `json:"-"`
	ClusterCredentialRef  string `json:"-"`
	ClusterKubeconfig     string `json:"-"`
	RegistryURL           string `json:"-"`
	RegistryCredentialRef string `json:"-"`
	JenkinsURL            string `json:"-"`
	JenkinsCredentialRef  string `json:"-"`
}

type EnvironmentResourceBinding struct {
	ID            string `json:"id"`
	EnvironmentID string `json:"environmentId"`
	BindingRole   string `json:"bindingRole"`
	ResourceType  string `json:"resourceType"`
	ResourceID    string `json:"resourceId"`
	ScopeType     string `json:"scopeType"`
	ScopeValue    string `json:"scopeValue"`
	IsDefault     bool   `json:"isDefault"`
}

type KubernetesCluster struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	APIServer    string   `json:"apiServer"`
	Context      string   `json:"context"`
	Status       string   `json:"status"`
	LastCheckAt  string   `json:"lastCheckAt"`
	ProbeMessage string   `json:"probeMessage"`
	Namespaces   []string `json:"namespaces"`

	CredentialRef string `json:"-"`
	Kubeconfig    string `json:"-"`
}

type HarborRegistry struct {
	ID                    string   `json:"id"`
	Name                  string   `json:"name"`
	URL                   string   `json:"url"`
	RegistryHost          string   `json:"registryHost"`
	Scheme                string   `json:"scheme"`
	Username              string   `json:"username"`
	InsecureSkipTLSVerify bool     `json:"insecureSkipTLSVerify"`
	Status                string   `json:"status"`
	LastCheckAt           string   `json:"lastCheckAt"`
	ProbeMessage          string   `json:"probeMessage"`
	Projects              []string `json:"projects"`

	CredentialRef string `json:"-"`
	Password      string `json:"-"`
}

type JenkinsInstance struct {
	ID                    string            `json:"id"`
	Name                  string            `json:"name"`
	URL                   string            `json:"url"`
	Username              string            `json:"username"`
	InsecureSkipTLSVerify bool              `json:"insecureSkipTLSVerify"`
	Status                string            `json:"status"`
	LastCheckAt           string            `json:"lastCheckAt"`
	ProbeMessage          string            `json:"probeMessage"`
	Views                 []string          `json:"views"`
	Jobs                  []string          `json:"jobs"`
	Pipelines             []JenkinsPipeline `json:"pipelines"`

	CredentialRef string `json:"-"`
	Token         string `json:"-"`
}

type JenkinsPipelineParameter struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	DefaultValue string `json:"defaultValue,omitempty"`
	Description  string `json:"description,omitempty"`
	Required     bool   `json:"required"`
}

type JenkinsPipeline struct {
	Name       string                     `json:"name"`
	View       string                     `json:"view,omitempty"`
	ViewURL    string                     `json:"viewUrl,omitempty"`
	URL        string                     `json:"url,omitempty"`
	Parameters []JenkinsPipelineParameter `json:"parameters,omitempty"`
}

type Agent struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	EnvironmentID   string        `json:"environmentId"`
	EnvironmentName string        `json:"environmentName"`
	Version         string        `json:"version"`
	Status          string        `json:"status"`
	ClaimStatus     string        `json:"claimStatus"`
	Capabilities    []string      `json:"capabilities"`
	RuntimeStatus   RuntimeStatus `json:"runtimeStatus"`
	LastHeartbeatAt string        `json:"lastHeartbeatAt"`
	CurrentTaskID   *string       `json:"currentTaskId"`
}

type RuntimeStatus struct {
	Kubernetes RuntimeComponentStatus `json:"kubernetes"`
	Harbor     RuntimeComponentStatus `json:"harbor"`
}

type RuntimeComponentStatus struct {
	Status       string            `json:"status"`
	Message      string            `json:"message"`
	UpdatedAt    string            `json:"updatedAt"`
	Endpoint     string            `json:"endpoint,omitempty"`
	RegistryHost string            `json:"registryHost,omitempty"`
	Items        []string          `json:"items"`
	Workloads    []RuntimeWorkload `json:"workloads"`
}

type RuntimeWorkload struct {
	Namespace     string             `json:"namespace"`
	Name          string             `json:"name"`
	Type          string             `json:"type"`
	Replicas      int                `json:"replicas"`
	ReadyReplicas int                `json:"readyReplicas"`
	Containers    []RuntimeContainer `json:"containers"`
}

type RuntimeContainer struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Image string `json:"image"`
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
	ServiceID                string            `json:"serviceId"`
	ServiceName              string            `json:"serviceName"`
	Namespace                string            `json:"namespace"`
	WorkloadName             string            `json:"workloadName"`
	WorkloadType             string            `json:"workloadType"`
	ImageRegistry            string            `json:"imageRegistry"`
	ImageProject             string            `json:"imageProject"`
	ImageRepository          string            `json:"imageRepository"`
	ImageTag                 string            `json:"imageTag"`
	ImageSource              string            `json:"imageSource"`
	PrivateRegistryHost      string            `json:"privateRegistryHost,omitempty"`
	PrivateRegistryConfirmed bool              `json:"privateRegistryConfirmed"`
	JenkinsJobName           string            `json:"jenkinsJobName,omitempty"`
	JenkinsJobURL            string            `json:"jenkinsJobUrl,omitempty"`
	JenkinsBranch            string            `json:"jenkinsBranch,omitempty"`
	JenkinsPipelineBound     bool              `json:"jenkinsPipelineBound"`
	PipelineBoundAt          string            `json:"pipelineBoundAt,omitempty"`
	Tags                     []ReleaseImageTag `json:"tags"`
	Publishable              bool              `json:"publishable"`
	Message                  string            `json:"message,omitempty"`
}

type ManagedService struct {
	ID                       string `json:"id"`
	ProductID                string `json:"productId"`
	Name                     string `json:"name"`
	Namespace                string `json:"namespace"`
	WorkloadName             string `json:"workloadName"`
	WorkloadType             string `json:"workloadType"`
	ContainerName            string `json:"containerName"`
	ContainerType            string `json:"containerType"`
	Image                    string `json:"image"`
	ImageRegistry            string `json:"imageRegistry"`
	ImageProject             string `json:"imageProject"`
	ImageRepository          string `json:"imageRepository"`
	ImageTag                 string `json:"imageTag"`
	ImageSource              string `json:"imageSource"`
	PrivateRegistryHost      string `json:"privateRegistryHost,omitempty"`
	PrivateRegistryConfirmed bool   `json:"privateRegistryConfirmed"`
	JenkinsJobName           string `json:"jenkinsJobName,omitempty"`
	JenkinsJobURL            string `json:"jenkinsJobUrl,omitempty"`
	JenkinsBranch            string `json:"jenkinsBranch,omitempty"`
	JenkinsPipelineBound     bool   `json:"jenkinsPipelineBound"`
	PipelineBoundAt          string `json:"pipelineBoundAt,omitempty"`
	Replicas                 int    `json:"replicas"`
	ReadyReplicas            int    `json:"readyReplicas"`
	CreatedAt                string `json:"createdAt"`
	UpdatedAt                string `json:"updatedAt"`
}

type DiscoveredService struct {
	ID                       string `json:"id"`
	ProductID                string `json:"productId"`
	Name                     string `json:"name"`
	Namespace                string `json:"namespace"`
	WorkloadName             string `json:"workloadName"`
	WorkloadType             string `json:"workloadType"`
	ContainerName            string `json:"containerName"`
	ContainerType            string `json:"containerType"`
	Image                    string `json:"image"`
	ImageRegistry            string `json:"imageRegistry"`
	ImageProject             string `json:"imageProject"`
	ImageRepository          string `json:"imageRepository"`
	ImageTag                 string `json:"imageTag"`
	ImageSource              string `json:"imageSource"`
	PrivateRegistryHost      string `json:"privateRegistryHost,omitempty"`
	PrivateRegistryConfirmed bool   `json:"privateRegistryConfirmed"`
	Replicas                 int    `json:"replicas"`
	ReadyReplicas            int    `json:"readyReplicas"`
	Managed                  bool   `json:"managed"`
}

type AdoptServiceInput struct {
	Services []DiscoveredService `json:"services"`
}

type RemoveManagedServiceInput struct {
	ServiceIDs []string `json:"serviceIds"`
}

type ConfirmServiceRegistryInput struct {
	PrivateRegistryHost string `json:"privateRegistryHost"`
}

type BindServicePipelineInput struct {
	JenkinsJobName string `json:"jenkinsJobName"`
	JenkinsJobURL  string `json:"jenkinsJobUrl"`
	JenkinsBranch  string `json:"jenkinsBranch"`
}

type ReleaseSource struct {
	EnvironmentID    string                 `json:"environmentId"`
	Services         []ReleaseSourceService `json:"services"`
	JenkinsJobs      []string               `json:"jenkinsJobs"`
	JenkinsPipelines []JenkinsPipeline      `json:"jenkinsPipelines"`
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
	ID                    string   `json:"id"`
	Type                  string   `json:"type"`
	SourceBaselineID      string   `json:"sourceBaselineId,omitempty"`
	ReleaseSource         string   `json:"releaseSource,omitempty"`
	ExecutionMode         string   `json:"executionMode,omitempty"`
	BuildID               string   `json:"buildId,omitempty"`
	BuildStatus           string   `json:"buildStatus,omitempty"`
	BuildURL              string   `json:"buildUrl,omitempty"`
	JenkinsID             string   `json:"jenkinsId,omitempty"`
	JenkinsJobName        string   `json:"jenkinsJobName,omitempty"`
	JenkinsJobURL         string   `json:"jenkinsJobUrl,omitempty"`
	ImageRepository       string   `json:"imageRepository,omitempty"`
	ImageTag              string   `json:"imageTag,omitempty"`
	ImageDigest           string   `json:"imageDigest,omitempty"`
	TargetEnvironmentID   string   `json:"targetEnvironmentId,omitempty"`
	TargetEnvironmentName string   `json:"targetEnvironmentName"`
	Status                string   `json:"status"`
	Progress              int      `json:"progress"`
	AgentName             string   `json:"agentName"`
	ServiceIDs            []string `json:"serviceIds,omitempty"`
	ServiceNames          []string `json:"serviceNames,omitempty"`
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
	JenkinsID            string
	JenkinsJobName       string
	JenkinsJobURL        string
	ImageRepository      string
	ImageTag             string
	ImageDigest          string
	TargetEnvironmentID  string
	AgentID              string
	Status               string
	Progress             int
	SelectedServiceCount int
	ServiceIDs           []string
	ServiceNames         []string
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
	JenkinsID             string           `json:"jenkinsId,omitempty"`
	JenkinsJobName        string           `json:"jenkinsJobName,omitempty"`
	JenkinsJobURL         string           `json:"jenkinsJobUrl,omitempty"`
	ImageRepository       string           `json:"imageRepository,omitempty"`
	ImageTag              string           `json:"imageTag,omitempty"`
	ImageDigest           string           `json:"imageDigest,omitempty"`
	TargetEnvironmentID   string           `json:"targetEnvironmentId,omitempty"`
	TargetEnvironmentName string           `json:"targetEnvironmentName"`
	Status                string           `json:"status"`
	Progress              int              `json:"progress"`
	AgentName             string           `json:"agentName"`
	AgentTaskID           string           `json:"agentTaskId"`
	ServiceIDs            []string         `json:"serviceIds,omitempty"`
	ServiceNames          []string         `json:"serviceNames,omitempty"`
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
