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
  grid-template-columns: 240px 1fr;
  gap: 0;
  min-height: 100vh;
  background: var(--color-bg);
}

.content {
  display: grid;
  grid-template-rows: auto 1fr;
  gap: 0;
}

.hero {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  padding: 20px 32px;
  border-bottom: 1px solid var(--color-border);
  background: var(--color-bg);
}

.eyebrow {
  margin: 0;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--color-text-muted);
}

h1 {
  margin: 4px 0 0;
  font-size: 24px;
  font-weight: 600;
  line-height: 1.2;
  color: var(--color-text);
}

.subtitle {
  display: none;
}

.page {
  max-width: 1200px;
  width: 100%;
  margin: 0 auto;
  padding: 24px 32px;
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
    align-items: flex-start;
  }
}
</style>
