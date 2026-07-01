<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>更新日志</h1>
        <p>记录平台每个小版本上线后的迭代内容、修复问题和已知问题。</p>
      </div>
      <PermissionButton permission="changelog:write">新增版本记录</PermissionButton>
    </div>

    <el-card shadow="never">
      <div class="toolbar">
        <div class="toolbar-left">
          <el-input v-model="keyword" placeholder="搜索版本、标题、内容" clearable />
          <el-select v-model="typeFilter" placeholder="全部类型" clearable>
            <el-option label="功能" value="FEATURE" />
            <el-option label="修复" value="FIX" />
            <el-option label="优化" value="IMPROVEMENT" />
            <el-option label="安全" value="SECURITY" />
          </el-select>
        </div>
      </div>
      <el-alert v-if="errorMessage" :title="errorMessage" type="error" show-icon :closable="false" />
      <ChangelogTimeline v-loading="loading" :items="filteredItems" />
    </el-card>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { listChangelog } from '@/api/changelog'
import ChangelogTimeline from '@/components/ChangelogTimeline.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import { useChangelogStore } from '@/stores/changelogStore'

const changelogStore = useChangelogStore()
const keyword = ref('')
const typeFilter = ref('')
const loading = ref(false)
const errorMessage = ref('')

const filteredItems = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return changelogStore.items.filter((item) => {
    const typeMatched = !typeFilter.value || item.type === typeFilter.value
    const keywordMatched =
      !q ||
      `${item.title} ${item.description ?? ''} ${item.author ?? ''} ${(item.tags ?? []).join(' ')}`
        .toLowerCase()
        .includes(q)
    return typeMatched && keywordMatched
  })
})

async function loadData() {
  loading.value = true
  errorMessage.value = ''
  try {
    changelogStore.items = await listChangelog()
  } catch (error) {
    changelogStore.items = []
    errorMessage.value = error instanceof Error ? error.message : '更新日志加载失败'
    ElMessage.error(errorMessage.value)
  } finally {
    loading.value = false
  }
}

onMounted(loadData)
</script>
