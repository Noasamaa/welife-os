<template>
  <section class="page">
    <div class="page-header">
      <h2>执行教练</h2>
      <p class="subtitle">将洞察转化为行动，追踪执行进度</p>
    </div>

    <div v-if="error" class="card error-banner">{{ error }}</div>

    <!-- Filter bar -->
    <div class="card filter-bar">
      <select v-model="filterStatus" class="form-select" @change="reload">
        <option value="">全部状态</option>
        <option value="pending">待处理</option>
        <option value="completed">已完成</option>
        <option value="dismissed">已取消</option>
      </select>
      <select v-model="filterCategory" class="form-select" @change="reload">
        <option value="">全部类别</option>
        <option value="relationship">关系维护</option>
        <option value="opportunity">机会跟进</option>
        <option value="risk">风险防范</option>
        <option value="general">通用</option>
      </select>
      <button class="btn-secondary" @click="reload">刷新</button>
    </div>

    <!-- Action items -->
    <div v-if="loading" class="loading">加载中...</div>

    <div v-if="items.length === 0 && !loading" class="card empty">
      暂无行动项。通过辩论论坛生成行动计划。
    </div>

    <div class="items-grid">
      <ActionItemCard
        v-for="item in items"
        :key="item.id"
        :item="item"
        @complete="handleComplete"
        @dismiss="handleDismiss"
      />
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useCoach } from "../composables/useCoach";
import ActionItemCard from "../components/ActionItemCard.vue";

const { items, loading, error, loadItems, updateStatus } = useCoach();

const filterStatus = ref("");
const filterCategory = ref("");

onMounted(() => reload());

function reload() {
  void loadItems(filterStatus.value || undefined, filterCategory.value || undefined);
}

function handleComplete(id: string) { void updateStatus(id, "completed"); }
function handleDismiss(id: string) { void updateStatus(id, "dismissed"); }
</script>

<style scoped>
.page { padding: 24px; display: flex; flex-direction: column; gap: 20px; }
.page-header h2 { margin: 0; }
.subtitle { color: var(--color-text-secondary, #666); margin: 4px 0 0; font-size: 14px; }
.card { background: var(--color-bg-card, #fff); border: 1px solid var(--color-border, #e0e0e0); border-radius: 10px; padding: 20px; }
.error-banner { background: #fde8e8; border-color: #e74c3c; color: #c0392b; }
.filter-bar { display: flex; gap: 12px; align-items: center; }
.form-select { padding: 8px 12px; border: 1px solid var(--color-border, #ddd); border-radius: 6px; font-size: 14px; background: var(--color-bg, #fff); }
.btn-secondary { padding: 8px 14px; background: transparent; border: 1px solid var(--color-border, #ddd); border-radius: 6px; cursor: pointer; font-size: 13px; }
.loading, .empty { text-align: center; padding: 24px; color: var(--color-text-secondary, #888); font-size: 14px; }
.items-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); gap: 12px; }
</style>
