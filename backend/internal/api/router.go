package api

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"ops-release-platform/backend/internal/agent"
	"ops-release-platform/backend/internal/integration"
	"ops-release-platform/backend/internal/middleware"
	"ops-release-platform/backend/internal/repository"
)

func NewRouter(repo repository.Store, queue *agent.Queue, protocol agent.Protocol, integrations integration.Suite) *gin.Engine {
	router := gin.Default()
	router.Use(middleware.CORS())
	router.NoRoute(NoRoute)

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	api := router.Group("/api")
	api.GET("/healthz", func(c *gin.Context) {
		OK(c, gin.H{"status": "ok"})
	})

	handler := NewHandler(repo, queue, protocol, integrations)

	api.POST("/auth/login", handler.Login)
	api.POST("/auth/logout", handler.Logout)
	api.GET("/auth/me", handler.Me)
	api.GET("/users", handler.ListUsers)
	api.GET("/roles", handler.ListRoles)
	api.GET("/permissions", handler.ListPermissions)
	api.GET("/changelog", handler.ListChangelog)

	api.GET("/projects", handler.ListProjects)
	api.GET("/projects/:id", handler.GetProject)
	api.POST("/projects", handler.CreateProject)
	api.PUT("/projects/:id", handler.UpdateProject)
	api.GET("/environments", handler.ListEnvironments)
	api.GET("/environments/:id", handler.GetEnvironment)
	api.POST("/environments", handler.CreateEnvironment)
	api.PUT("/environments/:id", handler.UpdateEnvironment)
	api.POST("/environments/:id/check", handler.CheckEnvironment)
	api.POST("/environments/:id/remote-probe", handler.ProbeEnvironment)
	api.GET("/environments/:id/discovered-services", handler.ListDiscoveredEnvironmentServices)
	api.GET("/environments/:id/services", handler.ListEnvironmentServices)
	api.POST("/environments/:id/services/adopt", handler.AdoptEnvironmentServices)
	api.POST("/environments/:id/services/remove", handler.RemoveEnvironmentServices)
	api.POST("/environments/:id/services/confirm-registry", handler.ConfirmEnvironmentServiceRegistry)
	api.GET("/kubernetes-clusters", handler.ListKubernetesClusters)
	api.POST("/kubernetes-clusters", handler.CreateKubernetesCluster)
	api.PUT("/kubernetes-clusters/:id", handler.UpdateKubernetesCluster)
	api.POST("/kubernetes-clusters/:id/test", handler.TestKubernetesCluster)
	api.POST("/kubernetes-clusters/:id/refresh", handler.RefreshKubernetesCluster)
	api.GET("/harbor-registries", handler.ListHarborRegistries)
	api.POST("/harbor-registries", handler.CreateHarborRegistry)
	api.PUT("/harbor-registries/:id", handler.UpdateHarborRegistry)
	api.POST("/harbor-registries/:id/test", handler.TestHarborRegistry)
	api.POST("/harbor-registries/:id/refresh", handler.RefreshHarborRegistry)
	api.GET("/jenkins-instances", handler.ListJenkinsInstances)
	api.POST("/jenkins-instances", handler.CreateJenkinsInstance)
	api.PUT("/jenkins-instances/:id", handler.UpdateJenkinsInstance)
	api.POST("/jenkins-instances/:id/test", handler.TestJenkinsInstance)
	api.POST("/jenkins-instances/:id/refresh", handler.RefreshJenkinsInstance)

	api.GET("/agents", handler.ListAgents)
	api.POST("/agents/register-token", handler.CreateAgentRegisterToken)
	api.POST("/agents/register", handler.RegisterAgent)
	api.POST("/agents/:id/claim", handler.ClaimAgent)
	api.POST("/agents/:id/heartbeat", handler.AgentHeartbeat)
	api.POST("/agents/:id/tasks/pull", handler.PullAgentTask)
	api.POST("/agent-tasks/lease", handler.LeaseAgentTask)
	api.POST("/agent-tasks/:id/steps", handler.ReportAgentTaskStep)
	api.POST("/agent-tasks/:id/logs", handler.AppendAgentTaskLog)
	api.POST("/agent-tasks/:id/result", handler.ReportAgentTaskResult)

	api.POST("/baselines", handler.CreateBaseline)
	api.GET("/baselines", handler.ListBaselines)
	api.GET("/baselines/:id", handler.GetBaseline)
	api.POST("/baselines/:id/lock", handler.LockBaseline)
	api.POST("/baselines/:id/compare", handler.CompareBaseline)

	api.GET("/releases", handler.ListReleases)
	api.GET("/release-sources", handler.ListReleaseSources)
	api.POST("/releases", handler.CreateRelease)
	api.GET("/releases/:id", handler.GetRelease)
	api.POST("/releases/:id/retry", handler.RetryRelease)
	api.POST("/releases/:id/rollback", handler.RollbackRelease)

	api.GET("/deploy-tasks", handler.ListDeployTasks)
	api.POST("/deploy-tasks", handler.CreateDeployTask)
	api.GET("/deploy-tasks/:id", handler.GetDeployTask)
	api.POST("/deploy-tasks/:id/steps/:stepId/retry", handler.RetryDeployStep)
	api.POST("/deploy-tasks/:id/steps/:stepId/skip", handler.SkipDeployStep)
	api.POST("/deploy-tasks/:id/steps/:stepId/confirm", handler.ConfirmDeployStep)
	api.GET("/agent-tasks/:id/status", handler.GetAgentTaskStatus)

	return router
}

func BuildRepository(db *gorm.DB) repository.Store {
	mockRepo, err := repository.NewMockRepository()
	if err != nil {
		log.Fatalf("load mock repository: %v", err)
	}
	if db == nil {
		return mockRepo
	}
	return repository.NewDatabaseStore(db, mockRepo)
}
