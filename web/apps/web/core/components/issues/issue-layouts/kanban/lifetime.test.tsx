// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { tab, view, container, dragSources, mount } from "./fake-kanban";
import { act } from "react";
import { expect, it, vi } from "vitest";
import { CustomMenu } from "@nerve/ui";

it.each(["Enter", " "])(
  "the actual card opens its menu by %s without opening peek, then returns focus",
  async (key) => {
    await mount(undefined, ({ customActionButton }) => (
      <CustomMenu customButton={customActionButton} ariaLabel="Card options">
        <CustomMenu.MenuItem>Copy</CustomMenu.MenuItem>
      </CustomMenu>
    ));
    const button = container.querySelector<HTMLButtonElement>('[aria-label="Card options"]')!;
    await act(async () => {
      button.focus();
      button.dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true }));
      await vi.advanceTimersByTimeAsync(50);
    });
    expect(button.getAttribute("aria-expanded")).toBe("true");
    expect(tab.app.issue.issueDetail.peekIssue).toBeUndefined();
    await act(async () => {
      document
        .querySelector('[role="menu"]')!
        .dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }));
      await vi.advanceTimersByTimeAsync(50);
    });
    expect(button.getAttribute("aria-expanded")).toBe("false");
    expect(document.activeElement).toBe(button);
  }
);

it("unmounting the actual drag source releases its delete-banner state", async () => {
  await mount();
  const source = [...dragSources.values()][0];
  await act(async () => source.onDragStart!({} as never));
  expect(tab.app.issue.issueKanBanView.isDragging).toBe(true);
  await act(async () => view.render(null));
  expect(tab.app.issue.issueKanBanView.isDragging).toBe(false);
});
it("a stale source drop cannot clear a newer drag's state", async () => {
  await mount();
  const [oldSource, newSource] = [...dragSources.values()];
  await act(async () => oldSource.onDragStart!({} as never));
  await act(async () => newSource.onDragStart!({} as never));
  await act(async () => oldSource.onDrop!({} as never));
  expect(tab.app.issue.issueKanBanView.isDragging).toBe(true);
  await act(async () => newSource.onDrop!({} as never));
  expect(tab.app.issue.issueKanBanView.isDragging).toBe(false);
  await act(async () => view.render(null));
  expect(tab.app.issue.issueKanBanView.isDragging).toBe(false);
});
