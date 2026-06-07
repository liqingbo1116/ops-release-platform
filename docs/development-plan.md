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

## 阶段 5：服务发版与服务部署闭环

目标：区分服务发版和服务部署，形成更贴近真实研发/项目环境的发布逻辑。

交付：

- 发布创建页支持选择“服务发版”和“服务部署”
- 服务发版不基于来源基线：
  - 方式 1：Jenkins Job 发版。选择与 Jenkins 视图或特征 job 关联后的 job，执行构建 jar/dist、制作镜像并推送到本地 Harbor
  - 方式 2：本地 Harbor 镜像发版。扫描本地 Harbor 上该服务的镜像版本，选择镜像 tag 发版
- 上述两种服务发版方式最终都需要通过项目环境中运行的 Agent 同步到项目环境，完成镜像同步和 tag 更新
- 服务部署基于基线/生产环境与目标环境对比，识别目标缺失服务
- 目标缺失服务在差异对比页明确标记为“服务部署”
- 服务部署提交后创建部署任务，并通过 Agent task status 展示执行状态和日志
- 发布/部署详情页根据 `agentTaskId` 轮询 Agent 日志；无 `agentTaskId` 时降级展示静态日志
- 预留真实项目环境 Agent 扫描服务列表并上报 RuntimeSnapshot 的后续实现

执行规则：

- 服务发版：目标环境已有服务。Jenkins Job 路径先调用 Jenkins 完成构建、镜像制作和推送本地 Harbor；本地 Harbor 镜像路径直接选择已有镜像 tag；两者最终都通过项目环境 Agent 完成项目 Harbor 镜像同步和 workload tag 更新
- 服务部署：目标环境缺失服务，从来源基线/生产环境确认服务清单后创建部署任务，将服务部署到目标环境
- 混合场景应拆分为 Jenkins 发版任务和服务部署任务，不把两类执行流混在一个任务里

## 暂不做

- 复杂审批流
- 灰度发布
- 完整离线交付
- 自动数据库结构 diff
- 完整 CMDB
- 真实 Nacos 配置写入
- 真实项目环境 Agent 扫描与上报
