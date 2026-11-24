// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { describe, expect, it, vi } from "vitest";
import type { EmptyStateCompact, EmptyStateDetailed } from "@nerve/propel/empty-state";
import { useIssueDetailInput } from "@/hooks/use-issue-detail-input";
import type { RootStore } from "@/store/root.store";
import { detailOf, intakeIssueOf } from "@/store/issue/intake/fake-intake";
import { buttonSaying, detailRegion, opened, page, settled, useIntakePageFixture } from "../fake-intake-page";

// The detail region of a project's intake page says the result of a content command its detail host confirmed (P5b
// spec 2.2, R11): a file uploaded, or deleted, as the work item detail and its peek say it (ConfirmedContentNotice),
// drawn inside the host, whose input the command confirms to. The page is the fixture's (fake-intake-page.tsx); the
// detail's body is a stand-in that confirms a command to the host's input, as an attachment list does once nerve has
// confirmed it (issue-detail-widgets/attachments/content.tsx).

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
vi.mock("@/components/inbox/content/detail", () => ({ IntakeDetail: () => <Files /> }));
vi.mock("@/components/inbox/filters/menu", () => ({ IntakeFiltersMenu: () => null }));
vi.mock("@/components/inbox/filters/order-by", () => ({ IntakeOrderBy: () => null }));
vi.mock("@/components/inbox/filters/applied", () => ({ IntakeAppliedFilters: () => null }));
vi.mock("@/components/issues/issue-modal/modal", () => ({ CreateUpdateIssueModal: () => null }));

/** The detail's body: its files' commands, each confirmed to the detail host's input as nerve confirms it. */
function Files() {
  const inputs = useIssueDetailInput();
  return (
    <>
      <button type="button" onClick={() => inputs?.confirm({ kind: "uploaded", id: "file-1" }, "files")}>
        upload
      </button>
      <button type="button" onClick={() => inputs?.confirm({ kind: "deleted", id: "file-1" }, "files")}>
        delete
      </button>
    </>
  );
}

useIntakePageFixture();

const notice = () => detailRegion()?.querySelector('p[role="status"]')?.textContent;
/** The detail's body confirms the command saying text. */
async function confirmed(text: "upload" | "delete") {
  await act(async () => buttonSaying(text, detailRegion()!)!.click());
  await settled();
}

describe("the detail region's confirmed content notice", () => {
  it("says a file uploaded, then deleted, once its detail host confirms it", async () => {
    await opened([intakeIssueOf("x")], detailOf("x"));
    expect(notice()).toBeUndefined();
    await confirmed("upload");
    expect(notice()).toBe("content_files.confirmed_upload");
    await confirmed("delete");
    expect(notice()).toBe("content_files.confirmed_delete");
  });
});
