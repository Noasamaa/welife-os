import { describe, it, expect } from "vitest";
import { mount } from "@vue/test-utils";
import DropZone from "../DropZone.vue";

describe("DropZone", () => {
  it("renders default prompt text", () => {
    const wrapper = mount(DropZone);
    expect(wrapper.text()).toContain("拖拽聊天记录到此处");
    expect(wrapper.text()).toContain("支持 CSV");
  });

  it("shows upload text when dragging", async () => {
    const wrapper = mount(DropZone);
    await wrapper.find(".dropzone").trigger("dragover");
    expect(wrapper.text()).toContain("松开以上传");
  });

  it("adds active class on dragover", async () => {
    const wrapper = mount(DropZone);
    await wrapper.find(".dropzone").trigger("dragover");
    expect(wrapper.find(".dropzone").classes()).toContain("active");
  });

  it("removes active class on dragleave", async () => {
    const wrapper = mount(DropZone);
    const zone = wrapper.find(".dropzone");
    await zone.trigger("dragover");
    expect(zone.classes()).toContain("active");

    await zone.trigger("dragleave");
    expect(zone.classes()).not.toContain("active");
  });

  it("emits file event on drop with a file", async () => {
    const wrapper = mount(DropZone);
    const file = new File(["hello"], "test.csv", { type: "text/csv" });

    await wrapper.find(".dropzone").trigger("drop", {
      dataTransfer: { files: [file] },
    });

    expect(wrapper.emitted("file")).toBeTruthy();
    expect(wrapper.emitted("file")![0][0]).toEqual(file);
  });

  it("does not emit file on drop without files", async () => {
    const wrapper = mount(DropZone);

    await wrapper.find(".dropzone").trigger("drop", {
      dataTransfer: { files: [] },
    });

    expect(wrapper.emitted("file")).toBeFalsy();
  });

  it("passes accept prop to hidden input", () => {
    const wrapper = mount(DropZone, { props: { accept: ".csv,.json" } });
    const input = wrapper.find("input[type=file]");
    expect(input.attributes("accept")).toBe(".csv,.json");
  });
});
