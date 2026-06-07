# Integration Adapters

MVP 阶段仅保留第三方系统 adapter 目录结构，不实现真实 Jenkins、Harbor、Kubernetes、GitLab、ArgoCD、Nacos 集成。

后续真实接入时在对应目录内增加 interface 和 mock/real adapter 实现，并保持上层 service 调用契约稳定。
