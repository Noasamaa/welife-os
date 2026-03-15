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
}
.badge {
  position: absolute;
  top: -2px;
  right: 0;
  background: #e74c3c;
  color: white;
  font-size: 10px;
  font-weight: 600;
  min-width: 16px;
  height: 16px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 3px;
}
</style>
