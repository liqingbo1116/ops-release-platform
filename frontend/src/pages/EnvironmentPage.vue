<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>环境管理</h1>
        <p>管理网络模式、K8s、Harbor、Nacos、MySQL、MinIO、凭证与连接测试。</p>
      </div>
      <el-button type="primary">新增环境</el-button>
    </div>

    <el-card shadow="never">
      <div class="toolbar">
        <div class="toolbar-left">
          <el-input v-model="keyword" placeholder="搜索环境、编码" clearable />
          <el-select v-model="networkMode" placeholder="全部网络模式" clearable>
            <el-option label="平台直连" value="DIRECT" />
            <el-option label="Agent 模式" value="AGENT" />
          </el-select>
        </div>
        <el-button>批量连接测试</el-button>
      </div>
      <el-table :data="filteredRows" class="wide-table">
        <el-table-column prop="name" label="环境" min-width="160" />
        <el-table-column prop="code" label="编码" min-width="160" />
        <el-table-column label="类型" min-width="110">
          <template #default="{ row }">{{ row.type === 'LOCAL' ? '本地环境' : '项目环境' }}</template>
        </el-table-column>
        <el-table-column label="网络模式" min-width="120">
          <template #default="{ row }">{{ row.networkMode === 'DIRECT' ? '平台直连' : 'Agent 模式' }}</template>
        </el-table-column>
        <el-table-column label="Agent" min-width="110">
          <template #default="{ row }"><StatusTag :status="row.agentStatus" /></template>
        </el-table-column>
        <el-table-column prop="lastCheckAt" label="最近测试" min-width="170">
          <template #default="{ row }">{{ formatDateTime(row.lastCheckAt) }}</template>
        </el-table-column>
        <el-table-column label="状态" min-width="100">
          <template #default="{ row }"><StatusTag :status="row.status" /></template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" width="120">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDrawer(row)">连接配置</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <EnvironmentConfigDrawer v-model:visible="drawerVisible" :environment="activeEnvironment" />
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import EnvironmentConfigDrawer from '@/components/EnvironmentConfigDrawer.vue'
import StatusTag from '@/components/StatusTag.vue'
import { mockData } from '@/api/mockData'
import { formatDateTime } from '@/utils/format'

type Environment = (typeof mockData.environments)[number]

const keyword = ref('')
const networkMode = ref('')
const drawerVisible = ref(false)
const activeEnvironment = ref<Environment | null>(null)

const filteredRows = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return mockData.environments.filter((item) => {
    const keywordMatched = !q || `${item.name} ${item.code}`.toLowerCase().includes(q)
    const modeMatched = !networkMode.value || item.networkMode === networkMode.value
    return keywordMatched && modeMatched
  })
})

function openDrawer(row: Environment) {
  activeEnvironment.value = row
  drawerVisible.value = true
}
</script>
