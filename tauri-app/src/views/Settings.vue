<template>
  <section class="page">
    <div class="page-header">
      <h2>系统设置</h2>
      <p class="subtitle">管理 LLM 连接、存储信息、主题与应用信息</p>
    </div>

    <div class="settings-grid">
      <!-- Section 1: LLM 配置 -->
      <div class="card card-wide">
        <h3>LLM 配置</h3>
        <div class="form-grid">
          <div class="form-row">
            <label class="form-label" for="llm-provider">Provider</label>
            <select
              id="llm-provider"
              v-model="formProvider"
              class="form-input"
            >
              <option value="ollama">Ollama (本地)</option>
              <option value="openai-compatible">OpenAI 兼容 (云端)</option>
            </select>
          </div>
          <div class="form-row">
            <label class="form-label" for="llm-url">
              {{ formProvider === 'openai-compatible' ? 'API 地址' : 'Ollama 地址' }}
            </label>
            <input
              id="llm-url"
              v-model="formBaseURL"
              type="text"
              class="form-input"
              placeholder="http://127.0.0.1:11434"
            />
          </div>
          <div class="form-row">
            <label class="form-label" for="llm-model">模型</label>
            <input
              id="llm-model"
              v-model="formModel"
              type="text"
              class="form-input"
              placeholder="qwen3.5:9b"
            />
          </div>
          <div v-if="formProvider === 'openai-compatible'" class="form-row">
            <label class="form-label" for="llm-apikey">API Key</label>
            <input
              id="llm-apikey"
              v-model="formAPIKey"
              type="password"
              class="form-input"
              :placeholder="llmConfig.config?.api_key || '输入 API Key'"
            />
          </div>
          <div class="form-row">
            <label class="form-label" for="llm-embed">Embedding 模型</label>
            <input
              id="llm-embed"
              v-model="formEmbeddingModel"
              type="text"
              class="form-input"
              placeholder="留空则禁用向量搜索"
            />
          </div>
          <div class="form-row">
            <span class="form-label">连接状态</span>
            <span class="connection-status">
              <span class="status-dot" :class="llmConnected ? 'dot-ok' : 'dot-err'" />
              {{ llmStatusLabel }}
            </span>
          </div>
        </div>
        <div class="btn-group">
          <button
            class="btn-primary"
            :disabled="llmConfig.saving"
            @click="handleSaveConfig"
          >
            {{ llmConfig.saving ? "保存中..." : "保存配置" }}
          </button>
          <button
            class="btn-secondary"
            :disabled="testing"
            @click="handleTestConnection"
          >
            {{ testing ? "测试中..." : "测试连接" }}
          </button>
        </div>
        <div v-if="llmConfig.saveSuccess" class="test-result result-ok">
          配置已保存并生效
        </div>
        <div v-if="llmConfig.saveError" class="test-result result-err">
          {{ llmConfig.saveError }}
        </div>
        <div v-if="testResult" class="test-result" :class="testResult.ok ? 'result-ok' : 'result-err'">
          {{ testResult.message }}
        </div>
      </div>

      <!-- Section 2: 存储信息 -->
      <div class="card">
        <h3>存储信息</h3>
        <div class="info-list">
          <div class="info-row">
            <span class="info-label">数据库路径</span>
            <span class="info-value mono">{{ systemStatus?.storage.path ?? "检查中..." }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">加密状态</span>
            <span class="info-value">
              <span class="status-dot" :class="storageReady ? 'dot-ok' : 'dot-err'" />
              {{ storageReady ? "SQLCipher 已启用" : "未就绪" }}
            </span>
          </div>
          <div class="info-row">
            <span class="info-label">存储驱动</span>
            <span class="info-value">{{ systemStatus?.storage.driver ?? "检查中..." }}</span>
          </div>
        </div>
      </div>

      <!-- Section 3: 主题设置 -->
      <div class="card">
        <h3>主题设置</h3>
        <div class="theme-options">
          <label
            v-for="opt in themeOptions"
            :key="opt.value"
            class="theme-option"
            :class="{ active: currentTheme === opt.value }"
          >
            <input
              type="radio"
              :value="opt.value"
              v-model="currentTheme"
              class="sr-only"
              @change="handleThemeChange(opt.value)"
            />
            <span class="theme-icon">{{ opt.icon }}</span>
            <span class="theme-label">{{ opt.label }}</span>
          </label>
        </div>
      </div>

      <!-- Section 4: 关于 -->
      <div class="card">
        <h3>关于</h3>
        <div class="info-list">
          <div class="info-row">
            <span class="info-label">应用名称</span>
            <span class="info-value">WeLife OS</span>
          </div>
          <div class="info-row">
            <span class="info-label">版本</span>
            <span class="info-value">v1.0.0</span>
          </div>
          <div class="info-row">
            <span class="info-label">许可证</span>
            <span class="info-value">AGPL-3.0</span>
          </div>
          <div class="info-row">
            <span class="info-label">源代码</span>
            <a
              class="info-link"
              href="https://github.com/Noasamaa/welife-os"
              target="_blank"
              rel="noopener noreferrer"
            >
              github.com/Noasamaa/welife-os
            </a>
          </div>
        </div>
      </div>

      <!-- Section 5: 更新 -->
      <div class="card">
        <h3>应用更新</h3>
        <div class="info-list">
          <div class="info-row">
            <span class="info-label">当前版本</span>
            <span class="info-value">v1.0.0</span>
          </div>
          <div class="info-row">
            <span class="info-label">更新通道</span>
            <span class="info-value">{{ updater.enabled ? "已启用" : "已关闭" }}</span>
          </div>
        </div>
        <div class="test-result result-err">
          {{ updater.disabledReason }}
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from "vue";

import { useBackendHealth } from "../composables/useBackendHealth";
import { useLLMConfig } from "../composables/useLLMConfig";
import { useUpdater } from "../composables/useUpdater";
import { fetchSystemStatus } from "../services/api";

type ThemeValue = "light" | "dark" | "system";

interface TestResult {
  ok: boolean;
  message: string;
}

const { systemStatus } = useBackendHealth();
const llmConfig = reactive(useLLMConfig());
const updater = reactive(useUpdater());

const testing = ref(false);
const testResult = ref<TestResult | null>(null);
const currentTheme = ref<ThemeValue>("system");

// LLM form fields — populated from loaded config.
const formProvider = ref("ollama");
const formBaseURL = ref("");
const formModel = ref("");
const formAPIKey = ref("");
const formEmbeddingModel = ref("");

const themeOptions: ReadonlyArray<{ value: ThemeValue; icon: string; label: string }> = [
  { value: "light", icon: "\u2600", label: "\u4eae\u8272" },
  { value: "dark", icon: "\u263e", label: "\u6697\u8272" },
  { value: "system", icon: "\u2699", label: "\u8ddf\u968f\u7cfb\u7edf" },
];

const llmConnected = computed(() => systemStatus.value?.llm.reachable === true);

const llmStatusLabel = computed(() => {
  if (!systemStatus.value) return "检查中...";
  return systemStatus.value.llm.reachable ? "已连接" : "未连接";
});

const storageReady = computed(() => systemStatus.value?.storage.ready === true);

// Sync form fields when config is loaded from API.
watch(() => llmConfig.config, (cfg) => {
  if (!cfg) return;
  formProvider.value = cfg.provider || "ollama";
  formBaseURL.value = cfg.base_url || "";
  formModel.value = cfg.model || "";
  formEmbeddingModel.value = cfg.embedding_model || "";
  // Don't populate formAPIKey — it's masked from the server.
  formAPIKey.value = "";
});

async function handleSaveConfig(): Promise<void> {
  testResult.value = null;
  const patch: Record<string, string> = {
    provider: formProvider.value,
    base_url: formBaseURL.value,
    model: formModel.value,
    embedding_model: formEmbeddingModel.value,
  };
  // Only send api_key if the user actually typed something new.
  if (formAPIKey.value) {
    patch.api_key = formAPIKey.value;
  }
  await llmConfig.save(patch);
}

function applyTheme(theme: ThemeValue): void {
  const root = document.documentElement;
  root.classList.remove("light", "dark");
  if (theme === "light") root.classList.add("light");
  else if (theme === "dark") root.classList.add("dark");
  localStorage.setItem("welife-theme", theme);
}

function handleThemeChange(theme: ThemeValue): void {
  applyTheme(theme);
}

async function handleTestConnection(): Promise<void> {
  testing.value = true;
  testResult.value = null;
  try {
    const result = await fetchSystemStatus();
    if (result.llm.reachable) {
      testResult.value = {
        ok: true,
        message: `连接成功 - Provider: ${result.llm.provider}, 模型: ${result.llm.model}`,
      };
    } else {
      const hint = result.llm.provider === "ollama"
        ? "Ollama 服务不可达，请检查服务是否启动。"
        : "云端 LLM 服务不可达，请检查 API 地址和密钥。";
      testResult.value = {
        ok: false,
        message: hint,
      };
    }
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "未知错误";
    testResult.value = { ok: false, message: `连接失败: ${message}` };
  } finally {
    testing.value = false;
  }
}

onMounted(() => {
  const saved = localStorage.getItem("welife-theme");
  if (saved === "light" || saved === "dark" || saved === "system") {
    currentTheme.value = saved;
    applyTheme(saved);
  }
  void llmConfig.load();
});
</script>

<style scoped>
.page {
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.page-header h2 {
  margin: 0;
}

.subtitle {
  color: var(--color-text-secondary, #666);
  margin: 4px 0 0;
  font-size: 14px;
}

.settings-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 20px;
}

.card {
  background: var(--color-bg-card, #fff);
  border: 1px solid var(--color-border, #e0e0e0);
  border-radius: 10px;
  padding: 20px;
}

.card h3 {
  margin: 0 0 16px;
}

.info-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.info-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.info-label {
  font-size: 14px;
  color: var(--color-text-secondary, #666);
  flex-shrink: 0;
}

.info-value {
  font-size: 14px;
  text-align: right;
  word-break: break-all;
}

.info-value.mono {
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 12px;
}

.info-link {
  font-size: 14px;
  color: var(--color-primary, #4a90d9);
  text-decoration: none;
}

.info-link:hover {
  text-decoration: underline;
}

.connection-status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.dot-ok {
  background: #27ae60;
}

.dot-err {
  background: #e74c3c;
}

.test-btn {
  margin-top: 16px;
}

.btn-primary {
  padding: 8px 20px;
  background: var(--color-primary, #4a90d9);
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  white-space: nowrap;
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.test-result {
  margin-top: 12px;
  padding: 10px 14px;
  border-radius: 6px;
  font-size: 13px;
}

.result-ok {
  background: #e8f8ef;
  color: #27ae60;
  border: 1px solid #27ae60;
}

.result-err {
  background: #fde8e8;
  color: #c0392b;
  border: 1px solid #e74c3c;
}

.theme-options {
  display: flex;
  gap: 12px;
}

.theme-option {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 16px 12px;
  border: 2px solid var(--color-border, #e0e0e0);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s;
}

.theme-option:hover {
  border-color: var(--color-primary, #4a90d9);
}

.theme-option.active {
  border-color: var(--color-primary, #4a90d9);
  background: var(--color-bg-secondary, #f0f7ff);
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
}

.theme-icon {
  font-size: 24px;
}

.theme-label {
  font-size: 13px;
  font-weight: 500;
}

.card-wide {
  grid-column: span 2;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 14px;
}

.form-row {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.form-label {
  font-size: 13px;
  color: var(--color-text-secondary, #666);
}

.form-input {
  padding: 8px 10px;
  border: 1px solid var(--color-border, #e0e0e0);
  border-radius: 6px;
  font-size: 14px;
  background: var(--color-bg-card, #fff);
  color: var(--color-text, #333);
  outline: none;
  transition: border-color 0.15s;
}

.form-input:focus {
  border-color: var(--color-primary, #4a90d9);
}

.btn-group {
  display: flex;
  gap: 10px;
  margin-top: 16px;
}

.btn-secondary {
  padding: 8px 20px;
  background: var(--color-bg-secondary, #f0f7ff);
  color: var(--color-text, #333);
  border: 1px solid var(--color-border, #e0e0e0);
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  white-space: nowrap;
}

.btn-secondary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

@media (max-width: 768px) {
  .settings-grid {
    grid-template-columns: 1fr;
  }

  .card-wide {
    grid-column: span 1;
  }

  .form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
