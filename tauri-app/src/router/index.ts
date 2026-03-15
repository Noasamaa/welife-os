import { createRouter, createWebHistory } from "vue-router";

import CoachView from "../views/Coach.vue";
import DashboardView from "../views/Dashboard.vue";
import ForumView from "../views/Forum.vue";
import ImportView from "../views/Import.vue";
import ReportsView from "../views/Reports.vue";
import SettingsView from "../views/Settings.vue";
import SimulationView from "../views/Simulation.vue";
import TimelineView from "../views/Timeline.vue";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/", name: "dashboard", component: DashboardView },
    { path: "/import", name: "import", component: ImportView },
    { path: "/reports", name: "reports", component: ReportsView },
    { path: "/forum", name: "forum", component: ForumView },
    { path: "/coach", name: "coach", component: CoachView },
    { path: "/timeline", name: "timeline", component: TimelineView },
    { path: "/simulation", name: "simulation", component: SimulationView },
    { path: "/settings", name: "settings", component: SettingsView },
  ],
});

export default router;
