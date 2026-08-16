<template>
  <div class="card">
    <div class="card-head">
      <span class="tool-icon">{{ tool.icon }}</span>
      <span class="tool-name">{{ tool.name }}</span>
      <span class="env-badge" :class="envClass">{{ envLabel }}</span>
    </div>
    <p class="tool-desc">{{ tool.description }}</p>
    <div class="card-foot">
      <span class="use-count">已运行 {{ tool.use_count }} 次</span>
      <button class="run-btn" @click="$emit('run', tool)">▶ 运行</button>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
const props = defineProps({ tool: { type: Object, required: true } })
defineEmits(['run'])
const envLabel = computed(() => {
  if (props.tool.is_high_risk) return '🔴 VM高危'
  switch (props.tool.default_env) {
    case 'local': return '🟢 本地'
    case 'podman': return '🟡 容器'
    case 'vm': return '🔴 VM'
    default: return '⚪ 未知'
  }
})
const envClass = computed(() => {
  if (props.tool.is_high_risk) return 'badge-vm'
  switch (props.tool.default_env) {
    case 'local': return 'badge-local'
    case 'podman': return 'badge-podman'
    default: return 'badge-vm'
  }
})
</script>
