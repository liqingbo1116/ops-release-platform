<template>
  <el-drawer v-model="visible" :title="environment ? `${environment.name} / 连接配置` : '连接配置'" size="420px">
    <div v-if="environment" class="drawer-stack">
      <div class="kv"><span>环境编码</span><strong>{{ environment.code }}</strong></div>
      <div class="kv"><span>环境类型</span><strong>{{ environment.type === 'PROJECT' ? '项目环境' : '本地环境' }}</strong></div>
      <div class="kv"><span>网络模式</span><strong>{{ environment.networkMode === 'AGENT' ? 'Agent 模式' : '平台直连' }}</strong></div>
      <div class="kv"><span>K8s 集群</span><strong>{{ environment.clusterId || defaultIntegrationId }}</strong></div>
      <div class="kv"><span>命名空间</span><strong>{{ environment.namespace || '-' }}</strong></div>
      <div class="kv"><span>镜像仓库</span><strong>{{ environment.registryId || defaultIntegrationId }}</strong></div>
      <div class="kv"><span>镜像项目</span><strong>{{ environment.registryProject || '-' }}</strong></div>
      <div class="kv"><span>Jenkins</span><strong>{{ environment.jenkinsId || '-' }}</strong></div>
      <div class="kv"><span>Jenkins 视图</span><strong>{{ environment.jenkinsView || '-' }}</strong></div>
      <div class="kv"><span>Agent</span><StatusTag :status="environment.agentStatus" /></div>
      <div class="kv"><span>最近测试</span><span>{{ environment.lastCheckAt || '-' }}</span></div>
      <el-button type="primary" :loading="checking" @click="emit('check', environment.id)">执行连接测试</el-button>
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import StatusTag from './StatusTag.vue'

type Environment = {
  id: string
  name: string
  code: string
  type: string
  networkMode: string
  clusterId: string
  namespace: string
  registryId: string
  registryProject: string
  jenkinsId: string
  jenkinsView: string
  agentStatus: string
  lastCheckAt: string
}

const visible = defineModel<boolean>('visible', { required: true })
const emit = defineEmits<{
  check: [id: string]
}>()
defineProps<{
  environment: Environment | null
  checking?: boolean
}>()

const defaultIntegrationId = '未关联资源'
</script>
