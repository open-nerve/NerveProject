// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import React, { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { beforeEach, afterEach, expect, it, vi } from "vitest";
import { issueDetailOf, issueStoreTab } from "@/store/issue/fake-issues";
import { noContent, problem } from "@/lib/auth/fake-nerve";
import { toasts } from "@/lib/fake-toast";
import { DeleteIssueModal } from "./delete-issue-modal";
import en from "../../../../../packages/i18n/src/locales/en/common.json";
import zh from "../../../../../packages/i18n/src/locales/zh-CN/common.json";
const binding = vi.hoisted(() => ({
  tab: undefined as Awaited<ReturnType<typeof issueStoreTab>> | undefined,
  close: vi.fn(),
  locale: "en" as "en" | "zh",
}));
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
vi.mock("@nerve/i18n", () => ({
  useTranslation: () => ({
    t: (key: string, values?: { identifier?: string }) => {
      const locale = binding.locale === "en" ? en : zh;
      const value = key
        .split(".")
        .reduce<unknown>(
          (node, part) => (node && typeof node === "object" ? Reflect.get(node, part) : undefined),
          locale
        );
      return typeof value === "string" ? value.replace("{identifier}", values?.identifier ?? "") : key;
    },
  }),
}));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "ws" }) }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => binding.tab!.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-issues", () => ({
  useIssues: () => ({ issueMap: binding.tab!.app.issue.issues.issuesMap }),
}));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => binding.tab!.app.projectRoot.project }));
vi.mock("@/hooks/store/user", () => ({
  useUser: () => ({ data: { id: "actor" } }),
  useUserPermissions: () => ({ allowPermissions: () => true }),
}));
vi.mock("@nerve/ui", () => ({
  AlertModalCore: ({
    handleSubmit,
    handleClose,
    isSubmitting,
    isOpen,
    content,
  }: {
    handleSubmit: () => Promise<void>;
    handleClose: () => void;
    isSubmitting: boolean;
    isOpen: boolean;
    content: React.ReactNode;
  }) =>
    isOpen ? (
      <>
        <p>{content}</p>
        <button data-action="delete" disabled={isSubmitting} onClick={() => void handleSubmit()}>
          Delete
        </button>
        {/* Exercises a close event delivered before the pending render disables the shared modal. */}
        <button data-action="close" onClick={handleClose}>
          Cancel
        </button>
      </>
    ) : null,
}));
let renderer: Root;
let container: HTMLElement;
beforeEach(async () => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  toasts.length = 0;
  binding.close.mockReset();
  binding.locale = "en";
  binding.tab = await issueStoreTab();
  for (const id of ["one", "two"])
    binding.tab.app.issue.issues.acceptRead(issueDetailOf(id), binding.tab.app.issue.captureRead());
  container = document.createElement("div");
  document.body.append(container);
  renderer = createRoot(container);
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  container.remove();
  binding.tab!.app.dispose();
  vi.useRealTimers();
});
const render = (id: string, open = true) =>
  act(async () =>
    renderer.render(
      <DeleteIssueModal
        isOpen={open}
        handleClose={binding.close}
        data={issueDetailOf(id)}
        onSubmit={() => binding.tab!.app.issue.deleteIssue(id, binding.tab!.app.issue.scope("ws", "A"))}
      />
    )
  );
it("keeps the delete confirmation open and retryable on refusal", async () => {
  await render("one");
  await act(async () => container.querySelector("button")!.click());
  expect(container.querySelector("button")!.disabled).toBe(true);
  await act(async () => binding.tab!.nerve.calls[0]!.answer(problem(403, "forbidden")));
  expect(binding.close).not.toHaveBeenCalled();
  expect(container.querySelector("button")!.disabled).toBe(false);
  expect(toasts).toHaveLength(1);
  expect(toasts[0]?.type).toBe("error");
});
it("closes once after a confirmed deletion", async () => {
  await render("one");
  await act(async () => container.querySelector("button")!.click());
  expect(binding.close).not.toHaveBeenCalled();
  await act(async () => binding.tab!.nerve.calls[0]!.answer(noContent()));
  expect(binding.close).toHaveBeenCalledTimes(1);
});
it("does not close a new issue's modal when the old deletion completes", async () => {
  await render("one");
  await act(async () => container.querySelector("button")!.click());
  await render("two");
  await act(async () => binding.tab!.nerve.calls[0]!.answer(noContent()));
  expect(binding.close).not.toHaveBeenCalled();
  expect(toasts).toHaveLength(0);
});

it.each(["en", "zh"] as const)(
  "states the whole subtree and archived descendants without reading hidden counts (%s)",
  async (locale) => {
    binding.locale = locale;
    await render("one");
    const content = container.querySelector("p")!.textContent;
    expect(content).toContain(locale === "en" ? "all its sub-work items" : "全部子工作项");
    expect(content).toContain(locale === "en" ? "archived descendants" : "已归档的后代");
    expect(binding.tab!.nerve.calls).toHaveLength(0);
  }
);

it("keeps the owner open under a same-turn double submit and close until confirmation", async () => {
  await render("one");
  await act(async () => {
    container.querySelector<HTMLButtonElement>('[data-action="delete"]')!.click();
    container.querySelector<HTMLButtonElement>('[data-action="delete"]')!.click();
    container.querySelector<HTMLButtonElement>('[data-action="close"]')!.click();
  });
  expect(binding.tab!.nerve.calls).toHaveLength(1);
  expect(binding.close).not.toHaveBeenCalled();
  await act(async () => binding.tab!.nerve.calls[0]!.answer(noContent()));
  expect(binding.close).toHaveBeenCalledTimes(1);
});

it("does not close or unlock the reopened same issue when the old deletion answers", async () => {
  await render("one");
  await act(async () => container.querySelector("button")!.click());
  await render("one", false);
  await render("one");
  await act(async () => binding.tab!.nerve.calls[0]!.answer(noContent()));
  expect(binding.close).not.toHaveBeenCalled();
  expect(toasts).toHaveLength(0);
});
