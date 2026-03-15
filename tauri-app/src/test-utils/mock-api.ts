import { vi } from "vitest";
import * as api from "../services/api";

/**
 * Creates vi.spyOn mocks for all exported API functions.
 * Call in beforeEach; all spies are auto-restored via vi.restoreAllMocks().
 */
export function mockAllApi() {
  return {
    // System
    getAPIBaseURL: vi.spyOn(api, "getAPIBaseURL"),
    fetchSystemStatus: vi.spyOn(api, "fetchSystemStatus"),
    fetchLLMConfig: vi.spyOn(api, "fetchLLMConfig"),
    updateLLMConfig: vi.spyOn(api, "updateLLMConfig"),

    // Import
    uploadFile: vi.spyOn(api, "uploadFile"),
    fetchImportJobs: vi.spyOn(api, "fetchImportJobs"),

    // Conversations
    fetchConversations: vi.spyOn(api, "fetchConversations"),

    // Graph
    triggerGraphBuild: vi.spyOn(api, "triggerGraphBuild"),
    fetchGraphOverview: vi.spyOn(api, "fetchGraphOverview"),

    // Forum
    triggerDebate: vi.spyOn(api, "triggerDebate"),
    fetchForumSessions: vi.spyOn(api, "fetchForumSessions"),
    fetchForumSession: vi.spyOn(api, "fetchForumSession"),

    // Reports
    generateReport: vi.spyOn(api, "generateReport"),
    fetchReports: vi.spyOn(api, "fetchReports"),
    fetchReport: vi.spyOn(api, "fetchReport"),
    deleteReport: vi.spyOn(api, "deleteReport"),
    fetchReportExportBlob: vi.spyOn(api, "fetchReportExportBlob"),

    // Coach
    generateActionPlan: vi.spyOn(api, "generateActionPlan"),
    fetchActionItems: vi.spyOn(api, "fetchActionItems"),
    updateActionItemStatus: vi.spyOn(api, "updateActionItemStatus"),
    deleteActionItem: vi.spyOn(api, "deleteActionItem"),

    // Reminders
    fetchPendingReminders: vi.spyOn(api, "fetchPendingReminders"),
    markReminderRead: vi.spyOn(api, "markReminderRead"),
    dismissReminder: vi.spyOn(api, "dismissReminder"),
    fetchReminderRules: vi.spyOn(api, "fetchReminderRules"),
    createReminderRule: vi.spyOn(api, "createReminderRule"),
    updateReminderRule: vi.spyOn(api, "updateReminderRule"),
    deleteReminderRule: vi.spyOn(api, "deleteReminderRule"),

    // Simulation
    buildProfiles: vi.spyOn(api, "buildProfiles"),
    fetchProfiles: vi.spyOn(api, "fetchProfiles"),
    runSimulation: vi.spyOn(api, "runSimulation"),
    fetchSimulations: vi.spyOn(api, "fetchSimulations"),
    fetchSimulation: vi.spyOn(api, "fetchSimulation"),
  };
}
