<template>
  <section class="terminal">
    <header class="terminal-head">
      <span>{{ title }}</span>
      <span>{{ badge }}</span>
    </header>
    <pre ref="bodyRef" class="terminal-body">{{ logs.join('\n') }}</pre>
  </section>
</template>

<script setup lang="ts">
import { nextTick, onMounted, ref, watch } from 'vue'

const props = withDefaults(
  defineProps<{
    title: string
    logs: string[]
    badge?: string
  }>(),
  {
    badge: 'live',
  },
)

const bodyRef = ref<HTMLElement | null>(null)

async function scrollToLatest() {
  await nextTick()
  const body = bodyRef.value
  if (!body) return
  body.scrollTop = body.scrollHeight
}

watch(
  () => props.logs.join('\n'),
  () => {
    void scrollToLatest()
  },
)

onMounted(() => {
  void scrollToLatest()
})
</script>
