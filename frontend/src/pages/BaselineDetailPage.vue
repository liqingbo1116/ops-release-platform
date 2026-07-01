<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>基线详情：{{ detail.id }}</h1>
        <p>来源 {{ detail.sourceEnvironmentName }}，服务 {{ detail.serviceCount }} 个，{{ statusDescription }}</p>
      </div>
      <el-form inline class="top-actions">
        <el-form-item label="目标环境">
          <el-select v-model="targetEnvironmentId" placeholder="选择目标环境" style="width: 280px">
            <el-option
              v-for="environment in environments"
              :key="environment.id"
              :label="`${environment.name} / ${environment.code}`"
              :value="environment.id"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <div class="top-actions">
        <el-button :disabled="detail.status === 'LOCKED'" :loading="locking" @click="handleLock">
          {{ detail.status === 'LOCKED' ? '已锁定' : '锁定基线' }}
        </el-button>
        <el-button @click="goCompare">对比目标环境</el-button>
        <el-button type="primary" @click="goCreateRelease">基于此基线发布</el-button>
      </div>
    </div>

    <div class="metric-grid">
      <MetricCard label="服务数量" :value="detail.serviceCount" />
      <MetricCard label="健康服务" :value="healthyCount" foot="readyReplicas 正常" tone="good" />
      <MetricCard label="基线状态" :value="statusLabel" foot="可用于正式交付" tone="good" />
    </div>

    <el-card shadow="never" class="snapshot-card">
      <template #header>
        <div class="card-title">运行态快照</div>
      </template>
      <el-descriptions :column="2" border>
        <el-descriptions-item label="快照来源">{{ snapshotSource }}</el-descriptions-item>
        <el-descriptions-item label="采集时间">{{ snapshotCollectedAt }}</el-descriptions-item>
        <el-descriptions-item label="采集模式">{{ snapshotModeLabel }}</el-descriptions-item>
        <el-descriptions-item label="快照任务">{{ detail.snapshotTaskId || '待生成' }}</el-descriptions-item>
      </el-descriptions>
    </el-card>

    <el-card v-loading="loading" shadow="never">
      <el-table :data="detail.items" class="wide-table">
        <el-table-column prop="serviceName" label="服务" min-width="160" />
        <el-table-column prop="namespace" label="namespace" min-width="140" />
        <el-table-column prop="workloadName" label="workload" min-width="170" />
        <el-table-column prop="workloadType" label="类型" min-width="130" />
        <el-table-column prop="tag" label="镜像 tag" min-width="170" />
        <el-table-column prop="digest" label="digest" min-width="150" />
        <el-table-column label="副本" min-width="110">
          <template #default="{ row }">{{ row.readyReplicas }}/{{ row.replicas }}</template>
        </el-table-column>
        <el-table-column label="健康状态" min-width="110">
          <template #default="{ row }"><StatusTag :status="row.healthStatus" /></template>
        </el-table-column>
      </el-table>
    </el-card>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import MetricCard from '@/components/MetricCard.vue'
import StatusTag from '@/components/StatusTag.vue'
import { getBaselineDetail, lockBaseline, type BaselineDetailItem } from '@/api/baselines'
import { listEnvironments, type EnvironmentInfo } from '@/api/environments'
import { formatDateTime } from '@/utils/format'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const locking = ref(false)
const emptyBaselineDetail = (): BaselineDetailItem => ({
  id: '',
  name: '',
  sourceEnvironmentName: '',
  serviceCount: 0,
  createdBy: '',
  createdAt: '',
  status: 'UNKNOWN',
  purpose: '',
  items: [],
})
const detail = ref<BaselineDetailItem>(emptyBaselineDetail())
const environments = ref<EnvironmentInfo[]>([])
const targetEnvironmentId = ref(String(route.query.targetEnvironmentId || ''))
const healthyCount = computed(() => detail.value.items.filter((item) => item.healthStatus === 'HEALTHY').length)
const statusLabel = computed(() => detail.value.status === 'LOCKED' ? '已锁定' : detail.value.status)
const statusDescription = computed(() => detail.value.status === 'LOCKED' ? '已锁定，可用于项目环境差异发布。' : '当前为草稿，建议锁定后再用于正式交付。')
const snapshotSource = computed(() => detail.value.snapshotSource || detail.value.sourceEnvironmentName || '未知来源')
const snapshotCollectedAt = computed(() => detail.value.snapshotCollectedAt ? formatDateTime(detail.value.snapshotCollectedAt) : '未采集')
const snapshotModeLabel = computed(() => {
  if (detail.value.snapshotMode === 'AGENT_K8S') return 'Agent/Kubernetes 采集'
  return detail.value.snapshotMode || '未定义'
})

function syncTargetEnvironmentId() {
  const routeEnvironmentId = String(route.query.targetEnvironmentId || '')
  targetEnvironmentId.value = routeEnvironmentId || targetEnvironmentId.value || environments.value[0]?.id || ''
}

function goCompare() {
  if (!targetEnvironmentId.value) {
    ElMessage.warning('请先选择目标环境')
    return
  }
  router.push({
    path: '/compare',
    query: {
      baselineId: detail.value.id,
      targetEnvironmentId: targetEnvironmentId.value,
    },
  })
}

function goCreateRelease() {
  if (!targetEnvironmentId.value) {
    ElMessage.warning('请先选择目标环境')
    return
  }
  router.push({
    path: '/releases/create',
    query: {
      baselineId: detail.value.id,
      targetEnvironmentId: targetEnvironmentId.value,
    },
  })
}

async function loadEnvironments() {
  try {
    environments.value = await listEnvironments()
  } catch {
    ElMessage.error('加载环境列表失败')
    environments.value = []
  } finally {
    syncTargetEnvironmentId()
  }
}

async function loadDetail() {
  loading.value = true
  try {
    detail.value = await getBaselineDetail(String(route.params.id || detail.value.id))
  } catch {
    ElMessage.error('加载基线详情失败')
    detail.value = emptyBaselineDetail()
  } finally {
    loading.value = false
  }
}

async function handleLock() {
  if (detail.value.status === 'LOCKED') return
  locking.value = true
  try {
    detail.value = await lockBaseline(detail.value.id)
    ElMessage.success('基线已锁定')
  } catch {
    ElMessage.error('锁定基线失败')
  } finally {
    locking.value = false
  }
}

onMounted(async () => {
  await loadEnvironments()
  await loadDetail()
})

watch(() => route.fullPath, async () => {
  syncTargetEnvironmentId()
  await loadDetail()
})
</script>
