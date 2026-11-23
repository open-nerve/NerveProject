// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { describe, expect, it, vi } from "vitest";
import type { EmptyStateCompact, EmptyStateDetailed } from "@nerve/propel/empty-state";
import { useCommentInput } from "@/components/comments/use-comment-input";
import { json } from "@/lib/auth/fake-nerve";
import type { RootStore } from "@/store/root.store";
import { detailOf, intakeIssueOf, listPath, pageOf } from "@/store/issue/intake/fake-intake";
import {
  answer,
  callTo,
  callsTo,
  detailRegion,
  opened,
  page,
  settled,
  useIntakePageFixture,
} from "../fake-intake-page";

// The detail region of a project's intake page once its selection's own read has answered (P5a spec 2.5–2.7; M5
// design §9): that read owns the detail host and the input the host keeps, so the browser's reconnecting reads the
// work item not again, nor withdraws the host with what the caller was writing; the list reads again, its read its
// own. The page is the fixture's (fake-intake-page.tsx); the detail's body is a comment's box, its text kept by the host.

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", async () => ({
  rootStore: {},
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/propel/empty-state", () => ({
  EmptyStateCompact: (props: ComponentProps<typeof EmptyStateCompact>) => <p>{props.title}</p>,
  EmptyStateDetailed: (props: ComponentProps<typeof EmptyStateDetailed>) => <p>{props.title}</p>,
}));
vi.mock("@/hooks/use-profile-image", () => import("@/lib/fake-profile-image"));
vi.mock("@/components/issues/issue-detail/use-issue-operations", async (original) => ({
  ...(await original<typeof import("@/components/issues/issue-detail/use-issue-operations")>()),
  useIssueOperations: () => ({
    fetch: (_ws: string, _project: string, id: string) =>
      new Promise<void>((resolve) =>
        page.reads.push({
          id,
          answer: (detail) => {
            page.tab.app.issue.issues.acceptRead(detail, page.tab.app.issue.captureRead());
            resolve();
          },
        })
      ),
  }),
}));
vi.mock("@/components/inbox/content/detail", () => ({ IntakeDetail: () => <CommentBox /> }));
vi.mock("@/components/inbox/filters/menu", () => ({ IntakeFiltersMenu: () => null }));
vi.mock("@/components/inbox/filters/order-by", () => ({ IntakeOrderBy: () => null }));
vi.mock("@/components/inbox/filters/applied", () => ({ IntakeAppliedFilters: () => null }));
vi.mock("@/components/issues/issue-modal/modal", () => ({ CreateUpdateIssueModal: () => null }));

/** The detail's body: a comment's box, whose text the detail host keeps as it keeps a comment's (useCommentInput). */
function CommentBox() {
  const input = useCommentInput("new-comment", "");
  return (
    <textarea aria-label="comment" value={input.value.html} onChange={(event) => input.change(event.target.value)} />
  );
}

useIntakePageFixture();

const [x, y] = ["x", "y"].map((name) => intakeIssueOf(name));
const comment = () => detailRegion()?.querySelector("textarea") ?? null;
/** Writes text into the comment's box, as the browser's input does. */
const write = (text: string) =>
  act(async () => {
    const box = comment()!;
    Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, "value")!.set!.call(box, text);
    box.dispatchEvent(new Event("input", { bubbles: true }));
  });

describe("the detail region once its read has answered", () => {
  it("reads the work item not again as the browser reconnects: the host stays, with the comment being written", async () => {
    await opened([x, y], detailOf("x"));
    await write("Seen twice");
    const box = comment();
    expect(box?.value).toBe("Seen twice");
    // past SWR's deduping of the read made as the selection began, the browser is back online
    await act(async () => void (await vi.advanceTimersByTimeAsync(5_000)));
    await act(async () => void window.dispatchEvent(new Event("online")));
    await settled();
    // the list reads again, its reconnect read its own (spec 2.5); the work item does not
    await answer(callTo("GET", listPath("p-web"), 2), json(200, pageOf([x, y])));
    expect(page.reads.map((read) => read.id)).toEqual(["i-x"]);
    expect(callsTo("GET", "/api/v0/issues/i-x")).toEqual([]);
    expect(comment()).toBe(box);
    expect(comment()?.value).toBe("Seen twice");
  });
});
