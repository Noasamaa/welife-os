import { createApp, type App } from "vue";
import { beforeEach } from "vitest";

/**
 * Wraps a composable call inside a real Vue component instance so that
 * lifecycle hooks (onMounted, onUnmounted, etc.) are correctly registered.
 * Returns the composable's return value and an unmount function.
 */
export function withSetup<T>(composable: () => T): { result: T; app: App; unmount: () => void } {
  let result!: T;
  const app = createApp({
    setup() {
      result = composable();
      return () => {};
    },
  });
  const root = document.createElement("div");
  app.mount(root);
  return { result, app, unmount: () => app.unmount() };
}
