import { describe, it, expect, vi, beforeEach } from "vitest";
import { useReport } from "../useReport";
import { mockAllApi } from "../../test-utils/mock-api";
import type { Report } from "../../types/report";

describe("useReport", () => {
  let mocks: ReturnType<typeof mockAllApi>;

  beforeEach(() => {
    vi.restoreAllMocks();
    mocks = mockAllApi();
  });

  it("has correct initial state", () => {
    const { reports, currentReport, parsedContent, loading, generating, error } = useReport();
    expect(reports.value).toEqual([]);
    expect(currentReport.value).toBeNull();
    expect(parsedContent.value).toBeNull();
    expect(loading.value).toBe(false);
    expect(generating.value).toBe(false);
    expect(error.value).toBeNull();
  });

  it("loadReports fetches report list", async () => {
    const fakeReports: Report[] = [
      { id: "r1", type: "weekly", conversation_id: "c1", task_id: "t1", status: "completed", title: "Weekly", content: "{}", period_start: "2026-01-01", period_end: "2026-01-07", created_at: "2026-01-07" },
    ];
    mocks.fetchReports.mockResolvedValue(fakeReports);

    const { loadReports, reports, loading } = useReport();
    const promise = loadReports();
    expect(loading.value).toBe(true);
    await promise;

    expect(reports.value).toEqual(fakeReports);
    expect(loading.value).toBe(false);
  });

  it("loadReports sets error on failure", async () => {
    mocks.fetchReports.mockRejectedValue(new Error("fail"));

    const { loadReports, error } = useReport();
    await loadReports();

    expect(error.value).toBe("fail");
  });

  it("loadReport fetches a single report and populates parsedContent", async () => {
    const content = { title: "Test", type: "weekly", period: { start: "2026-01-01", end: "2026-01-07" }, sections: [], summary: "ok" };
    const fakeReport: Report = {
      id: "r1", type: "weekly", conversation_id: "c1", task_id: "t1", status: "completed",
      title: "Weekly", content: JSON.stringify(content),
      period_start: "2026-01-01", period_end: "2026-01-07", created_at: "2026-01-07",
    };
    mocks.fetchReport.mockResolvedValue(fakeReport);

    const { loadReport, currentReport, parsedContent } = useReport();
    await loadReport("r1");

    expect(currentReport.value).toEqual(fakeReport);
    expect(parsedContent.value).toEqual(content);
  });

  it("parsedContent returns null for invalid JSON", async () => {
    const fakeReport: Report = {
      id: "r1", type: "weekly", conversation_id: "c1", task_id: "t1", status: "completed",
      title: "Bad", content: "not-json",
      period_start: "2026-01-01", period_end: "2026-01-07", created_at: "2026-01-07",
    };
    mocks.fetchReport.mockResolvedValue(fakeReport);

    const { loadReport, parsedContent } = useReport();
    await loadReport("r1");

    expect(parsedContent.value).toBeNull();
  });

  it("generate triggers generation and reloads list", async () => {
    const genResult = { report_id: "r2", task_id: "t2" };
    mocks.generateReport.mockResolvedValue(genResult);
    mocks.fetchReports.mockResolvedValue([]);

    const { generate, generating } = useReport();
    const promise = generate("weekly", "c1", "2026-01-01", "2026-01-07");
    expect(generating.value).toBe(true);
    const result = await promise;

    expect(result).toEqual(genResult);
    expect(generating.value).toBe(false);
    expect(mocks.fetchReports).toHaveBeenCalled();
  });

  it("generate sets error on failure", async () => {
    mocks.generateReport.mockRejectedValue(new Error("gen fail"));

    const { generate, error, generating } = useReport();
    const result = await generate("weekly", "c1");

    expect(result).toBeNull();
    expect(error.value).toBe("gen fail");
    expect(generating.value).toBe(false);
  });

  it("remove deletes report and clears currentReport", async () => {
    mocks.deleteReport.mockResolvedValue(undefined);
    mocks.fetchReports.mockResolvedValue([]);

    const { remove, currentReport } = useReport();
    // Set a current report first
    const fakeReport: Report = {
      id: "r1", type: "weekly", conversation_id: "c1", task_id: "t1", status: "completed",
      title: "X", content: "{}", period_start: "", period_end: "", created_at: "",
    };
    mocks.fetchReport.mockResolvedValue(fakeReport);
    const { loadReport } = useReport();
    await loadReport("r1");

    await remove("r1");
    expect(currentReport.value).toBeNull();
    expect(mocks.deleteReport).toHaveBeenCalledWith("r1");
    expect(mocks.fetchReports).toHaveBeenCalled();
  });
});
