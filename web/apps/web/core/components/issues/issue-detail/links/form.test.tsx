// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { RootStore } from "@/store/root.store";
import type { Problem } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { heldChange } from "@/lib/auth/fake-tab";
import { issueDetailOf } from "@/store/issue/fake-issues";
import { issueSelectionTab, selectionButton } from "@/components/core/modals/fake-issue-selection";
import { toasts } from "@/lib/fake-toast";
import { IssueDetailHost } from "../../detail-host";
import { IssueLinkCreateUpdateModal } from "./create-update-link-modal";
import type { TLinkOperations } from "./types";

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "links" } }, publicClient: {} }));
vi.mock("@/lib/store-context", async () => ({
  rootStore: undefined,
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
let sent: ReturnType<typeof heldChange<void>>;
const close = vi.fn();
const create = vi.fn<TLinkOperations["create"]>();
const update = vi.fn<TLinkOperations["update"]>();
const operations = { create, update };
beforeEach(async () => {
  tab = await issueSelectionTab();
  tab.app.issue.issues.acceptRead(issueDetailOf("x"), tab.app.issue.captureRead());
  sent = heldChange<void>();
  create.mockReset().mockImplementation(() => sent.sent);
  update.mockReset();
  close.mockReset();
  toasts.length = 0;
});
afterEach(async () => tab.close());
const render = (open = true) =>
  tab.render(
    <IssueDetailHost workspaceSlug="ws" projectId="A" issueId="x">
      <IssueLinkCreateUpdateModal
        workspaceSlug="ws"
        projectId="A"
        issueId="x"
        isModalOpen={open}
        handleOnClose={close}
        linkOperations={operations}
      />
    </IssueDetailHost>
  );
const input = (id = "url") => document.querySelector<HTMLInputElement>(`#${id}`)!;
async function fill(value: string, id = "url") {
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input(id), value);
    input(id).dispatchEvent(new Event("input", { bubbles: true }));
  });
}
const submit = () =>
  act(async () =>
    document.querySelector("form")!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
  );
const description = () =>
  (input().getAttribute("aria-describedby")?.split(/\s+/) ?? [])
    .map((id) => document.getElementById(id)?.textContent ?? "")
    .join(" ");
function focusRecord() {
  const result: { invalid: string | null; message: string }[] = [];
  input().addEventListener("focus", () =>
    result.push({ invalid: input().getAttribute("aria-invalid"), message: description() })
  );
  return result;
}
it("refuses whitespace before sending and describes the error before focus without moving focus on typing", async () => {
  await render();
  await fill("  ");
  await act(async () => input("title").focus());
  const focuses = focusRecord();
  await submit();
  expect(create).not.toHaveBeenCalled();
  expect(focuses).toEqual([{ invalid: "true", message: "common.url_is_invalid" }]);
  expect(document.activeElement).toBe(input());
  await act(async () => input("title").focus());
  await fill("example.test");
  expect(document.activeElement).toBe(input("title"));
});
it("trims a bare URL for server normalization, names the modal and excludes Escape/cancel/duplicate submit while pending", async () => {
  await render();
  await fill("  example.test/release  ");
  const dialog = document.querySelector('[role="dialog"]')!;
  expect(document.getElementById(dialog.getAttribute("aria-labelledby")!)?.textContent).toBe("common.add_link");
  await submit();
  await submit();
  expect(create).toHaveBeenCalledExactlyOnceWith({ url: "example.test/release", title: null });
  await act(async () => {
    selectionButton("common.cancel").click();
    input().dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }));
  });
  expect(close).not.toHaveBeenCalled();
  await act(async () => sent.answer());
  expect(close).toHaveBeenCalledOnce();
  expect(toasts).toHaveLength(1);
});
it.each([422, 409])(
  "keeps a %s refusal in the URL field, announces it before focus, and retries once",
  async (status) => {
    await render();
    await fill("https://example.test/");
    await fill("Keep title", "title");
    await act(async () => input("title").focus());
    const focuses = focusRecord();
    await submit();
    const problem: Problem =
      status === 409
        ? { status, code: "issue.link_exists", title: "Conflict" }
        : {
            status,
            code: "validation_failed",
            title: "Invalid",
            errors: [{ field: "url", code: "not_allowed", message: "Invalid URL" }],
          };
    await act(async () => sent.refuse(new ApiError(status, problem)));
    expect(close).not.toHaveBeenCalled();
    expect(input().value).toBe("https://example.test/");
    expect(input("title").value).toBe("Keep title");
    expect(focuses).toHaveLength(1);
    expect(focuses[0]?.invalid).toBe("true");
    expect(focuses[0]?.message).toBeTruthy();
    expect(toasts).toHaveLength(0);
    sent = heldChange<void>();
    await submit();
    expect(create).toHaveBeenCalledTimes(2);
    await act(async () => sent.answer());
    expect(close).toHaveBeenCalledOnce();
    expect(toasts).toHaveLength(1);
  }
);
it("a confirmed command from a closed instance cannot clear or close the reopened form", async () => {
  await render();
  await fill("https://old.test/");
  await submit();
  await render(false);
  await render();
  await fill("https://new.test/");
  await act(async () => sent.answer());
  expect(input().value).toBe("https://new.test/");
  expect(close).not.toHaveBeenCalled();
  expect(toasts).toHaveLength(0);
});
