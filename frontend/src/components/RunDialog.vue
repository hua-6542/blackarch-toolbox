<template>
  <div class="dialog-mask" @click.self="$emit('close')">
    <div class="dialog">
      <h3>运行 {{ tool.name }}</h3>
      <p class="desc">{{ tool.description }}</p>
      <label>参数</label>
      <input v-model="args" class="args-input" placeholder="例如: -sV 192.168.1.1" @keydown.enter="run" />
      <label>运行环境</label>
      <select v-model="env" @change="refreshPreview">
        <option value="auto">自动（智能路由）</option>
        <option value="local">本地</option>
        <option value="podman">Podman 容器</option>
        <option value="vm">虚拟机</option>
      </select>
      <div v-if="decision" class="decision-box">
        将使用：<b>{{ envName(decision.env) }}</b>（{{ decision.reason }}）
      </div>
      <div v-if="tool.is_high_risk" class="danger-box">
        ⚠️ 该工具为高危工具，建议在 VM 隔离环境运行
      </div>
      <div class="dialog-actions">
        <button class="btn-ghost" @click="$emit('close')">取消</button>
        <button class="btn-primary" @click="run">执行</button>
      </div>
      <p v-if="error" class="error">{{ error }}</p>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { DryRun, RunTool } from '../../wailsjs/go/app/App'

const props = defineProps({ tool: { type: Object, required: true } })
const emit = defineEmits(['close', 'started'])
const args = ref('')
const env = ref('auto')
const decision = ref(null)
const error = ref('')

const envName = (e) => ({ local: '本地', podman: '容器', vm: '虚拟机' }[e] || e)

async function refreshPreview() {
  try {
    decision.value = await DryRun(props.tool.name, args.value, env.value)
  } catch (e) {
    decision.value = null
  }
}
async function run() {
  error.value = ''
  try {
    const res = await RunTool({ tool: props.tool.name, args: args.value, env: env.value })
    emit('started', { ...res, tool: props.tool.name })
    emit('close')
  } catch (e) {
    error.value = String(e)
  }
}
onMounted(refreshPreview)
</script>
