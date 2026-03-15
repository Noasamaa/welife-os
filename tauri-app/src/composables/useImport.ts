import { ref } from "vue";
import type { ImportJob, ImportResult } from "../types/import";
import { uploadFile, fetchImportJobs } from "../services/api";

export function useImport() {
  const jobs = ref<ImportJob[]>([]);
  const uploading = ref(false);
  const error = ref<string | null>(null);

  async function upload(file: File, format?: string, selfName?: string): Promise<ImportResult | null> {
    uploading.value = true;
    error.value = null;
    try {
      const result = await uploadFile(file, format, selfName);
      await refreshJobs();
      return result;
    } catch (e: any) {
      error.value = e.message ?? "上传失败";
      return null;
    } finally {
      uploading.value = false;
    }
  }

  async function refreshJobs() {
    try {
      jobs.value = await fetchImportJobs();
    } catch (e: any) {
      error.value = e.message ?? "获取任务列表失败";
    }
  }

  return { jobs, uploading, error, upload, refreshJobs };
}
