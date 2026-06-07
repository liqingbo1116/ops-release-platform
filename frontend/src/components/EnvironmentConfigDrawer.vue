<template>
  <el-drawer v-model="visible" :title="environment ? `${environment.name} / 连接配置` : '连接配置'" size="420px">
    <div v-if="environment" class="drawer-stack">
      <div class="kv"><span>环境编码</span><strong>{{ environment.code }}</strong></div>
      <div class="kv"><span>网络模式</span><strong>{{ environment.networkMode === 'AGENT' ? 'Agent 模式' : '平台直连' }}</strong></div>
      <div class="kv"><span>K8s 集群</span><span class="mono">https://10.12.8.21:6443</span></div>
      <div class="kv"><span>Harbor</span><span class="mono">harbor.project.local</span></div>
      <div class="kv"><span>Nacos</span><span class="mono">nacos.project.local:8848</span></div>
      <div class="kv"><span>Agent</span><StatusTag :status="environment.agentStatus" /></div>
      <el-button type="primary">执行连接测试</el-button>
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import StatusTag from './StatusTag.vue'

type Environment = {
  name: string
  code: string
  networkMode: string
  agentStatus: string
}

const visible = defineModel<boolean>('visible', { required: true })
defineProps<{
  environment: Environment | null
}>()
</script>
