<template>
  <div class="shell">
    <SidebarNav class="card" />
    <main class="content">
      <header class="hero card">
        <div>
          <p class="eyebrow">人生第二大脑</p>
          <h1>WeLife OS</h1>
          <p class="subtitle">把散落的对话，整理成可以行动的人生洞察。</p>
        </div>
        <div class="header-actions">
          <ReminderBell :count="pendingCount" @click="$router.push('/timeline')" />
          <StatusBar class="status" />
        </div>
      </header>
      <section class="page">
        <slot />
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import SidebarNav from "./SidebarNav.vue";
import StatusBar from "./StatusBar.vue";
import ReminderBell from "./ReminderBell.vue";
import { useReminder } from "../composables/useReminder";
import { computed } from "vue";

const { pending, startPolling } = useReminder();
startPolling();

const pendingCount = computed(() => pending.value.length);
</script>

<style scoped>
.shell {
  display: grid;
  grid-template-columns: 280px 1fr;
  gap: 20px;
  padding: 20px;
  min-height: 100vh;
}

.content {
  display: grid;
  grid-template-rows: auto 1fr;
  gap: 20px;
}

.hero {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  padding: 28px;
}

.eyebrow {
  margin: 0;
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--color-primary);
}

h1 {
  margin: 8px 0 10px;
  font-size: 38px;
  line-height: 1;
  color: var(--color-text);
}

.subtitle {
  margin: 0;
  max-width: 520px;
  color: var(--color-text-secondary);
}

.page {
  min-height: 0;
}

.status {
  min-width: 240px;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

@media (max-width: 900px) {
  .shell {
    grid-template-columns: 1fr;
  }

  .hero {
    flex-direction: column;
  }
}
</style>
