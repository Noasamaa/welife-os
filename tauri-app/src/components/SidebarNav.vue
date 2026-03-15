<template>
  <aside class="sidebar">
    <div>
      <p class="label">导航</p>
      <nav>
        <RouterLink
          v-for="item in items"
          :key="item.to"
          :to="item.to"
          class="nav-item"
        >
          <span>{{ item.label }}</span>
          <small>{{ item.hint }}</small>
        </RouterLink>
      </nav>
    </div>

    <div class="footnote">
      <p>v1.0.0</p>
      <small>先把结构搭稳，再接入导入、报告与 Agent 能力。</small>
    </div>
  </aside>
</template>

<script setup lang="ts">
const items = [
  { to: "/", label: "总览", hint: "Dashboard" },
  { to: "/import", label: "导入", hint: "Import" },
  { to: "/reports", label: "报告", hint: "Reports" },
  { to: "/forum", label: "论坛", hint: "Forum" },
  { to: "/coach", label: "执行教练", hint: "Coach" },
  { to: "/timeline", label: "时间线", hint: "Timeline" },
  { to: "/simulation", label: "模拟", hint: "Simulation" },
  { to: "/settings", label: "设置", hint: "Settings" },
];
</script>

<style scoped>
.sidebar {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  padding: 24px;
}

.label {
  margin: 0 0 16px;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--color-text-label);
}

nav {
  display: grid;
  gap: 10px;
}

.nav-item {
  display: grid;
  gap: 4px;
  padding: 14px 16px;
  border-radius: 16px;
  color: var(--color-nav-text);
  transition: transform 160ms ease, background 160ms ease;
}

.nav-item.router-link-exact-active {
  background: var(--color-primary-bg);
}

.nav-item:hover {
  background: var(--color-primary-bg-hover);
  transform: translateX(2px);
}

small {
  color: var(--color-text-muted);
}

.footnote {
  padding-top: 24px;
  border-top: 1px solid var(--color-border);
}

.footnote p {
  margin: 0 0 8px;
  font-weight: 700;
  color: var(--color-text);
}
</style>
