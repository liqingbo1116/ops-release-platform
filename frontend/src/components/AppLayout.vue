<template>
  <div class="app-shell">
    <aside class="sidebar">
      <div class="brand">
        <div class="brand-mark">发</div>
        <div>
          <div class="brand-title">运维发布交付平台</div>
          <div class="brand-sub">Baseline Delivery Console</div>
        </div>
      </div>
      <SideNav />
    </aside>

    <section class="main-shell">
      <TopBar />
      <main class="content">
        <router-view />
      </main>
    </section>
  </div>
</template>

<script setup lang="ts">
import SideNav from './SideNav.vue'
import TopBar from './TopBar.vue'
</script>
