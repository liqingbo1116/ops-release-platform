<template>
  <section class="terminal">
    <header class="terminal-head">
      <span>{{ title }}</span>
      <span>{{ badge }}</span>
    </header>
    <pre class="terminal-body">{{ logs.join('\n') }}</pre>
  </section>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    title: string
    logs: string[]
    badge?: string
  }>(),
  {
    badge: 'live',
  },
)
</script>
