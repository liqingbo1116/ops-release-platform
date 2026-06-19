package repository

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectAndMigrate(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := AutoMigrate(db); err != nil {
		return nil, err
	}
	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&ProductModel{},
		&ServiceModel{},
		&EnvironmentModel{},
		&EnvironmentResourceBindingModel{},
		&KubernetesClusterModel{},
		&HarborRegistryModel{},
		&JenkinsInstanceModel{},
		&AgentModel{},
		&EnvironmentBaselineModel{},
		&BaselineServiceItemModel{},
		&ReleaseOrderModel{},
		&DeployTaskModel{},
		&DeployStepModel{},
		&AgentTaskModel{},
		&AgentTaskLogModel{},
		&UserModel{},
		&RoleModel{},
		&UserRoleModel{},
		&EnvironmentPermissionModel{},
		&ChangelogModel{},
		&OperationLogModel{},
	)
}
