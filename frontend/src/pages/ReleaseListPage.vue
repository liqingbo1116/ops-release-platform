<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>发布单列表</h1>
        <p>统一查看服务发版与服务部署任务，按发版来源、部署基线、目标环境和执行状态快速检索。</p>
      </div>
      <el-button type="primary" @click="$router.push('/releases/create')">新建发布单</el-button>
    </div>

    <el-card shadow="never">
      <div class="toolbar">
        <el-input v-model="keyword" placeholder="搜索发布单、发版来源、基线、目标环境、Agent" clearable />
        <el-button @click="$router.push('/releases/REL-20260607-031')">查看示例详情</el-button>
      </div>
      <el-table v-loading="loading" :data="filteredRows" class="wide-table">
        <el-table-column prop="id" label="发布单" min-width="160" />
        <el-table-column prop="type" label="类型" min-width="150" />
        <el-table-column label="来源" min-width="220">
          <template #default="{ row }">
            <div class="source-cell">
              <strong>{{ sourceLabel(row) }}</strong>
              <span>{{ sourceFoot(row) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="targetEnvironmentName" label="目标环境" min-width="150" />
        <el-table-column prop="agentName" label="执行 Agent" min-width="170" />
        <el-table-column label="进度" min-width="160">
          <template #default="{ row }">
            <el-progress :percentage="row.progress" :status="row.status === 'PARTIAL_FAILED' ? 'exception' : undefined" />
          </template>
        </el-table-column>
        <el-table-column label="状态" min-width="120">
          <template #default="{ row }"><StatusTag :status="row.status" /></template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" width="100">
          <template #default="{ row }"><el-button link type="primary" @click="$router.push(`/releases/${row.id}`)">查看</el-button></template>
        </el-table-column>
      </el-table>
    </el-card>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, ref } from 'vue'
import { listReleases } from '@/api/releases'
import { releaseMockData } from '@/api/mockData/release'
import StatusTag from '@/components/StatusTag.vue'

const keyword = ref('')
const loading = ref(false)
const rows = ref([...releaseMockData.releases])

const filteredRows = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return rows.value
  return rows.value.filter((item) =>
    `${item.id} ${item.type} ${sourceLabel(item)} ${sourceFoot(item)} ${item.targetEnvironmentName} ${item.agentName}`
      .toLowerCase()
      .includes(q),
  )
})

function sourceLabel(item: typeof releaseMockData.releases[number]) {
  if (item.type === 'SERVICE_DEPLOYMENT') return item.sourceBaselineId || '来源基线'
  if (item.releaseSource === 'LOCAL_HARBOR_IMAGE') return '本地 Harbor 镜像'
  if (item.releaseSource === 'JENKINS_JOB') return 'Jenkins Job'
  return '服务发版'
}

function sourceFoot(item: typeof releaseMockData.releases[number]) {
  if (item.type === 'SERVICE_DEPLOYMENT') return '缺失服务首次部署'
  if (item.releaseSource === 'LOCAL_HARBOR_IMAGE') return `${item.imageRepository || '镜像仓库'}:${item.imageTag || '镜像 tag'}`
  return item.buildId || '等待构建任务'
}

async function loadRows() {
  loading.value = true
  try {
    rows.value = await listReleases()
  } catch {
    ElMessage.warning('加载发布单失败，已显示本地示例数据')
    rows.value = [...releaseMockData.releases]
  } finally {
    loading.value = false
  }
}

onMounted(loadRows)
</script>

<style scoped>
.source-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.source-cell span {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
</style>
