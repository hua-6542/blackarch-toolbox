<template>
  <div v-if="exec" class="log-panel">
    <div class="log-head">
      <span>📋 执行日志 — {{ exec.tool }}（{{ envName(exec.env_used) }}）</span>
      <span v-if="finished" class="exit" :class="exitClass">
        退出码: {{ exitCode }}
      </span>
      <button class="btn-ghost" @click="close">关闭</button>
    </div>
    <div ref="body" class="log-body">
      <div v-for="(l, i) in lines" :key="i" class="log-line">{{ l }}</div>
      <div v-if="!finished" class="log-line dim">运行中…</div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onUnmounted } from 'vue'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

const props = defineProps({ exec: { type: Object, default: null } })
const emit = defineEmits(['close'])
const lines = ref([])
const finished = ref(false)
const exitCode = ref(null)
const body = ref(null)
let boundId = null

const envName = (e) => ({ local: '本地', podman: '容器', vm: '虚拟机' }[e] || e)
const exitClass = computed(() => (exitCode.value === 0 ? 'exit-ok' : 'exit-bad'))

async function bind() {
  if (!props.exec) return
  if (boundId !== null) unbind()
  boundId = props.exec.execution_id
  finished.value = false
  exitCode.value = null
  lines.value = []
  const id = props.exec.execution_id
  await EventsOn(`toolbox:log:${id}`, (line) => {
    if (boundId !== id) return
    lines.value.push(line)
    nextTick(() => { if (body.value) body.value.scrollTop = body.value.scrollHeight })
  })
  await EventsOn(`toolbox:logend:${id}`, (payload) => {
    if (boundId !== id) return
    finished.value = true
    exitCode.value = payload.exit_code ?? -1
  })
}
function unbind() {
  if (boundId === null) return
  EventsOff(`toolbox:log:${boundId}`)
  EventsOff(`toolbox:logend:${boundId}`)
  boundId = null
}
function close() {
  unbind()
  emit('close')
}
watch(() => props.exec, () => bind(), { immediate: true })
onUnmounted(unbind)
</script>
