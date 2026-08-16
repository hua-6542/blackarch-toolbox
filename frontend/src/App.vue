<template>
  <div class="layout">
    <CategoryTree :categories="categories" :selected="selected" @select="select" />
    <div class="main">
      <div class="search-bar">
        <input v-model="search" class="search" placeholder="🔍 输入工具名..." />
        <button class="link-btn" @click="openAudit">🩺 健康审计</button>
      </div>
      <div class="grid">
        <ToolCard v-for="t in filtered" :key="t.id" :tool="t" @run="openRun" />
      </div>
      <div v-if="filtered.length === 0" class="empty">未找到匹配的工具</div>
    </div>
    <RunDialog v-if="runTool" :tool="runTool" @close="runTool = null" @started="onStarted" />
    <LogViewer :exec="currentExec" @close="currentExec = null" />
    <AuditPanel v-if="showAudit" :checks="checks" @close="showAudit = false" @refresh="refreshHealth" />
    <StatusBar :checks="checks" :last-run="lastRunText" @workspace="openWorkspace" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import CategoryTree from './components/CategoryTree.vue'
import ToolCard from './components/ToolCard.vue'
import RunDialog from './components/RunDialog.vue'
import LogViewer from './components/LogViewer.vue'
import AuditPanel from './components/AuditPanel.vue'
import StatusBar from './components/StatusBar.vue'
import { GetTools, RunHealthCheck, OpenWorkspaceFile } from '../wailsjs/go/app/App'

const categories = [
  { key: '', label: '全部', icon: '📁' },
  { key: 'scanner', label: '扫描', icon: '📁' },
  { key: 'cracker', label: '破解', icon: '📁' },
  { key: 'exploitation', label: '利用', icon: '📁' },
  { key: 'forensics', label: '取证', icon: '📁' },
  { key: 'wireless', label: '无线', icon: '📁' },
  { key: 'web', label: 'Web', icon: '📁' },
  { key: 'recon', label: '侦察', icon: '📁' },
  { key: 'stego', label: '隐写', icon: '📁' },
  { key: 'social', label: '社工', icon: '📁' },
  { key: 'automation', label: '自动化', icon: '📁' },
]

const tools = ref([])
const selected = ref('')
const search = ref('')
const runTool = ref(null)
const currentExec = ref(null)
const showAudit = ref(false)
const checks = ref([])
const lastRunText = ref(null)

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  return tools.value.filter((t) => {
    if (selected.value && t.category !== selected.value) return false
    if (!q) return true
    return t.name.toLowerCase().includes(q) || (t.description || '').toLowerCase().includes(q)
  })
})

function select(key) { selected.value = key }

async function refreshTools() {
  try { tools.value = await GetTools('') } catch (e) { console.error(e) }
}
async function refreshHealth() {
  try { checks.value = await RunHealthCheck() } catch (e) { console.error(e) }
}
function openRun(tool) { runTool.value = tool }
function openAudit() { showAudit.value = true }
function onStarted(res) {
  currentExec.value = res
  lastRunText.value = new Date().toLocaleTimeString()
  refreshTools()
  refreshHealth()
}
async function openWorkspace() {
  try { await OpenWorkspaceFile('') } catch (e) { console.error(e) }
}
onMounted(() => { refreshTools(); refreshHealth() })
</script>
