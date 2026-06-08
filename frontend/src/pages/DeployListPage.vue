<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>部署任务列表</h1>
        <p>将原有 shell 脚本包装为平台化步骤，统一参数、顺序、日志、重试、跳过与人工确认。</p>
      </div>
      <el-button type="primary" @click="$router.push('/deploy-tasks/DEP-20260607-MOCK')">查看部署详情</el-button>
    </div>

    <el-card shadow="never">
      <div class="toolbar">
        <el-input v-model="keyword" placeholder="搜索部署任务、产品、环境" clearable />
        <el-button>批量重试失败步骤</el-button>
      </div>
      <el-table v-loading="loading" :data="filteredRows" class="wide-table">
        <el-table-column prop="id" label="任务" min-width="160" />
        <el-table-column prop="productName" label="产品" min-width="110" />
        <el-table-column prop="targetEnvironmentName" label="目标环境" min-width="160" />
        <el-table-column prop="source" label="来源" min-width="150" />
        <el-table-column prop="currentStep" label="当前步骤" min-width="150" />
        <el-table-column label="进度" min-width="160">
          <template #default="{ row }"><el-progress :percentage="row.progress" :status="row.status === 'FAILED' ? 'exception' : undefined" /></template>
        </el-table-column>
        <el-table-column label="状态" min-width="110">
          <template #default="{ row }"><StatusTag :status="row.status" /></template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" width="100">
          <template #default="{ row }"><el-button link type="primary" @click="$router.push(`/deploy-tasks/${row.id}`)">查看</el-button></template>
        </el-table-column>
      </el-table>
    </el-card>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, ref } from 'vue'
import StatusTag from '@/components/StatusTag.vue'
import { listDeployTasks } from '@/api/deployTasks'
import { deployMockData } from '@/api/mockData/deploy'

const keyword = ref('')
const loading = ref(false)
const rows = ref([...deployMockData.deployTasks])
const filteredRows = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return rows.value
  return rows.value.filter((item) =>
    `${item.id} ${item.productName} ${item.targetEnvironmentName} ${item.source}`.toLowerCase().includes(q),
  )
})

async function loadRows() {
  loading.value = true
  try {
    rows.value = await listDeployTasks()
  } catch {
    ElMessage.warning('加载部署任务失败，已显示本地示例数据')
    rows.value = [...deployMockData.deployTasks]
  } finally {
    loading.value = false
  }
}

onMounted(loadRows)
</script>
