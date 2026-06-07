<template>
  <main class="home-page">
    <el-card class="status-card" shadow="never">
      <template #header>
        <div class="card-header">
          <span>运维发布交付平台</span>
          <el-tag type="success">工程已初始化</el-tag>
        </div>
      </template>

      <el-descriptions :column="1" border>
        <el-descriptions-item label="前端">
          Vue 3 + Vite + TypeScript + Pinia + Vue Router + Element Plus
        </el-descriptions-item>
        <el-descriptions-item label="后端">
          Go + Gin，入口 backend/cmd/server
        </el-descriptions-item>
        <el-descriptions-item label="当前范围">
          仅初始化工程，不包含业务页面和真实第三方集成
        </el-descriptions-item>
      </el-descriptions>
    </el-card>
  </main>
</template>

<style scoped>
.home-page {
  display: grid;
  min-height: 100vh;
  place-items: center;
  padding: 24px;
}

.status-card {
  width: min(760px, 100%);
  border-radius: 8px;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  font-weight: 600;
}
</style>
