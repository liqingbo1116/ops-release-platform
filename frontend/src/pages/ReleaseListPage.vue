<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>发版记录</h1>
        <p>查看服务发版结果、执行状态和日志；新发版请从服务管理进入。</p>
      </div>
    </div>

    <el-card shadow="never">
      <div class="toolbar">
        <el-input v-model="keyword" placeholder="搜索记录、服务、产品、Agent" clearable />
      </div>
      <el-table v-loading="loading" :data="filteredRows" class="wide-table">
        <el-table-column prop="id" label="记录" min-width="150" />
        <el-table-column label="服务" min-width="220">
          <template #default="{ row }">
            <div class="service-cell">
              <strong>{{ serviceLabel(row) }}</strong>
              <span>{{ serviceFoot(row) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="targetEnvironmentName" label="产品" min-width="150" />
        <el-table-column prop="agentName" label="执行 Agent" min-width="170" />
        <el-table-column label="状态" min-width="120">
          <template #default="{ row }"><StatusTag :status="row.status" /></template>
        </el-table-column>
        <el-table-column label="目标镜像" min-width="240">
          <template #default="{ row }">{{ imageLabel(row) }}</template>
        </el-table-column>
        <el-table-column label="执行" min-width="160">
          <template #default="{ row }">{{ executionLabel(row) }}</template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" width="110">
          <template #default="{ row }"><el-button link type="primary" @click="$router.push(`/releases/${row.id}`)">查看日志</el-button></template>
        </el-table-column>
      </el-table>
    </el-card>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, ref } from 'vue'
import { listReleases, type ReleaseOrder } from '@/api/releases'
import StatusTag from '@/components/StatusTag.vue'

const keyword = ref('')
const loading = ref(false)
const rows = ref<ReleaseOrder[]>([])

const filteredRows = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return rows.value
  return rows.value.filter((item) =>
    `${item.id} ${serviceLabel(item)} ${serviceFoot(item)} ${item.targetEnvironmentName} ${item.agentName} ${imageLabel(item)} ${executionLabel(item)}`
      .toLowerCase()
      .includes(q),
  )
})

function serviceLabel(item: ReleaseOrder) {
  const names = item.serviceNames?.filter(Boolean) ?? []
  return names.length > 0 ? names.join('、') : '未记录服务'
}

function serviceFoot(item: ReleaseOrder) {
  if (item.releaseSource === 'LOCAL_HARBOR_IMAGE') return '镜像发版'
  if (item.releaseSource === 'JENKINS_JOB') return 'Jenkins 发版'
  if (item.type === 'SERVICE_DEPLOYMENT') return '服务部署'
  return '服务发版'
}

function imageLabel(item: ReleaseOrder) {
  if (item.imageRepository && item.imageTag) return `${item.imageRepository}:${item.imageTag}`
  if (item.imageRepository || item.imageTag) return item.imageRepository || item.imageTag
  return '等待环境确认'
}

function executionLabel(item: ReleaseOrder) {
  if (item.buildId) return `Jenkins #${item.buildId}`
  if (item.buildStatus) return item.buildStatus
  return item.agentName || '待生成任务'
}

async function loadRows() {
  loading.value = true
  try {
    rows.value = await listReleases()
  } catch {
    ElMessage.error('加载发布单失败')
    rows.value = []
  } finally {
    loading.value = false
  }
}

onMounted(loadRows)
</script>

<style scoped>
.service-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.service-cell span {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
</style>
