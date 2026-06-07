<template>
  <nav class="nav">
    <template v-for="group in groups" :key="group.title">
      <div class="nav-group">{{ group.title }}</div>
      <router-link v-for="item in group.items" :key="item.path" :to="item.path" class="nav-link">
        <span class="nav-dot" />
        {{ item.label }}
      </router-link>
    </template>
  </nav>
</template>

<script setup lang="ts">
const groups = [
  { title: '工作台', items: [{ label: '首页工作台', path: '/dashboard' }] },
  {
    title: '环境与接入',
    items: [
      { label: '环境管理', path: '/environments' },
      { label: 'Agent 管理', path: '/agents' },
    ],
  },
  {
    title: '环境基线',
    items: [
      { label: '环境基线列表', path: '/baselines' },
      { label: '基线详情', path: '/baselines/BL-20260607-0001' },
      { label: '环境差异对比', path: '/compare' },
    ],
  },
  {
    title: '发布交付',
    items: [
      { label: '创建发布单', path: '/releases/create' },
      { label: '发布详情', path: '/releases/REL-20260607-031' },
    ],
  },
  {
    title: '部署脚本平台化',
    items: [
      { label: '部署任务列表', path: '/deploy-tasks' },
      { label: '部署任务详情', path: '/deploy-tasks/DEP-20260607-009' },
    ],
  },
]
</script>
