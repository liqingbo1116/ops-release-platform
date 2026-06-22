package repository

import "time"

type ProjectModel struct {
	ID          string    `gorm:"primaryKey;size:64"`
	Name        string    `gorm:"size:128;not null"`
	Code        string    `gorm:"size:128;uniqueIndex;not null"`
	Description string    `gorm:"size:512"`
	Status      string    `gorm:"size:32;index;not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

func (ProjectModel) TableName() string {
	return "projects"
}

type ProductModel struct {
	ID          string    `gorm:"primaryKey;size:64"`
	Name        string    `gorm:"size:128;not null"`
	Code        string    `gorm:"size:128;uniqueIndex;not null"`
	Description string    `gorm:"size:512"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

func (ProductModel) TableName() string {
	return "products"
}

type ServiceModel struct {
	ID              string    `gorm:"primaryKey;size:64"`
	ProductID       string    `gorm:"size:64;index;not null"`
	Name            string    `gorm:"size:128;not null"`
	Namespace       string    `gorm:"size:128;not null"`
	WorkloadName    string    `gorm:"size:128;not null"`
	WorkloadType    string    `gorm:"size:32;not null"`
	ImageRepository string    `gorm:"size:512;not null"`
	HealthCheckPath string    `gorm:"size:256"`
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`
}

func (ServiceModel) TableName() string {
	return "services"
}

type EnvironmentModel struct {
	ID               string     `gorm:"primaryKey;size:64"`
	Name             string     `gorm:"size:128;not null"`
	Code             string     `gorm:"size:128;uniqueIndex;not null"`
	ProjectID        string     `gorm:"size:64;index"`
	ProductStatus    string     `gorm:"size:32;index;not null;default:UNBOUND"`
	Type             string     `gorm:"size:32;index;not null"`
	DeployTargetType string     `gorm:"size:32;not null;default:KUBERNETES"`
	NetworkMode      string     `gorm:"size:32;not null"`
	ClusterID        string     `gorm:"size:64"`
	Namespace        string     `gorm:"size:128"`
	RegistryID       string     `gorm:"size:64"`
	RegistryProject  string     `gorm:"size:128"`
	JenkinsID        string     `gorm:"size:64"`
	JenkinsView      string     `gorm:"size:128"`
	AgentID          string     `gorm:"size:64"`
	Status           string     `gorm:"size:32;index;not null"`
	LastCheckAt      *time.Time `gorm:"index"`
	CreatedAt        time.Time  `gorm:"autoCreateTime"`
	UpdatedAt        time.Time  `gorm:"autoUpdateTime"`
}

func (EnvironmentModel) TableName() string {
	return "environments"
}

type EnvironmentResourceBindingModel struct {
	ID            string    `gorm:"primaryKey;size:128"`
	EnvironmentID string    `gorm:"size:64;index;not null"`
	ResourceType  string    `gorm:"size:32;index;not null"`
	ResourceID    string    `gorm:"size:64;index;not null"`
	ScopeType     string    `gorm:"size:32;not null"`
	ScopeValue    string    `gorm:"size:128;not null"`
	IsDefault     bool      `gorm:"index;not null"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
}

func (EnvironmentResourceBindingModel) TableName() string {
	return "environment_resource_bindings"
}

type KubernetesClusterModel struct {
	ID            string     `gorm:"primaryKey;size:64"`
	Name          string     `gorm:"size:128;not null"`
	APIServer     string     `gorm:"size:512;not null"`
	CredentialRef string     `gorm:"size:256"`
	Kubeconfig    string     `gorm:"type:text"`
	Context       string     `gorm:"size:128"`
	Namespaces    []string   `gorm:"serializer:json;type:jsonb"`
	ProbeMessage  string     `gorm:"size:512"`
	Status        string     `gorm:"size:32;index;not null"`
	LastCheckAt   *time.Time `gorm:"index"`
	CreatedAt     time.Time  `gorm:"autoCreateTime"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime"`
}

func (KubernetesClusterModel) TableName() string {
	return "kubernetes_clusters"
}

type HarborRegistryModel struct {
	ID                    string `gorm:"primaryKey;size:64"`
	Name                  string `gorm:"size:128;not null"`
	URL                   string `gorm:"size:512;not null"`
	Scheme                string `gorm:"size:16"`
	Username              string `gorm:"size:128"`
	Password              string `gorm:"type:text"`
	CredentialRef         string `gorm:"size:256"`
	InsecureSkipTLSVerify bool
	Projects              []string   `gorm:"serializer:json;type:jsonb"`
	ProbeMessage          string     `gorm:"size:512"`
	Status                string     `gorm:"size:32;index;not null"`
	LastCheckAt           *time.Time `gorm:"index"`
	CreatedAt             time.Time  `gorm:"autoCreateTime"`
	UpdatedAt             time.Time  `gorm:"autoUpdateTime"`
}

func (HarborRegistryModel) TableName() string {
	return "harbor_registries"
}

type JenkinsInstanceModel struct {
	ID                    string `gorm:"primaryKey;size:64"`
	Name                  string `gorm:"size:128;not null"`
	URL                   string `gorm:"size:512;not null"`
	Username              string `gorm:"size:128"`
	Token                 string `gorm:"type:text"`
	CredentialRef         string `gorm:"size:256"`
	InsecureSkipTLSVerify bool
	Views                 []string   `gorm:"serializer:json;type:jsonb"`
	Jobs                  []string   `gorm:"serializer:json;type:jsonb"`
	ProbeMessage          string     `gorm:"size:512"`
	Status                string     `gorm:"size:32;index;not null"`
	LastCheckAt           *time.Time `gorm:"index"`
	CreatedAt             time.Time  `gorm:"autoCreateTime"`
	UpdatedAt             time.Time  `gorm:"autoUpdateTime"`
}

func (JenkinsInstanceModel) TableName() string {
	return "jenkins_instances"
}

type AgentModel struct {
	ID              string     `gorm:"primaryKey;size:64"`
	Name            string     `gorm:"size:128;not null"`
	EnvironmentID   string     `gorm:"size:64;index"`
	Version         string     `gorm:"size:64;not null"`
	Status          string     `gorm:"size:32;index;not null"`
	ClaimStatus     string     `gorm:"size:32;index;not null;default:PENDING_CLAIM"`
	TokenHash       string     `gorm:"size:128;index"`
	Capabilities    []string   `gorm:"serializer:json;type:jsonb;not null"`
	LastHeartbeatAt *time.Time `gorm:"index"`
	CurrentTaskID   string     `gorm:"size:64"`
	CreatedAt       time.Time  `gorm:"autoCreateTime"`
	UpdatedAt       time.Time  `gorm:"autoUpdateTime"`
}

func (AgentModel) TableName() string {
	return "agents"
}

type AgentRegisterTokenModel struct {
	ID            string     `gorm:"primaryKey;size:64"`
	TokenHash     string     `gorm:"size:128;uniqueIndex;not null"`
	AgentID       string     `gorm:"size:64;index"`
	EnvironmentID string     `gorm:"size:64;index"`
	ExpiresAt     time.Time  `gorm:"index;not null"`
	UsedAt        *time.Time `gorm:"index"`
	CreatedAt     time.Time  `gorm:"autoCreateTime"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime"`
}

func (AgentRegisterTokenModel) TableName() string {
	return "agent_register_tokens"
}

type EnvironmentBaselineModel struct {
	ID                  string     `gorm:"primaryKey;size:64"`
	Name                string     `gorm:"size:128;not null"`
	SourceEnvironmentID string     `gorm:"size:64;index;not null"`
	ProductID           string     `gorm:"size:64;index"`
	ServiceCount        int        `gorm:"not null"`
	Status              string     `gorm:"size:32;index;not null"`
	Purpose             string     `gorm:"size:512"`
	CreatedBy           string     `gorm:"size:64;not null"`
	CreatedAt           time.Time  `gorm:"autoCreateTime"`
	LockedAt            *time.Time `gorm:"index"`
}

func (EnvironmentBaselineModel) TableName() string {
	return "environment_baselines"
}

type BaselineServiceItemModel struct {
	ID            uint   `gorm:"primaryKey"`
	BaselineID    string `gorm:"size:64;uniqueIndex:idx_baseline_service;not null"`
	ServiceID     string `gorm:"size:64;uniqueIndex:idx_baseline_service;not null"`
	ServiceName   string `gorm:"size:128;not null"`
	Namespace     string `gorm:"size:128;not null"`
	WorkloadName  string `gorm:"size:128;not null"`
	WorkloadType  string `gorm:"size:32;not null"`
	Image         string `gorm:"size:512;not null"`
	Tag           string `gorm:"size:128;not null"`
	Digest        string `gorm:"size:128"`
	Replicas      int
	ReadyReplicas int
	HealthStatus  string    `gorm:"size:32;index;not null"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
}

func (BaselineServiceItemModel) TableName() string {
	return "baseline_service_items"
}

type ReleaseOrderModel struct {
	ID                   string    `gorm:"primaryKey;size:64"`
	Type                 string    `gorm:"size:32;index;not null"`
	SourceBaselineID     string    `gorm:"size:64;index"`
	ReleaseSource        string    `gorm:"size:64;index"`
	ExecutionMode        string    `gorm:"size:64"`
	BuildID              string    `gorm:"size:128"`
	BuildStatus          string    `gorm:"size:64"`
	BuildURL             string    `gorm:"size:512"`
	ImageRepository      string    `gorm:"size:512"`
	ImageTag             string    `gorm:"size:128"`
	ImageDigest          string    `gorm:"size:128"`
	TargetEnvironmentID  string    `gorm:"size:64;index;not null"`
	AgentID              string    `gorm:"size:64;index"`
	Status               string    `gorm:"size:32;index;not null"`
	Progress             int       `gorm:"not null"`
	SelectedServiceCount int       `gorm:"not null"`
	CreatedBy            string    `gorm:"size:64;not null"`
	CreatedAt            time.Time `gorm:"autoCreateTime"`
	UpdatedAt            time.Time `gorm:"autoUpdateTime"`
}

func (ReleaseOrderModel) TableName() string {
	return "release_orders"
}

type DeployTaskModel struct {
	ID                  string    `gorm:"primaryKey;size:64"`
	ProductID           string    `gorm:"size:64;index;not null"`
	TargetEnvironmentID string    `gorm:"size:64;index;not null"`
	SourceType          string    `gorm:"size:32;not null"`
	SourceRef           string    `gorm:"size:256;not null"`
	Status              string    `gorm:"size:32;index;not null"`
	CurrentStepID       string    `gorm:"size:64"`
	Progress            int       `gorm:"not null"`
	CreatedBy           string    `gorm:"size:64;not null"`
	CreatedAt           time.Time `gorm:"autoCreateTime"`
	UpdatedAt           time.Time `gorm:"autoUpdateTime"`
}

func (DeployTaskModel) TableName() string {
	return "deploy_tasks"
}

type DeployStepModel struct {
	ID           string     `gorm:"primaryKey;size:64"`
	DeployTaskID string     `gorm:"size:64;index;not null"`
	Name         string     `gorm:"size:128;not null"`
	Type         string     `gorm:"size:32;not null"`
	Status       string     `gorm:"size:32;index;not null"`
	Order        int        `gorm:"column:step_order;not null"`
	RetryCount   int        `gorm:"not null"`
	StartedAt    *time.Time `gorm:"index"`
	FinishedAt   *time.Time `gorm:"index"`
	CreatedAt    time.Time  `gorm:"autoCreateTime"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime"`
}

func (DeployStepModel) TableName() string {
	return "deploy_steps"
}

type AgentTaskModel struct {
	ID            string            `gorm:"primaryKey;size:64"`
	Type          string            `gorm:"size:64;index;not null"`
	Action        string            `gorm:"size:128;not null"`
	Status        string            `gorm:"size:32;index;not null"`
	Step          string            `gorm:"size:128;not null"`
	AgentID       string            `gorm:"size:64;index"`
	EnvironmentID string            `gorm:"size:64;index"`
	LeaseID       string            `gorm:"size:128;index"`
	LeaseUntil    *time.Time        `gorm:"index"`
	Payload       map[string]string `gorm:"serializer:json;type:jsonb;not null"`
	StepURL       string            `gorm:"size:512"`
	LogURL        string            `gorm:"size:512"`
	ResultURL     string            `gorm:"size:512"`
	CreatedAt     time.Time         `gorm:"autoCreateTime"`
	UpdatedAt     time.Time         `gorm:"autoUpdateTime"`
}

func (AgentTaskModel) TableName() string {
	return "agent_tasks"
}

type AgentTaskLogModel struct {
	ID        uint      `gorm:"primaryKey"`
	TaskID    string    `gorm:"size:64;index;not null"`
	Line      string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime;index"`
}

func (AgentTaskLogModel) TableName() string {
	return "agent_task_logs"
}

type UserModel struct {
	ID           string     `gorm:"primaryKey;size:64"`
	Username     string     `gorm:"size:64;uniqueIndex;not null"`
	DisplayName  string     `gorm:"size:128;not null"`
	PasswordHash string     `gorm:"size:256"`
	Status       string     `gorm:"size:32;index;not null"`
	LastLoginAt  *time.Time `gorm:"index"`
	CreatedAt    time.Time  `gorm:"autoCreateTime"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime"`
}

func (UserModel) TableName() string {
	return "users"
}

type RoleModel struct {
	Code        string    `gorm:"primaryKey;size:64"`
	Name        string    `gorm:"size:128;not null"`
	Description string    `gorm:"size:512"`
	Permissions []string  `gorm:"serializer:json;type:jsonb;not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

func (RoleModel) TableName() string {
	return "roles"
}

type UserRoleModel struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    string    `gorm:"size:64;uniqueIndex:idx_user_role;not null"`
	RoleCode  string    `gorm:"size:64;uniqueIndex:idx_user_role;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (UserRoleModel) TableName() string {
	return "user_roles"
}

type EnvironmentPermissionModel struct {
	ID            uint      `gorm:"primaryKey"`
	EnvironmentID string    `gorm:"size:64;uniqueIndex:idx_env_role;not null"`
	RoleCode      string    `gorm:"size:64;uniqueIndex:idx_env_role;not null"`
	Scope         string    `gorm:"size:32;not null"`
	Actions       []string  `gorm:"serializer:json;type:jsonb;not null"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
}

func (EnvironmentPermissionModel) TableName() string {
	return "environment_permissions"
}

type ChangelogModel struct {
	ID          string    `gorm:"primaryKey;size:64"`
	Version     string    `gorm:"size:64;uniqueIndex;not null"`
	ReleasedAt  time.Time `gorm:"index;not null"`
	Title       string    `gorm:"size:256;not null"`
	Type        string    `gorm:"size:32;index;not null"`
	Operator    string    `gorm:"size:64;not null"`
	Features    []string  `gorm:"serializer:json;type:jsonb;not null"`
	Fixes       []string  `gorm:"serializer:json;type:jsonb;not null"`
	KnownIssues []string  `gorm:"serializer:json;type:jsonb;not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

func (ChangelogModel) TableName() string {
	return "changelogs"
}

type OperationLogModel struct {
	ID            string    `gorm:"primaryKey;size:64"`
	OperatorID    string    `gorm:"size:64;index;not null"`
	OperatorName  string    `gorm:"size:128;not null"`
	Action        string    `gorm:"size:64;index;not null"`
	ResourceType  string    `gorm:"size:64;index;not null"`
	ResourceID    string    `gorm:"size:128;index;not null"`
	EnvironmentID string    `gorm:"size:64;index"`
	TaskID        string    `gorm:"size:64;index"`
	Result        string    `gorm:"size:32;index;not null"`
	Detail        string    `gorm:"type:text"`
	CreatedAt     time.Time `gorm:"autoCreateTime;index"`
}

func (OperationLogModel) TableName() string {
	return "operation_logs"
}
