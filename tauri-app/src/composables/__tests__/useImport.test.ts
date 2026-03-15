import { describe, it, expect, vi, beforeEach } from "vitest";
import { useImport } from "../useImport";
import { mockAllApi } from "../../test-utils/mock-api";
import type { ImportResult, ImportJob } from "../../types/import";

describe("useImport", () => {
  let mocks: ReturnType<typeof mockAllApi>;

  beforeEach(() => {
    mocks = mockAllApi();
    vi.restoreAllMocks();
    mocks = mockAllApi();
  });

  it("has correct initial state", () => {
    const { jobs, uploading, error } = useImport();
    expect(jobs.value).toEqual([]);
    expect(uploading.value).toBe(false);
    expect(error.value).toBeNull();
  });

  it("upload sets uploading and refreshes jobs on success", async () => {
    const fakeResult: ImportResult = {
      job_id: "j1",
      task_id: "t1",
      conversation_id: "c1",
      message_count: 10,
    };
    const fakeJobs: ImportJob[] = [
      {
        id: "j1",
        task_id: "t1",
        file_name: "chat.csv",
        format: "generic_csv",
        status: "succeeded",
        message_count: 10,
        started_at: "2026-01-01",
      },
    ];

    mocks.uploadFile.mockResolvedValue(fakeResult);
    mocks.fetchImportJobs.mockResolvedValue(fakeJobs);

    const { upload, uploading, jobs } = useImport();
    const file = new File(["content"], "chat.csv");
    const result = await upload(file, "generic_csv");

    expect(result).toEqual(fakeResult);
    expect(uploading.value).toBe(false);
    expect(jobs.value).toEqual(fakeJobs);
    expect(mocks.uploadFile).toHaveBeenCalledWith(file, "generic_csv", undefined);
  });

  it("upload sets error on failure", async () => {
    mocks.uploadFile.mockRejectedValue(new Error("network error"));

    const { upload, error, uploading } = useImport();
    const result = await upload(new File([""], "f.csv"));

    expect(result).toBeNull();
    expect(error.value).toBe("network error");
    expect(uploading.value).toBe(false);
  });

  it("refreshJobs fetches job list", async () => {
    const fakeJobs: ImportJob[] = [];
    mocks.fetchImportJobs.mockResolvedValue(fakeJobs);

    const { refreshJobs, jobs } = useImport();
    await refreshJobs();

    expect(jobs.value).toEqual(fakeJobs);
  });

  it("refreshJobs sets error on failure", async () => {
    mocks.fetchImportJobs.mockRejectedValue(new Error("fail"));

    const { refreshJobs, error } = useImport();
    await refreshJobs();

    expect(error.value).toBe("fail");
  });
});
