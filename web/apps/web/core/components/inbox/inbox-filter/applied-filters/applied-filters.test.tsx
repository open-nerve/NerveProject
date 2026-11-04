// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { InboxIssueAppliedFiltersDate } from "./date";
import { InboxIssueAppliedFiltersMember } from "./member";

const changed = vi.hoisted(() => vi.fn());
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/hooks/store/use-project-inbox", () => ({
  useProjectInbox: () => ({
    inboxFilters: { assignees: ["ada", "bob"], created_at: ["2026-10-01;after", "2026-11-01;before"] },
    handleInboxIssueFilters: changed,
  }),
}));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({
    workspace: {
      fetchWorkspaceMembers: () => {
        throw new Error("Empty avatars must not load members");
      },
    },
    getUserDetails: (id: string) => ({ display_name: id === "ada" ? "Ada" : "Bob", avatar_url: "" }),
  }),
}));
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
beforeEach(() => {
  changed.mockClear();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
});
async function activate(button: HTMLButtonElement, key: string) {
  await act(async () => {
    button.focus();
    const down = new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true });
    button.dispatchEvent(down);
    button.dispatchEvent(new KeyboardEvent("keyup", { key, bubbles: true }));
    // jsdom does not implement native keyboard activation; activate only an unhandled native button.
    if (!down.defaultPrevented) button.click();
  });
}
for (const kind of ["member", "date"] as const) {
  it(`${kind}: named native removal buttons preserve the remaining values and never submit their form`, async () => {
    const submitted = vi.fn();
    await act(async () =>
      root.render(
        <form
          onSubmit={(event) => {
            event.preventDefault();
            submitted();
          }}
        >
          {kind === "member" ? (
            <InboxIssueAppliedFiltersMember filterKey="assignees" label="Assignees" />
          ) : (
            <InboxIssueAppliedFiltersDate filterKey="created_at" label="Created" />
          )}
        </form>
      )
    );
    const buttons = [...host.querySelectorAll("button")];
    expect(buttons).toHaveLength(3);
    expect(
      buttons.every(
        (button) => button.type === "button" && button.tabIndex === 0 && !!button.getAttribute("aria-label")
      )
    ).toBe(true);
    expect(buttons[0].getAttribute("aria-label")).not.toBe(buttons[1].getAttribute("aria-label"));
    await activate(buttons[0], "Enter");
    await activate(buttons[2], " ");
    expect(changed.mock.calls).toEqual(
      kind === "member"
        ? [
            ["assignees", ["bob"]],
            ["assignees", undefined],
          ]
        : [
            ["created_at", ["2026-11-01;before"]],
            ["created_at", undefined],
          ]
    );
    expect(submitted).not.toHaveBeenCalled();
  });
}
