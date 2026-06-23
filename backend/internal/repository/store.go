package repository

import (
	"time"

	"ops-release-platform/backend/internal/domain"
)

type Store interface {
	ListProjects(query string) []domain.Project
	GetProject(id string) (domain.Project, bool)
	CreateProject(input domain.Project) (domain.Project, error)
	UpdateProject(id string, input domain.Project) (domain.Project, bool, error)
	ListEnvironments(query string) []domain.Environment
	GetEnvironment(id string) (domain.Environment, bool)
	CreateEnvironment(input domain.Environment) (domain.Environment, error)
	UpdateEnvironment(id string, input domain.Environment) (domain.Environment, bool, error)
	UpdateEnvironmentCheck(id string, status string, checkedAt time.Time) (domain.Environment, bool, error)
	ListKubernetesClusters(query string) []domain.KubernetesCluster
	GetKubernetesCluster(id string) (domain.KubernetesCluster, bool)
	CreateKubernetesCluster(input domain.KubernetesCluster) (domain.KubernetesCluster, error)
	UpdateKubernetesCluster(id string, input domain.KubernetesCluster) (domain.KubernetesCluster, bool, error)
	UpdateKubernetesClusterProbe(id string, status string, message string, namespaces []string, checkedAt time.Time) (domain.KubernetesCluster, bool, error)
	ListHarborRegistries(query string) []domain.HarborRegistry
	GetHarborRegistry(id string) (domain.HarborRegistry, bool)
	CreateHarborRegistry(input domain.HarborRegistry) (domain.HarborRegistry, error)
	UpdateHarborRegistry(id string, input domain.HarborRegistry) (domain.HarborRegistry, bool, error)
	UpdateHarborRegistryProbe(id string, status string, message string, projects []string, registryHost string, checkedAt time.Time) (domain.HarborRegistry, bool, error)
	ListJenkinsInstances(query string) []domain.JenkinsInstance
	GetJenkinsInstance(id string) (domain.JenkinsInstance, bool)
	CreateJenkinsInstance(input domain.JenkinsInstance) (domain.JenkinsInstance, error)
	UpdateJenkinsInstance(id string, input domain.JenkinsInstance) (domain.JenkinsInstance, bool, error)
	UpdateJenkinsInstanceProbe(id string, status string, message string, views []string, jobs []string, checkedAt time.Time) (domain.JenkinsInstance, bool, error)
	ListAgents(query string) []domain.Agent
	GetAgent(id string) (domain.Agent, bool)
	CreateAgentRegisterToken(tokenHash string, agentID string, environmentID string, expiresAt time.Time) bool
	ConsumeAgentRegisterToken(tokenHash string, now time.Time) (string, string, bool)
	RegisterAgent(id string, environmentID string, version string, capabilities []string, tokenHash string) (domain.Agent, bool)
	ValidateAgentToken(id string, tokenHash string) bool
	ClaimAgent(id string, environmentID string) (domain.Agent, bool)
	UpsertAgent(id string, environmentID string, version string, capabilities []string, status string) (domain.Agent, bool)
	UpdateAgentHeartbeat(id string, environmentID string, version string, capabilities []string, runtimeStatus domain.RuntimeStatus) (domain.Agent, bool)
	AssignAgentTask(id string, taskID string) (domain.Agent, bool)
	GetCurrentUser() domain.CurrentUser
	ListUsers(query string) []domain.User
	ListRoles(query string) []domain.Role
	ListPermissions(query string) []domain.EnvironmentPermission
	ListChangelog(query string) []domain.ChangelogEntry
	CreateBaseline(sourceEnvironmentID string, name string, purpose string) (domain.BaselineDetail, error)
	ListBaselines(query string) []domain.Baseline
	GetBaselineDetail(id string) (domain.BaselineDetail, bool)
	LockBaseline(id string) (domain.BaselineDetail, bool)
	GetDiffResult(id string, targetEnvironmentID string) (domain.DiffResult, bool)
	ListReleaseSourceServices(query string) []domain.ReleaseSourceService
	ListManagedServices(productID string) []domain.ManagedService
	UpsertManagedServices(productID string, services []domain.DiscoveredService) ([]domain.ManagedService, error)
	RemoveManagedServices(productID string, serviceIDs []string) ([]domain.ManagedService, error)
	ConfirmManagedServiceRegistry(productID string, registryHost string, harborProjects []string) ([]domain.ManagedService, error)
	CreateReleaseOrder(input domain.CreateReleaseOrderInput) (domain.ReleaseOrder, error)
	ListReleases(query string) []domain.ReleaseOrder
	GetReleaseDetail(id string) (domain.ReleaseDetail, bool)
	ListDeployTasks(query string) []domain.DeployTask
	GetDeployDetail(id string) (domain.DeployDetail, bool)
	HasEnvironmentAction(environmentID string, action string) bool
}
