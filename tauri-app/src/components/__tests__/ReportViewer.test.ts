import { describe, it, expect } from "vitest";
import { mount } from "@vue/test-utils";
import ReportViewer from "../ReportViewer.vue";
import type { ReportContent } from "../../types/report";

// Stub ReportChart to avoid ECharts dependency in tests
const ReportChartStub = { template: "<div class='chart-stub' />" };

function makeContent(overrides?: Partial<ReportContent>): ReportContent {
  return {
    title: "Weekly Report",
    type: "weekly",
    period: { start: "2026-01-01T00:00:00Z", end: "2026-01-07T00:00:00Z" },
    sections: [],
    summary: "All good.",
    ...overrides,
  };
}

describe("ReportViewer", () => {
  function mountViewer(content: ReportContent) {
    return mount(ReportViewer, {
      props: { content },
      global: {
        stubs: { ReportChart: ReportChartStub },
      },
    });
  }

  it("renders title and type badge", () => {
    const wrapper = mountViewer(makeContent());
    expect(wrapper.find("h2").text()).toBe("Weekly Report");
    expect(wrapper.find(".type-badge").text()).toBe("每周简报");
  });

  it("renders period dates", () => {
    const wrapper = mountViewer(makeContent());
    expect(wrapper.text()).toContain("2026-01-01");
    expect(wrapper.text()).toContain("2026-01-07");
  });

  it("renders monthly type label", () => {
    const wrapper = mountViewer(makeContent({ type: "monthly" }));
    expect(wrapper.find(".type-badge").text()).toBe("每月报告");
  });

  it("renders annual type label", () => {
    const wrapper = mountViewer(makeContent({ type: "annual" }));
    expect(wrapper.find(".type-badge").text()).toBe("年度复盘");
  });

  it("renders summary section", () => {
    const wrapper = mountViewer(makeContent({ summary: "Very insightful" }));
    expect(wrapper.find(".summary-card").text()).toContain("Very insightful");
  });

  it("does not render summary when empty", () => {
    const wrapper = mountViewer(makeContent({ summary: "" }));
    expect(wrapper.find(".summary-card").exists()).toBe(false);
  });

  it("renders chart section using ReportChart stub", () => {
    const wrapper = mountViewer(
      makeContent({
        sections: [
          { title: "Activity", type: "chart", chart_type: "line", data: {}, items: [], narrative: "" },
        ],
      }),
    );
    expect(wrapper.find(".chart-stub").exists()).toBe(true);
    expect(wrapper.find(".section-title").text()).toBe("Activity");
  });

  it("renders list section with items", () => {
    const wrapper = mountViewer(
      makeContent({
        sections: [
          {
            title: "Key Contacts",
            type: "list",
            chart_type: undefined,
            data: undefined,
            items: [
              { title: "Alice", description: "Friend" },
              { title: "Bob", description: "Colleague" },
            ],
            narrative: "",
          },
        ],
      }),
    );
    const items = wrapper.findAll(".list-item");
    expect(items.length).toBe(2);
    expect(items[0].text()).toContain("Alice");
    expect(items[1].text()).toContain("Bob");
  });

  it("renders text section with narrative", () => {
    const wrapper = mountViewer(
      makeContent({
        sections: [
          { title: "Summary", type: "text", chart_type: undefined, data: undefined, items: [], narrative: "Great progress this week." },
        ],
      }),
    );
    expect(wrapper.find(".narrative").text()).toBe("Great progress this week.");
  });
});
