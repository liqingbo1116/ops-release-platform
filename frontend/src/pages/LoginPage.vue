<template>
  <main class="login-page">
    <section class="login-panel">
      <div>
        <div class="brand-row">
          <div class="brand-mark">发</div>
          <div>
            <h1>运维发布交付平台</h1>
            <p>Baseline Delivery Console</p>
          </div>
        </div>
      </div>

      <el-form label-position="top" @submit.prevent="submit">
        <el-form-item label="用户名">
          <el-input v-model="form.username" autocomplete="username" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="form.password" type="password" autocomplete="current-password" show-password />
        </el-form-item>
        <el-button type="primary" class="login-button" :loading="loading" @click="submit">登录</el-button>
      </el-form>
    </section>
  </main>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/authStore'

const router = useRouter()
const authStore = useAuthStore()
const loading = ref(false)
const form = reactive({
  username: '',
  password: '',
})

async function submit() {
  loading.value = true
  try {
    await authStore.login(form.username, form.password)
    await router.push('/dashboard')
  } finally {
    loading.value = false
  }
}
</script>
