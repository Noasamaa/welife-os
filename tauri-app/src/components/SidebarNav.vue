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
  padding: 16px 12px;
  background: var(--color-bg-secondary);
  border-right: 1px solid var(--color-border);
}

.label {
  margin: 0 0 12px;
  padding: 0 12px;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: var(--color-text-label);
}

nav {
  display: grid;
  gap: 2px;
}

.nav-item {
  display: grid;
  gap: 2px;
  padding: 8px 12px;
  border-radius: var(--radius-md);
  font-size: 13px;
  font-weight: 500;
  color: var(--color-nav-text);
  position: relative;
  transition: background var(--transition-fast) ease;
}

.nav-item.router-link-exact-active {
  background: var(--color-nav-active-bg);
  color: var(--color-nav-active);
}

.nav-item.router-link-exact-active::before {
  content: "";
  position: absolute;
  left: 0;
  top: 6px;
  bottom: 6px;
  width: 2px;
  border-radius: var(--radius-full);
  background: var(--color-nav-active);
}

.nav-item:hover {
  background: var(--color-bg-hover);
}

small {
  font-size: 11px;
  color: var(--color-text-muted);
}

.footnote {
  padding-top: 16px;
  border-top: 1px solid var(--color-border);
}

.footnote p {
  margin: 0;
  padding: 0 12px;
  font-size: 11px;
  font-weight: 500;
  color: var(--color-text-muted);
}
</style>
