<template>
  <div class="reminder-bell" @click="$emit('click')">
    <span class="bell-icon">&#128276;</span>
    <span v-if="count > 0" class="badge">{{ count > 9 ? '9+' : count }}</span>
  </div>
</template>

<script setup lang="ts">
defineProps<{ count: number }>();
defineEmits<{ click: [] }>();
</script>

<style scoped>
.reminder-bell {
  position: relative;
  cursor: pointer;
  padding: 4px 8px;
  font-size: 18px;
  transition: opacity var(--transition-fast);
}

.reminder-bell:hover {
  opacity: 0.8;
}

.badge {
  position: absolute;
  top: -2px;
  right: 0;
  background: var(--color-danger);
  color: var(--color-text-inverse);
  font-size: 10px;
  font-weight: 600;
  min-width: 16px;
  height: 16px;
  border-radius: var(--radius-full);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 3px;
}
</style>
