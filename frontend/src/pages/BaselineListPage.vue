<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>环境基线列表</h1>
        <p>从真实运行态采集生成交付基线，替代难以维护的传统产品版本。</p>
      </div>
      <el-button type="primary">从运行环境生成基线</el-button>
    </div>

    <el-card shadow="never">
      <div class="toolbar">
        <el-input v-model="keyword" placeholder="搜索基线 ID、来源环境、用途" clearable />
        <div class="top-actions">
          <el-button>导出清单</el-button>
          <el-button>批量锁定</el-button>
        </div>
      </div>
      <el-table :data="filteredRows" class="wide-table">
        <el-table-column type="selection" width="48" />
        <el-table-column prop="id" label="基线 ID" min-width="170" />
        <el-table-column prop="name" label="基线名称" min-width="220" />
        <el-table-column prop="sourceEnvironmentName" label="来源环境" min-width="150" />
        <el-table-column prop="serviceCount" label="服务数" min-width="90" />
        <el-table-column prop="createdBy" label="创建人" min-width="90" />
        <el-table-column label="创建时间" min-width="170">
          <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
        </el-table-column>
        <el-table-column label="状态" min-width="100">
          <template #default="{ row }"><StatusTag :status="row.status" /></template>
        </el-table-column>
        <el-table-column prop="purpose" label="用途" min-width="140" />
        <el-table-column label="操作" fixed="right" width="100">
          <template #default="{ row }">
            <el-button link type="primary" @click="$router.push(`/baselines/${row.id}`)">查看</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import StatusTag from '@/components/StatusTag.vue'
import { mockData } from '@/api/mockData'
import { formatDateTime } from '@/utils/format'

const keyword = ref('')
const filteredRows = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return mockData.baselines
  return mockData.baselines.filter((item) =>
    `${item.id} ${item.name} ${item.sourceEnvironmentName} ${item.purpose}`.toLowerCase().includes(q),
  )
})
</script>
