<template>
  <aside class="sidebar">
    <div class="brand">🛠️ BlackArch ToolBox</div>
    <ul class="tree">
      <li
        v-for="cat in categories"
        :key="cat.key"
        :class="{ active: cat.key === selected }"
        @click="$emit('select', cat.key)"
      >
        <span class="icon">{{ cat.icon }}</span>{{ cat.label }}
      </li>
    </ul>
  </aside>
</template>

<script setup>
defineProps({
  categories: { type: Array, required: true },
  selected: { type: String, required: true },
})
defineEmits(['select'])
</script>
