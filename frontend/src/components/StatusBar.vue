<template>
  <footer class="statusbar">
    <span class="health" :class="healthClass">{{ healthIcon }} {{ healthText }}</span>
    <span class="spacer"></span>
    <span>📋 就绪</span>
    <button class="link-btn" @click="$emit('workspace')">📂 产物目录</button>
    <span v-if="lastRun !== null">执行时间: {{ lastRun }}</span>
  </footer>
</template>

<script setup>
import { computed } from 'vue'
const props = defineProps({
  checks: { type: Array, default: () => [] },
  lastRun: { type: String, default: null },
})
defineEmits(['workspace'])
const health = computed(() => {
  if (!props.checks.length) return { level: 'ok', icon: '🟢', text: '全部正常' }
  const errs = props.checks.filter((c) => c.status === 'error').length
  const warns = props.checks.filter((c) => c.status === 'warning').length
  if (errs > 0) return { level: 'error', icon: '🔴', text: '关键服务故障' }
  if (warns > 0) return { level: 'warning', icon: '🟡', text: '容器或 VM 未启动' }
  return { level: 'ok', icon: '🟢', text: '全部正常' }
})
const healthIcon = computed(() => health.value.icon)
const healthText = computed(() => health.value.text)
const healthClass = computed(() => 'health-' + health.value.level)
</script>
