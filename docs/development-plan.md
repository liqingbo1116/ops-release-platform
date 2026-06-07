# 开发计划

## 阶段 1：前端原型工程化

目标：使用 Vue 3 还原 HTML 原型，接入 mock JSON。

交付：

- Vue 项目脚手架
- 页面路由
- Layout、导航、顶部栏
- 10 个核心页面
- 登录页、用户权限页、更新日志页
- mock 数据接入
- 基础筛选、勾选、抽屉、日志展示
- 路由守卫和 mock 登录态

## 阶段 2：Go 后端 mock API

目标：后端提供稳定 API，前端从 API 获取数据，并提供 mock 登录、权限和更新日志接口。

交付：

- Gin 服务
- REST API 路由
- mock repository
- 统一响应格式
- 基础错误码
- mock auth API
- 用户、角色、权限 mock API
- 更新日志 mock API

## 阶段 3：数据库与任务模型

目标：引入 PostgreSQL 和 Redis，任务状态可持久化。

交付：

- 数据库迁移
- GORM model
- 用户、角色、权限、更新日志、操作日志模型
- 发布单、部署任务 CRUD
- Redis Stream mock Agent worker

## 阶段 4：真实集成预留

目标：为 Jenkins、Harbor、K8s 接入预留 adapter。

交付：

- adapter interface
- mock adapter
- integration config
- 单元测试

## 暂不做

- 复杂审批流
- 灰度发布
- 完整离线交付
- 自动数据库结构 diff
- 完整 CMDB
- 真实 Nacos 配置写入
