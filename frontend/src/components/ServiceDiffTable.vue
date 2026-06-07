<template>
  <el-table
    ref="tableRef"
    :data="items"
    row-key="serviceId"
    class="wide-table"
    @selection-change="handleSelectionChange"
  >
    <el-table-column type="selection" width="48" :selectable="isSelectable" />
    <el-table-column prop="serviceName" label="服务" min-width="160" />
    <el-table-column prop="namespace" label="namespace" min-width="140" />
    <el-table-column prop="sourceTag" label="来源 tag" min-width="170" />
    <el-table-column label="目标 tag" min-width="170">
      <template #default="{ row }">{{ row.targetTag ?? '未部署' }}</template>
    </el-table-column>
    <el-table-column label="差异状态" min-width="140">
      <template #default="{ row }"><StatusTag :status="row.diffStatus" /></template>
    </el-table-column>
    <el-table-column label="发布性" min-width="120">
      <template #default="{ row }">
        <el-tag :type="row.publishable ? 'success' : 'danger'" round>{{ row.publishable ? '可发布' : '不可发布' }}</el-tag>
      </template>
    </el-table-column>
    <el-table-column prop="strategy" label="处理策略" min-width="180" />
  </el-table>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import StatusTag from './StatusTag.vue'

type DiffItem = {
  serviceId: string
  serviceName: string
  namespace: string
  sourceTag: string
  targetTag: string | null
  diffStatus: string
  publishable: boolean
  strategy: string
}

const props = defineProps<{
  items: DiffItem[]
}>()

const selectedIds = defineModel<string[]>('selectedIds', { required: true })
const syncing = ref(false)
const tableRef = ref<{
  clearSelection: () => void
  toggleRowSelection: (row: DiffItem, selected?: boolean) => void
}>()

function handleSelectionChange(rows: DiffItem[]) {
  if (syncing.value) return
  selectedIds.value = rows.map((row) => row.serviceId)
}

function isSelectable(row: DiffItem) {
  return row.publishable
}

watch(
  [() => selectedIds.value, () => props.items],
  async () => {
    await nextTick()
    syncing.value = true
    tableRef.value?.clearSelection()
    props.items.forEach((item) => {
      if (selectedIds.value.includes(item.serviceId) && item.publishable) {
        tableRef.value?.toggleRowSelection(item, true)
      }
    })
    await nextTick()
    syncing.value = false
  },
  { immediate: true },
)
</script>
