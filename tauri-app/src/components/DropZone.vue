<template>
  <div
    class="dropzone"
    :class="{ active: dragging }"
    @dragover.prevent="dragging = true"
    @dragleave="dragging = false"
    @drop.prevent="onDrop"
    @click="openPicker"
    role="button"
    tabindex="0"
    aria-label="拖拽文件到此处或点击选择"
    @keydown.enter="openPicker"
  >
    <input
      ref="fileInput"
      type="file"
      :accept="accept"
      style="display: none"
      @change="onFileChange"
    />
    <p v-if="!dragging">拖拽聊天记录到此处，或点击选择文件</p>
    <p v-else>松开以上传</p>
    <p class="hint">支持 CSV / JSON / TXT / SQLite(chat.db) 等聊天导出文件</p>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";

const props = defineProps<{
  accept?: string;
}>();

const emit = defineEmits<{
  (e: "file", file: File): void;
}>();

const dragging = ref(false);
const fileInput = ref<HTMLInputElement>();

function openPicker() {
  fileInput.value?.click();
}

function onDrop(e: DragEvent) {
  dragging.value = false;
  const file = e.dataTransfer?.files[0];
  if (file) emit("file", file);
}

function onFileChange(e: Event) {
  const input = e.target as HTMLInputElement;
  const file = input.files?.[0];
  if (file) emit("file", file);
  input.value = "";
}
</script>

<style scoped>
.dropzone {
  border: 2px dashed #b7c9c1;
  border-radius: 12px;
  padding: 40px 24px;
  text-align: center;
  cursor: pointer;
  transition: all 0.2s;
}

.dropzone:hover,
.dropzone.active {
  border-color: #2d6a4f;
  background: rgba(45, 106, 79, 0.04);
}

.hint {
  margin-top: 8px;
  font-size: 13px;
  color: #7a9a8e;
}
</style>
