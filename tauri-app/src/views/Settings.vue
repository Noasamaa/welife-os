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
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
}

.subtitle {
  color: var(--color-text-secondary);
  margin: 4px 0 0;
  font-size: 13px;
}

.settings-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.card {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: 24px;
  box-shadow: var(--shadow-sm);
}

.card h3 {
  margin: 0 0 16px;
  font-size: 14px;
  font-weight: 600;
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
  padding-bottom: 12px;
  border-bottom: 1px solid var(--color-border);
}

.info-row:last-child {
  padding-bottom: 0;
  border-bottom: none;
}

.info-label {
  font-size: 13px;
  color: var(--color-text-secondary);
  flex-shrink: 0;
}

.info-value {
  font-size: 14px;
  color: var(--color-text);
  text-align: right;
  word-break: break-all;
}

.info-value.mono {
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 12px;
  color: var(--color-text-secondary);
}

.info-link {
  font-size: 14px;
  color: var(--color-primary);
  text-decoration: none;
  transition: opacity var(--transition-fast);
}

.info-link:hover {
  opacity: 0.8;
}

.connection-status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
}

.status-dot {
  width: 7px;
  height: 7px;
  border-radius: var(--radius-full);
  flex-shrink: 0;
}

.dot-ok {
  background: var(--color-success);
}

.dot-err {
  background: var(--color-danger);
}

.btn-primary {
  display: inline-flex;
  align-items: center;
  padding: 7px 16px;
  background: var(--color-primary);
  color: var(--color-text-inverse);
  border: 1px solid var(--color-primary);
  border-radius: var(--radius-md);
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  white-space: nowrap;
  transition: all var(--transition-fast);
}

.btn-primary:hover:not(:disabled) {
  background: var(--color-primary-hover);
  border-color: var(--color-primary-hover);
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.test-result {
  margin-top: 12px;
  padding: 10px 14px;
  border-radius: var(--radius-md);
  font-size: 13px;
  line-height: 1.5;
}

.result-ok {
  background: var(--color-success-bg);
  color: var(--color-success);
}

.result-err {
  background: var(--color-danger-bg);
  color: var(--color-danger);
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
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.theme-option:hover {
  border-color: var(--color-border-strong);
  background: var(--color-bg-hover);
}

.theme-option.active {
  border-color: var(--color-primary);
  background: var(--color-primary-bg);
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
  color: var(--color-text-secondary);
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
  font-weight: 500;
  color: var(--color-text-secondary);
}

.form-input {
  padding: 7px 12px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  font-size: 14px;
  color: var(--color-text);
  background: var(--color-bg-card);
  outline: none;
  transition: border-color var(--transition-fast);
}

.form-input:focus {
  border-color: var(--color-primary);
}

.form-input::placeholder {
  color: var(--color-text-muted);
}

.btn-group {
  display: flex;
  gap: 10px;
  margin-top: 16px;
}

.btn-secondary {
  display: inline-flex;
  align-items: center;
  padding: 7px 16px;
  background: var(--color-bg-card);
  color: var(--color-text);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  white-space: nowrap;
  transition: all var(--transition-fast);
}

.btn-secondary:hover:not(:disabled) {
  background: var(--color-bg-hover);
  border-color: var(--color-border-strong);
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

  .theme-options {
    flex-direction: column;
  }
}
</style>
