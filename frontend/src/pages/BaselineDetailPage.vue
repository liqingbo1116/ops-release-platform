<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>基线详情：{{ detail.id }}</h1>
        <p>来源 {{ detail.sourceEnvironmentName }}，服务 {{ detail.serviceCount }} 个，已锁定，可用于项目环境差异发布。</p>
      </div>
      <div class="top-actions">
        <el-button @click="$router.push('/compare')">对比目标环境</el-button>
        <el-button type="primary" @click="$router.push('/releases/create')">基于此基线发布</el-button>
      </div>
    </div>

    <div class="metric-grid">
      <MetricCard label="服务数量" :value="detail.serviceCount" />
      <MetricCard label="健康服务" :value="healthyCount" foot="readyReplicas 正常" tone="good" />
      <MetricCard label="基线状态" value="已锁定" foot="可用于正式交付" tone="good" />
    </div>

    <el-card shadow="never">
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
import MetricCard from '@/components/MetricCard.vue'
import StatusTag from '@/components/StatusTag.vue'
import { mockData } from '@/api/mockData'

const detail = mockData.baselineDetail
const healthyCount = detail.items.filter((item) => item.healthStatus === 'HEALTHY').length
</script>
