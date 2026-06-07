<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>部署任务详情：{{ deploy.id }}</h1>
        <p>{{ deploy.targetEnvironmentName }} 初始化部署，来源 {{ deploy.source }}，当前进度 {{ deploy.progress }}%。</p>
      </div>
      <div class="top-actions">
        <el-button>暂停</el-button>
        <el-button>跳过当前步骤</el-button>
        <el-button type="danger">重试失败步骤</el-button>
      </div>
    </div>

    <div class="metric-grid six">
      <MetricCard label="整体进度" :value="`${deploy.progress}%`" foot="5/13 步完成" />
      <MetricCard label="执行步骤" :value="deploy.steps.length" foot="含人工确认" />
      <MetricCard label="脚本步骤" :value="scriptStepCount" />
      <MetricCard label="失败次数" value="1" foot="MinIO 重试中" tone="bad" />
      <MetricCard label="当前 Agent" value="在线" tone="good" />
      <MetricCard label="耗时" value="28:14" />
    </div>

    <div class="two-col">
      <DeployStepPanel title="部署步骤编排" :status="deploy.status" :steps="deploy.steps" />
      <el-card shadow="never">
        <template #header><div class="panel-head"><strong>当前步骤日志</strong><el-button type="danger" link>重试 MinIO</el-button></div></template>
        <LogTerminal :title="`${deploy.id} / restore-minio`" :logs="deploy.logs" badge="retry #1" />
      </el-card>
    </div>
  </section>
</template>

<script setup lang="ts">
import DeployStepPanel from '@/components/DeployStepPanel.vue'
import LogTerminal from '@/components/LogTerminal.vue'
import MetricCard from '@/components/MetricCard.vue'
import { mockData } from '@/api/mockData'

const deploy = mockData.deployDetail
const scriptStepCount = deploy.steps.filter((item) => ['SHELL', 'SQL'].includes(item.type)).length
</script>
