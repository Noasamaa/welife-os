import { ref, computed } from "vue";
import {
  type Report,
  type ReportContent,
  type ReportType,
  sanitizeReportContent,
} from "../types/report";
import {
  generateReport,
  fetchReports,
  fetchReport,
  deleteReport,
} from "../services/api";

export function useReport() {
  const reports = ref<Report[]>([]);
  const currentReport = ref<Report | null>(null);
  const loading = ref(false);
  const generating = ref(false);
  const error = ref<string | null>(null);

  const parsedContent = computed<ReportContent | null>(() => {
    if (!currentReport.value || !currentReport.value.content) return null;
    try {
      return sanitizeReportContent(JSON.parse(currentReport.value.content));
    } catch {
      return null;
    }
  });

  async function loadReports() {
    loading.value = true;
    error.value = null;
    try {
      reports.value = await fetchReports();
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : "加载报告列表失败";
    } finally {
      loading.value = false;
    }
  }

  async function loadReport(id: string) {
    loading.value = true;
    error.value = null;
    try {
      currentReport.value = await fetchReport(id);
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : "加载报告失败";
    } finally {
      loading.value = false;
    }
  }

  async function generate(
    type: ReportType,
    conversationID: string,
    periodStart?: string,
    periodEnd?: string,
  ): Promise<{ report_id: string; task_id: string } | null> {
    generating.value = true;
    error.value = null;
    try {
      const result = await generateReport(type, conversationID, periodStart, periodEnd);
      await loadReports();
      return result;
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : "生成报告失败";
      return null;
    } finally {
      generating.value = false;
    }
  }

  async function remove(id: string) {
    error.value = null;
    try {
      await deleteReport(id);
      currentReport.value = null;
      await loadReports();
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : "删除报告失败";
    }
  }

  return {
    reports,
    currentReport,
    parsedContent,
    loading,
    generating,
    error,
    loadReports,
    loadReport,
    generate,
    remove,
  };
}
