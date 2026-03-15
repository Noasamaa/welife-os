import { ref } from "vue";

import { fetchLLMConfig, updateLLMConfig } from "../services/api";
import type { LLMConfig } from "../types/api";

export function useLLMConfig() {
  const config = ref<LLMConfig | null>(null);
  const loading = ref(false);
  const saving = ref(false);
  const saveError = ref<string | null>(null);
  const saveSuccess = ref(false);

  async function load(): Promise<void> {
    loading.value = true;
    try {
      config.value = await fetchLLMConfig();
    } catch (e: unknown) {
      const message = e instanceof Error ? e.message : "加载配置失败";
      saveError.value = message;
    } finally {
      loading.value = false;
    }
  }

  async function save(patch: Partial<LLMConfig>): Promise<void> {
    saving.value = true;
    saveError.value = null;
    saveSuccess.value = false;
    try {
      await updateLLMConfig(patch);
      saveSuccess.value = true;
      // Reload to get updated masked values.
      await load();
    } catch (e: unknown) {
      const message = e instanceof Error ? e.message : "保存失败";
      saveError.value = message;
    } finally {
      saving.value = false;
    }
  }

  return { config, loading, saving, saveError, saveSuccess, load, save };
}
