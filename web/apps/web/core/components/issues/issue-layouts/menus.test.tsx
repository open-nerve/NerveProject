// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { tab, host, renderer } from "../fake-issue-properties";
import { act, type ComponentType } from "react";
import { expect, it, vi } from "vitest";
import { ModalCore, ModalTitle } from "@nerve/ui";
import { issueDetailOf } from "@/store/issue/fake-issues";
import { ProjectIssueQuickActions } from "./quick-action-dropdowns/project-issue";
import { WorkItemDetailQuickActions } from "./quick-action-dropdowns/issue-detail";
import { ArchivedIssueQuickActions } from "./quick-action-dropdowns/archived-issue";
import type { IQuickActionProps } from "./list/list-view-types";
import { IssueBlock } from "./list/block";

vi.mock("@/hooks/store/use-app-theme", () => ({ useAppTheme: () => ({ sidebarCollapsed: false }) }));

vi.mock("@/hooks/store/use-issues", () => ({ useIssues: () => ({ issuesFilter: tab.app.issue.projectIssuesFilter }) }));
vi.mock("@/hooks/store/use-project-state", () => ({ useProjectState: () => tab.app.state }));
vi.mock("@/components/issues/archive-issue-modal", () => ({ ArchiveIssueModal: () => null }));
vi.mock("@/components/issues/delete-issue-modal", () => ({ DeleteIssueModal: () => null }));
vi.mock("@/components/issues/issue-modal/modal", () => ({ CreateUpdateIssueModal: () => null }));
const menus: [string, ComponentType<IQuickActionProps>][] = [
  ["project", ProjectIssueQuickActions],
  ["detail", WorkItemDetailQuickActions],
  ["archived", ArchivedIssueQuickActions],
];
it("a real list row owns one menu across responsive layouts, so a context action is sent once", async () => {
  const newTab = vi.spyOn(window, "open").mockReturnValue(null);
  const issue = issueDetailOf("x");
  await act(async () =>
    renderer.render(
      <IssueBlock
        issuesMap={{ x: issue }}
        issueId="x"
        groupId="todo"
        updateIssue={undefined}
        displayProperties={{}}
        canEditProperties={() => true}
        nestingLevel={0}
        isExpanded={false}
        setExpanded={() => {}}
        isCurrentBlockDragging={false}
        setIsCurrentBlockDragging={() => {}}
        canDrag={false}
        quickActions={({ issue: row, parentRef }) => (
          <ProjectIssueQuickActions issue={row} parentRef={parentRef} handleDelete={async () => {}} />
        )}
      />
    )
  );
  expect(document.querySelectorAll('[data-context-menu="true"]')).toHaveLength(1);
  const row = host.querySelector<HTMLElement>('a[id="issue-x"] > div')!;
  await act(async () => row.dispatchEvent(new MouseEvent("contextmenu", { bubbles: true, cancelable: true })));
  const panel = document.querySelector('[data-context-menu="true"]')!;
  const item = [...panel.querySelectorAll("button")].find(
    (button) => button.textContent === "common.actions.open_in_new_tab"
  )!;
  await act(async () => item.focus());
  await press(item, "Enter", true);
  expect(newTab).not.toHaveBeenCalled();
  await press(item, "Enter");
  expect(newTab).toHaveBeenCalledTimes(1);
  expect(panel.closest("[inert]")).not.toBeNull();
  expect(panel.closest('[aria-hidden="true"]')).not.toBeNull();
});
const opened = () => document.querySelector<HTMLElement>('[role="menu"]:not([inert] *)');
const trigger = () => document.querySelector<HTMLButtonElement>("[data-main-menu] > button")!;
async function press(target: Element, key: string, composing = false) {
  await act(async () => {
    const event = new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true, isComposing: composing });
    target.dispatchEvent(event);
    const up = new KeyboardEvent("keyup", { key, bubbles: true, cancelable: true, isComposing: composing });
    target.dispatchEvent(up);
    if (
      !composing &&
      !event.defaultPrevented &&
      !up.defaultPrevented &&
      (key === "Enter" || key === " ") &&
      target instanceof HTMLButtonElement
    )
      target.click();
    await vi.advanceTimersByTimeAsync(50);
  });
}
async function render(Menu: ComponentType<IQuickActionProps>, modal = false) {
  const content = (
    <Menu
      issue={{ ...issueDetailOf("x"), name: "Keyboard issue" }}
      parentRef={{ current: host }}
      handleDelete={async () => {}}
    />
  );
  await act(async () =>
    renderer.render(
      modal ? (
        <ModalCore isOpen>
          <ModalTitle>Outer editor</ModalTitle>
          {content}
        </ModalCore>
      ) : (
        content
      )
    )
  );
}
it.each(menus)(
  "%s real menu is named, has one trigger, supports arrows/IME/Escape and restores focus",
  async (_name, Menu) => {
    await render(Menu);
    expect(document.querySelector("button button")).toBeNull();
    expect(trigger().getAttribute("aria-label")).toBe("common.options: Keyboard issue");
    await act(async () => trigger().focus());
    await press(trigger(), "Enter");
    expect(opened()).not.toBeNull();
    await press(opened()!, "ArrowDown");
    expect(opened()!.getAttribute("aria-activedescendant")).toBeTruthy();
    await press(opened()!, "Escape", true);
    expect(opened()).not.toBeNull();
    await press(opened()!, "Escape");
    expect(opened()).toBeNull();
    expect(document.activeElement).toBe(trigger());
    await press(trigger(), " ");
    expect(opened()).not.toBeNull();
  }
);
it.each(menus)(
  "%s menu in ModalCore leaves the background hidden and closes before the outer dialog",
  async (_name, Menu) => {
    await render(Menu, true);
    await act(async () => trigger().focus());
    await press(trigger(), "Enter");
    expect(host.closest('[aria-hidden="true"], [inert]')).not.toBeNull();
    await press(opened()!, "Escape");
    expect(document.querySelector('[role="dialog"][aria-modal="true"]')).not.toBeNull();
    expect(host.closest('[aria-hidden="true"], [inert]')).not.toBeNull();
  }
);
it.each(menus)(
  "%s actual context menu is inert when closed, owns focus while open, and ignores IME",
  async (_name, Menu) => {
    await render(Menu);
    const panel = document.querySelector<HTMLElement>('[data-context-menu="true"]')!;
    expect(panel.closest("[inert]")).not.toBeNull();
    expect(panel.closest('[aria-hidden="true"]')).not.toBeNull();
    await act(async () => {
      host.tabIndex = 0;
      host.focus();
      host.dispatchEvent(new MouseEvent("contextmenu", { bubbles: true, cancelable: true, clientX: 20, clientY: 20 }));
    });
    expect(panel.closest("[inert]")).toBeNull();
    expect(panel.contains(document.activeElement)).toBe(true);
    await press(document.activeElement!, "ArrowDown");
    expect(panel.contains(document.activeElement)).toBe(true);
    await press(document.activeElement!, "Escape", true);
    expect(panel.closest("[inert]")).toBeNull();
    await press(document.activeElement!, "Escape");
    expect(panel.closest("[inert]")).not.toBeNull();
    expect(panel.closest('[aria-hidden="true"]')).not.toBeNull();
    expect(document.activeElement).toBe(host);
  }
);
