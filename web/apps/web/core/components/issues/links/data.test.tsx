// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import React, { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueDetailOf, issueLinkOf, issueStoreTab, requestedIssue } from "@/store/issue/fake-issues";
import { IssueLinkCreateUpdateModal } from "../issue-detail/links/create-update-link-modal";
import { IssueLinksCollapsibleContent } from "../issue-detail-widgets/links/content";
import { useLinkOperations } from "../issue-detail-widgets/links/helper";
import { IssueDetailHost } from "../detail-host";
import { useIssueDetailModals } from "@/hooks/store/use-issue-detail";
import { toasts } from "@/lib/fake-toast";
const binding = vi.hoisted(() => ({ tab: undefined as Awaited<ReturnType<typeof issueStoreTab>> | undefined }));
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
vi.mock("@/hooks/store/use-issue-detail", async (original) => ({
  ...(await original<typeof import("@/hooks/store/use-issue-detail")>()),
  useIssueDetail: () => binding.tab!.app.issue.issueDetail,
}));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@makeplane/propel/components/tooltip", () => ({
  Tooltip: ({ children }: React.PropsWithChildren) => children,
}));
vi.mock("@makeplane/propel/components/field", () => ({
  Field: ({ children }: React.PropsWithChildren) => children,
  FieldError: ({ children }: React.PropsWithChildren) => children,
}));
vi.mock("@makeplane/propel/components/input", () => ({
  InputGroup: ({ children }: React.PropsWithChildren) => children,
  Input: ({
    size: _size,
    ...props
  }: Omit<React.InputHTMLAttributes<HTMLInputElement>, "size"> & {
    size?: string;
    ref?: React.Ref<HTMLInputElement>;
  }) => <input {...props} />,
}));
vi.mock("@nerve/propel/button", () => ({
  Button: ({
    loading,
    variant: _variant,
    size: _size,
    ...props
  }: React.ButtonHTMLAttributes<HTMLButtonElement> & { loading?: boolean; variant?: string; size?: string }) => (
    <button {...props} disabled={loading || props.disabled} />
  ),
}));
vi.mock("@nerve/ui", () => ({
  ModalTitle: ({ children }: React.PropsWithChildren) => <h2>{children}</h2>,
  ModalCore: ({ children, isOpen }: React.PropsWithChildren<{ isOpen: boolean }>) =>
    isOpen ? <div role="dialog">{children}</div> : null,
  CustomMenu: Object.assign(({ children }: React.PropsWithChildren) => <div>{children}</div>, {
    MenuItem: ({ children, onClick }: React.PropsWithChildren<{ onClick: () => void }>) => (
      <button onClick={onClick}>{children}</button>
    ),
  }),
}));
let renderer: Root;
let container: HTMLElement;
const close = vi.fn();
let modals: ReturnType<typeof useIssueDetailModals>;
function Host({ id = "x" }: { id?: string }) {
  return (
    <IssueDetailHost workspaceSlug="ws" projectId="A" issueId={id}>
      <Body id={id} />
    </IssueDetailHost>
  );
}
function Body({ id }: { id: string }) {
  modals = useIssueDetailModals();
  const operations = useLinkOperations("ws", "A", id);
  return (
    <>
      <IssueLinksCollapsibleContent workspaceSlug="ws" projectId="A" issueId={id} disabled={false} />
      <IssueLinkCreateUpdateModal
        workspaceSlug="ws"
        projectId="A"
        issueId={id}
        isModalOpen
        handleOnClose={close}
        linkOperations={operations}
      />
    </>
  );
}
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  toasts.length = 0;
  close.mockReset();
  binding.tab = await issueStoreTab();
  for (const id of ["x", "other"])
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
const render = (id = "x") => act(async () => renderer.render(<Host id={id} />));
const fill = (id: string, value: string) =>
  act(async () => {
    const input = container.querySelector<HTMLInputElement>(`#${id}`)!;
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
const submit = () =>
  act(async () =>
    container.querySelector("form")!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
  );
it.each([undefined, "data:image/png;base64,aWNvbg=="])(
  "renders only the native embedded favicon (%s), never the source URL",
  async (favicon) => {
    const { app, nerve } = binding.tab!;
    const link = {
      ...issueLinkOf("link"),
      metadata: { favicon_url: "https://example.test/icon.ico", favicon, error: "blocked" as const },
    };
    const read = await requestedIssue(nerve, () => app.issue.issueDetail.link.fetchLinks("ws", "A", "x"), [
      "GET",
      "/api/v0/issues/x/links",
    ]);
    read.call.answer(json(200, { data: [link] }));
    await read.promise;
    await render();
    expect(container.querySelector("a")?.textContent).toContain(link.url);
    expect(container.querySelector("img")?.getAttribute("src")).toBe(favicon);
  }
);
it.each([200, 500])(
  "waits for confirmation, retains refused input, and separates metadata/write/read failures (read=%s)",
  async (status) => {
    const { nerve } = binding.tab!;
    await render();
    await fill("url", "https://example.test/");
    await fill("title", "Keep my title");
    await submit();
    await until(() => nerve.to("/api/v0/issues/x/links").length === 1, "link POST");
    expect(close).not.toHaveBeenCalled();
    await act(async () => nerve.to("/api/v0/issues/x/links")[0]!.answer(problem(403, "forbidden")));
    expect(close).not.toHaveBeenCalled();
    expect(container.querySelector<HTMLInputElement>("#title")?.value).toBe("Keep my title");
    await submit();
    await until(() => nerve.to("/api/v0/issues/x/links").length === 2, "retry POST");
    const call = nerve.to("/api/v0/issues/x/links")[1]!;
    expect(call.body).toEqual({ url: "https://example.test/", title: "Keep my title" });
    await act(async () => call.answer(json(201, { ...issueLinkOf("link"), metadata: { error: "failed" } })));
    await until(() => nerve.to("/api/v0/issues/x").length === 1, "count verification");
    expect(close).not.toHaveBeenCalled();
    await act(async () =>
      nerve
        .to("/api/v0/issues/x")[0]!
        .answer(status === 200 ? json(200, { ...issueDetailOf("x"), link_count: 9 }) : problem(500, "unavailable"))
    );
    expect(close).toHaveBeenCalledTimes(1);
    if (status === 500) {
      expect(container.querySelector('[role="alert"]')).not.toBeNull();
      await act(async () => {
        [...container.querySelectorAll("button")].find((button) => button.textContent === "common.retry")!.click();
      });
      await until(() => nerve.to("/api/v0/issues/x").length === 2, "read-only retry");
      await act(async () =>
        nerve.to("/api/v0/issues/x")[1]!.answer(json(200, { ...issueDetailOf("x"), link_count: 9 }))
      );
      expect(container.querySelector('[role="alert"]')).toBeNull();
      expect(nerve.calls.filter((entry) => entry.method === "POST")).toHaveLength(2);
    }
  }
);
it("a successful old host's write cannot close the new host or clear its editing data", async () => {
  const { nerve } = binding.tab!;
  await render();
  await fill("url", "https://example.test/");
  await submit();
  await until(() => nerve.to("/api/v0/issues/x/links").length === 1, "old link POST");
  await render("other");
  const current = issueLinkOf("current", "other");
  await act(async () => modals.setIssueLinkData(current));
  await act(async () => nerve.to("/api/v0/issues/x/links")[0]!.answer(json(201, issueLinkOf("old"))));
  await until(() => nerve.to("/api/v0/issues/x").length === 1, "old count");
  await act(async () => nerve.to("/api/v0/issues/x")[0]!.answer(json(200, issueDetailOf("x"))));
  expect(close).not.toHaveBeenCalled();
  expect(modals.issueLinkData).toEqual(current);
  expect(toasts).toHaveLength(0);
});
