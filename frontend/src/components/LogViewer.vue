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
import { GetExecutionLog, GetExecutionResult } from '../../wailsjs/go/app/App'

const props = defineProps({ exec: { type: Object, default: null } })
const emit = defineEmits(['close'])
const lines = ref([])
const finished = ref(false)
const exitCode = ref(null)
const body = ref(null)
let boundId = null

const envName = (e) => ({ local: '本地', podman: '容器', vm: '虚拟机' }[e] || e)
const exitClass = computed(() => (exitCode.value === 0 ? 'exit-ok' : 'exit-bad'))

function scrollBottom() {
  nextTick(() => { if (body.value) body.value.scrollTop = body.value.scrollHeight })
}

function mergeBuffered(buffered) {
  if (!buffered || buffered.length === 0) return
  const live = lines.value
  let overlap = 0
  const maxN = Math.min(live.length, buffered.length)
  outer: for (let n = maxN; n > 0; n--) {
    for (let i = 0; i < n; i++) {
      if (live[live.length - n + i] !== buffered[buffered.length - n + i]) continue outer
    }
    overlap = n
    break
  }
  lines.value = [...buffered, ...live.slice(overlap)]
}

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
    scrollBottom()
  })
  await EventsOn(`toolbox:logend:${id}`, (payload) => {
    if (boundId !== id) return
    finished.value = true
    exitCode.value = payload.exit_code ?? -1
  })
  const buffered = await GetExecutionLog(id).catch(() => [])
  if (boundId !== id) return
  mergeBuffered(buffered)
  scrollBottom()
  const result = await GetExecutionResult(id).catch(() => null)
  if (boundId !== id) return
  if (result && result.finished !== false) {
    finished.value = true
    exitCode.value = result.exit_code ?? -1
  }
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
