<template>
  <div class="graph-filter-panel">
    <div class="filter-section">
      <input
        v-model="searchInput"
        type="text"
        class="search-input"
        placeholder="搜索节点..."
        @input="onSearchInput"
      />
    </div>

    <div class="filter-section">
      <span class="filter-label">类型筛选</span>
      <div class="type-chips">
        <button
          v-for="t in allTypes"
          :key="t"
          class="type-chip"
          :class="{ active: isTypeActive(t) }"
          :style="chipStyle(t)"
          @click="toggleType(t)"
        >
          {{ t }}
        </button>
      </div>
    </div>

    <div class="filter-section">
      <label class="orphan-toggle">
        <input v-model="showOrphansLocal" type="checkbox" @change="onOrphanToggle" />
        <span>显示孤立节点</span>
      </label>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onUnmounted } from "vue";
import type { GraphFilters } from "../composables/usePixiGraph";

const TYPE_COLORS: Record<string, string> = {
  person: "#2d6a4f",
  event: "#e67e22",
  topic: "#3498db",
  promise: "#9b59b6",
  place: "#e74c3c",
};

const props = defineProps<{
  filters: GraphFilters;
  entityTypes: Record<string, number>;
  onRefresh: () => void;
}>();

const allTypes = computed(() => Object.keys(props.entityTypes));

const searchInput = ref(props.filters.searchQuery.value);
const showOrphansLocal = ref(props.filters.showOrphans.value);

let debounceTimer: ReturnType<typeof setTimeout> | null = null;

function onSearchInput(): void {
  if (debounceTimer !== null) {
    clearTimeout(debounceTimer);
  }
  debounceTimer = setTimeout(() => {
    props.filters.searchQuery.value = searchInput.value;
    props.onRefresh();
  }, 200);
}

function isTypeActive(type: string): boolean {
  const active = props.filters.activeTypes.value;
  return active.size === 0 || active.has(type);
}

function toggleType(type: string): void {
  const current = props.filters.activeTypes.value;
  const next = new Set(current);
  if (next.has(type)) {
    next.delete(type);
  } else {
    next.add(type);
  }
  // If all types selected, clear the filter (show all)
  if (next.size === allTypes.value.length) {
    props.filters.activeTypes.value = new Set();
  } else {
    props.filters.activeTypes.value = next;
  }
  props.onRefresh();
}

function chipStyle(type: string): Record<string, string> {
  const color = TYPE_COLORS[type] ?? "#888";
  const active = isTypeActive(type);
  return {
    borderColor: color,
    backgroundColor: active ? color + "30" : "transparent",
    color: active ? color : "#666",
  };
}

function onOrphanToggle(): void {
  props.filters.showOrphans.value = showOrphansLocal.value;
  props.onRefresh();
}

onUnmounted(() => {
  if (debounceTimer !== null) clearTimeout(debounceTimer);
});
</script>

<style scoped>
.graph-filter-panel {
  position: absolute;
  top: 8px;
  right: 8px;
  width: 200px;
  padding: 10px;
  background: rgba(20, 20, 40, 0.92);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  backdrop-filter: blur(8px);
  z-index: 10;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.filter-section {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.filter-label {
  font-size: 11px;
  color: #888;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.search-input {
  width: 100%;
  padding: 6px 8px;
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.06);
  color: #e0e0e0;
  font-size: 13px;
  outline: none;
  box-sizing: border-box;
}

.search-input::placeholder {
  color: #666;
}

.search-input:focus {
  border-color: rgba(255, 255, 255, 0.3);
}

.type-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.type-chip {
  padding: 2px 8px;
  font-size: 11px;
  border: 1px solid;
  border-radius: 12px;
  cursor: pointer;
  background: transparent;
  transition: all 0.15s ease;
}

.type-chip:hover {
  opacity: 0.8;
}

.orphan-toggle {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #aaa;
  cursor: pointer;
}

.orphan-toggle input {
  accent-color: #2d6a4f;
}
</style>
