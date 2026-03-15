import { describe, it, expect } from "vitest";
import { mount } from "@vue/test-utils";
import ReminderBell from "../ReminderBell.vue";

describe("ReminderBell", () => {
  it("renders bell icon", () => {
    const wrapper = mount(ReminderBell, { props: { count: 0 } });
    expect(wrapper.find(".bell-icon").exists()).toBe(true);
  });

  it("does not show badge when count is 0", () => {
    const wrapper = mount(ReminderBell, { props: { count: 0 } });
    expect(wrapper.find(".badge").exists()).toBe(false);
  });

  it("shows badge with count when count > 0", () => {
    const wrapper = mount(ReminderBell, { props: { count: 3 } });
    const badge = wrapper.find(".badge");
    expect(badge.exists()).toBe(true);
    expect(badge.text()).toBe("3");
  });

  it("shows 9+ when count exceeds 9", () => {
    const wrapper = mount(ReminderBell, { props: { count: 15 } });
    expect(wrapper.find(".badge").text()).toBe("9+");
  });

  it("emits click event on click", async () => {
    const wrapper = mount(ReminderBell, { props: { count: 1 } });
    await wrapper.find(".reminder-bell").trigger("click");
    expect(wrapper.emitted("click")).toBeTruthy();
  });
});
