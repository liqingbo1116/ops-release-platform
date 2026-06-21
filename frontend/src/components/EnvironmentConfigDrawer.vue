<template>
  <el-drawer v-model="visible" :title="environment ? `${environment.name} / 环境详情` : '环境详情'" size="420px">
    <div v-if="environment" class="drawer-stack">
      <div class="kv"><span>环境标识</span><strong>{{ environment.code }}</strong></div>
      <div class="kv"><span>环境类型</span><strong>{{ isLocalEnvironment ? '本地环境' : '远程环境' }}</strong></div>
      <div class="kv"><span>环境状态</span><StatusTag :status="environment.status" /></div>
      <template v-if="isLocalEnvironment">
        <BindingList title="K8s 命名空间" :items="k8sBindings" empty-text="未关联 K8s 命名空间" />
        <BindingList title="Harbor 镜像项目" :items="harborBindings" empty-text="未关联 Harbor 镜像项目" />
        <BindingList title="Jenkins 流水线视图" :items="jenkinsBindings" empty-text="未关联 Jenkins 流水线视图" />
        <div class="kv"><span>最近测试</span><span>{{ checkTimeText }}</span></div>
        <div class="check-help">{{ checkHelpText }}</div>
        <DiagnosticList :items="diagnostics" />
        <el-button type="primary" :loading="checking" @click="emit('check', environment.id)">执行连接测试</el-button>
      </template>
      <template v-else>
        <div class="kv"><span>Agent</span><StatusTag :status="environment.agentStatus" /></div>
        <div class="kv"><span>Agent 环境 ID</span><strong>{{ environment.id }}</strong></div>
        <BindingList title="Harbor 镜像项目" :items="harborBindings" empty-text="未关联 Harbor 镜像项目" />
        <BindingList title="Jenkins 流水线视图" :items="jenkinsBindings" empty-text="未关联 Jenkins 流水线视图" />
        <div class="kv"><span>最近上报</span><span>{{ checkTimeText }}</span></div>
        <div class="check-help">{{ checkHelpText }}</div>
        <DiagnosticList :items="diagnostics" />
        <el-button type="primary" :loading="checking" @click="emit('check', environment.id)">执行连接测试</el-button>
      </template>
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, type PropType } from 'vue'
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

type BindingListItem = {
  resourceName: string
  scopeValue: string
  isDefault: boolean
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
const k8sBindings = computed(() => bindingItems('K8S', props.environment?.clusterId, props.environment?.namespace))
const harborBindings = computed(() => bindingItems('HARBOR', props.environment?.registryId, props.environment?.registryProject))
const jenkinsBindings = computed(() => bindingItems('JENKINS', props.environment?.jenkinsId, props.environment?.jenkinsView))

function bindingItems(resourceType: EnvironmentResourceBinding['resourceType'], fallbackResourceId = '', fallbackScope = '') {
  const bindings = props.environment?.bindings?.filter((item) => item.resourceType === resourceType) ?? []
  const source = bindings.length > 0
    ? bindings
    : fallbackResourceId && fallbackScope
      ? [{ resourceType, resourceId: fallbackResourceId, scopeValue: fallbackScope, isDefault: true }]
      : []
  return source.map((item) => ({
    resourceName: props.resourceName?.(resourceType, item.resourceId) || item.resourceId,
    scopeValue: item.scopeValue,
    isDefault: item.isDefault,
  }))
}

const BindingList = defineComponent({
  name: 'BindingList',
  props: {
    title: { type: String, required: true },
    items: { type: Array as PropType<BindingListItem[]>, required: true },
    emptyText: { type: String, required: true },
  },
  setup(listProps) {
    return () => h('div', { class: 'binding-block' }, [
      h('div', { class: 'binding-title' }, listProps.title),
      listProps.items.length > 0
        ? h('div', { class: 'binding-list' }, listProps.items.map((item) => h('div', { class: 'binding-item' }, [
          h('strong', `${item.resourceName} / ${item.scopeValue}`),
          item.isDefault ? h('span', { class: 'default-badge' }, '默认') : null,
        ])))
        : h('div', { class: 'binding-empty' }, listProps.emptyText),
    ])
  },
})

const DiagnosticList = defineComponent({
  name: 'DiagnosticList',
  props: {
    items: { type: Array as PropType<EnvironmentDiagnostic[]>, required: true },
  },
  setup(listProps) {
    return () => h('div', { class: 'diagnostic-block' }, [
      h('div', { class: 'binding-title' }, '诊断结果'),
      listProps.items.length > 0
        ? h('div', { class: 'diagnostic-list' }, listProps.items.map((item) => h('div', { class: 'diagnostic-item' }, [
          h('div', { class: 'diagnostic-head' }, [
            h('strong', item.component),
            h('span', { class: `diagnostic-status diagnostic-status-${item.status.toLowerCase()}` }, statusText(item.status)),
          ]),
          h('p', item.message),
          h('small', `下一步：${item.nextStep}`),
        ])))
        : h('div', { class: 'binding-empty' }, '暂无诊断结果，请执行连接测试或刷新基础资源探测结果'),
    ])
  },
})

function statusText(status: EnvironmentDiagnostic['status']) {
  if (status === 'HEALTHY') return '正常'
  if (status === 'DEGRADED') return '需处理'
  return '待确认'
}
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

.diagnostic-block {
  border-top: 1px solid #edf0f5;
  padding-top: 12px;
}

.diagnostic-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.diagnostic-item {
  background: #fff;
  border: 1px solid #e7ebf2;
  border-radius: 6px;
  padding: 10px;
}

.diagnostic-head {
  align-items: center;
  display: flex;
  gap: 8px;
  justify-content: space-between;
}

.diagnostic-head strong {
  color: #2f3847;
  font-size: 13px;
}

.diagnostic-status {
  border-radius: 10px;
  flex: 0 0 auto;
  font-size: 12px;
  line-height: 20px;
  padding: 0 8px;
}

.diagnostic-status-healthy {
  background: #eef8f0;
  color: #1f8a4c;
}

.diagnostic-status-degraded {
  background: #fff4e5;
  color: #b76a00;
}

.diagnostic-status-unknown {
  background: #eef2f7;
  color: #606a7b;
}

.diagnostic-item p {
  color: #2f3847;
  font-size: 13px;
  line-height: 20px;
  margin: 8px 0 4px;
  overflow-wrap: anywhere;
}

.diagnostic-item small {
  color: #7a8294;
  display: block;
  font-size: 12px;
  line-height: 18px;
  overflow-wrap: anywhere;
}

.binding-block {
  border-top: 1px solid #edf0f5;
  padding-top: 12px;
}

.binding-title {
  color: #606a7b;
  font-size: 13px;
  margin-bottom: 8px;
}

.binding-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.binding-item {
  align-items: center;
  background: #f7f8fa;
  border-radius: 6px;
  display: flex;
  gap: 8px;
  justify-content: space-between;
  padding: 8px 10px;
}

.binding-item strong {
  font-size: 13px;
  overflow-wrap: anywhere;
}

.default-badge {
  background: #e8f3ff;
  border-radius: 10px;
  color: #1677ff;
  flex: 0 0 auto;
  font-size: 12px;
  line-height: 20px;
  padding: 0 8px;
}

.binding-empty {
  color: #9aa3b2;
  font-size: 13px;
}
</style>
