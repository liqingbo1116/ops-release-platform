<template>
  <el-drawer v-model="visible" :title="environment ? `${environment.name} / 环境详情` : '环境详情'" size="380px">
    <div v-if="environment" class="drawer-stack">
      <div class="kv"><span>环境标识</span><strong>{{ environment.code }}</strong></div>
      <div class="kv"><span>环境类型</span><strong>{{ isLocalEnvironment ? '本地环境' : '远程环境' }}</strong></div>
      <div class="kv"><span>环境状态</span><StatusTag :status="environment.status" /></div>
      <template v-if="!isLocalEnvironment">
        <div class="kv"><span>Agent</span><StatusTag :status="environment.agentStatus" /></div>
        <div class="kv"><span>Agent 环境 ID</span><strong>{{ environment.id }}</strong></div>
        <div v-if="agentProblemText" class="check-help">{{ agentProblemText }}</div>
      </template>
      <div class="kv"><span>{{ isLocalEnvironment ? '最近测试' : '最近上报' }}</span><span>{{ checkTimeText }}</span></div>
      <div class="check-help">{{ checkHelpText }}</div>
      <div class="summary-block">
        <div class="summary-title">未就绪项</div>
        <div v-if="problemDiagnostics.length > 0" class="summary-list">
          <div v-for="item in problemDiagnostics" :key="`${item.component}-${item.message}`" class="summary-item">
            <strong>{{ item.message }}</strong>
            <span>{{ item.nextStep }}</span>
          </div>
        </div>
        <div v-else class="summary-empty">当前未发现问题</div>
      </div>
      <el-button type="primary" :loading="checking" @click="emit('check', environment.id)">执行连接测试</el-button>
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import StatusTag from './StatusTag.vue'
import { formatDateTime } from '@/utils/format'
import type { EnvironmentResourceBinding } from '@/api/environments'

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
  status: string
  agentStatus: string
  lastCheckAt: string
  bindings?: EnvironmentResourceBinding[]
}

type EnvironmentDiagnostic = {
  component: string
  status: 'HEALTHY' | 'DEGRADED' | 'UNKNOWN'
  message: string
  nextStep: string
}

const visible = defineModel<boolean>('visible', { required: true })
const emit = defineEmits<{
  check: [id: string]
}>()
const props = defineProps<{
  environment: Environment | null
  resourceName?: (resourceType: EnvironmentResourceBinding['resourceType'], resourceId: string) => string
  checking?: boolean
  diagnostics: EnvironmentDiagnostic[]
  checkHelpText?: string
}>()

const isLocalEnvironment = computed(() => props.environment?.type === 'LOCAL')
const checkTimeText = computed(() => (props.environment?.lastCheckAt ? formatDateTime(props.environment.lastCheckAt) : '-'))
const problemDiagnostics = computed(() => props.diagnostics.filter((item) => item.status !== 'HEALTHY'))
const agentProblemText = computed(() => {
  if (!props.environment || isLocalEnvironment.value || props.environment.agentStatus === 'ONLINE') return ''
  return 'Agent 未在线会影响远程发布/部署执行；资源范围验证通过时，Jenkins/Harbor 仍可作为后续服务关联范围使用。'
})
</script>

<style scoped>
.check-help {
  background: #f6f8fb;
  border: 1px solid #e7ebf2;
  border-radius: 6px;
  color: #5f6878;
  font-size: 13px;
  line-height: 20px;
  padding: 10px 12px;
}

.summary-block {
  border-top: 1px solid #edf0f5;
  padding-top: 12px;
}

.summary-title {
  color: #606a7b;
  font-size: 13px;
  margin-bottom: 8px;
}

.summary-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.summary-item {
  background: #fff;
  border: 1px solid #e7ebf2;
  border-radius: 6px;
  padding: 10px;
}

.summary-item strong,
.summary-item span {
  display: block;
  font-size: 13px;
  line-height: 20px;
  overflow-wrap: anywhere;
}

.summary-item strong {
  color: #2f3847;
}

.summary-item span {
  color: #7a8294;
  margin-top: 4px;
}

.summary-empty {
  color: #9aa3b2;
  font-size: 13px;
}

.drawer-stack {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.kv {
  align-items: center;
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

.kv span {
  color: #606a7b;
  font-size: 13px;
}

.kv strong {
  color: #2f3847;
  font-size: 12px;
  overflow-wrap: anywhere;
  text-align: right;
}
</style>
