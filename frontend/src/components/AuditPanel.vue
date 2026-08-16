<template>
  <div class="dialog-mask" @click.self="$emit('close')">
    <div class="dialog audit">
      <h3>🩺 环境健康审计</h3>
      <ul class="audit-list">
        <li v-for="c in checks" :key="c.name">
          <span class="dot" :class="dotClass(c.status)"></span>
          <span class="check-name">{{ c.name }}</span>
          <span class="check-detail">{{ c.detail }}</span>
        </li>
      </ul>
      <div class="dialog-actions">
        <button class="btn-ghost" @click="$emit('refresh')">🔄 重新检查</button>
        <button class="btn-primary" @click="$emit('close')">关闭</button>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({ checks: { type: Array, required: true } })
defineEmits(['close', 'refresh'])
const dotClass = (s) => ({ ok: 'dot-ok', warning: 'dot-warn', error: 'dot-err' }[s] || 'dot-err')
</script>
