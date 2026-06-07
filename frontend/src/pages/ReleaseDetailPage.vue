<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>发布详情：{{ release.id }}</h1>
        <p>基线差异发布，{{ release.sourceBaselineId }} 到 {{ release.targetEnvironmentName }}。</p>
      </div>
      <div class="top-actions">
        <el-button>暂停</el-button>
        <el-button type="danger">失败重试</el-button>
        <el-button type="primary">生成发布报告</el-button>
      </div>
    </div>

    <div class="metric-grid six">
      <MetricCard label="整体进度" :value="`${release.progress}%`" foot="正在健康检查" />
      <MetricCard label="镜像同步" value="68/68" foot="全部完成" tone="good" />
      <MetricCard label="更新 tag" value="66/68" foot="2 个等待 rollout" tone="warn" />
      <MetricCard label="健康检查" value="61/68" />
      <MetricCard label="失败服务" :value="release.failures.length" foot="可重试" tone="bad" />
      <MetricCard label="执行 Agent" value="在线" :foot="release.agentName" tone="good" />
    </div>

    <div class="two-col">
      <DeployStepPanel title="发布步骤" :status="release.status" :steps="release.steps" />
      <el-card shadow="never">
        <template #header><div class="panel-head"><strong>Agent 执行日志</strong><el-button link type="primary">复制日志</el-button></div></template>
        <LogTerminal :title="`${release.agentName} / ${release.id}`" :logs="release.logs" />
      </el-card>
    </div>

    <el-card shadow="never">
      <template #header><strong>失败定位建议</strong></template>
      <el-table :data="release.failures" class="wide-table">
        <el-table-column prop="serviceName" label="服务" min-width="150" />
        <el-table-column prop="reason" label="失败原因" min-width="160" />
        <el-table-column prop="suggestion" label="建议处理动作" min-width="320" />
        <el-table-column label="操作" fixed="right" width="120">
          <template #default="{ row }"><el-button link type="primary" @click="openFailure(row)">详情</el-button></template>
        </el-table-column>
      </el-table>
    </el-card>

    <ServiceFailureDrawer v-model:visible="drawerVisible" :failure="activeFailure" />
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import DeployStepPanel from '@/components/DeployStepPanel.vue'
import LogTerminal from '@/components/LogTerminal.vue'
import MetricCard from '@/components/MetricCard.vue'
import ServiceFailureDrawer from '@/components/ServiceFailureDrawer.vue'
import { mockData } from '@/api/mockData'

type Failure = (typeof mockData.releaseDetail.failures)[number]

const release = mockData.releaseDetail
const drawerVisible = ref(false)
const activeFailure = ref<Failure | null>(null)

function openFailure(row: Failure) {
  activeFailure.value = row
  drawerVisible.value = true
}
</script>
