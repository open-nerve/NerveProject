// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { issueDetailOf, issueStoreTab } from "@/store/issue/fake-issues";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import type { DateRangeDropdown } from "@/components/dropdowns/date-range";
import { IssueProperties } from "./issue-layouts/properties/all-properties";
import { SubIssuesListItemProperties } from "./issue-detail-widgets/sub-issues/issues-list/properties";
import { useIssueOperations } from "./issue-detail/use-issue-operations";
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "range" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/use-project-state", () => ({ useProjectState: () => tab.app.state }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => ({ workspaceSlug: "ws" }),
  useNavigate: () => vi.fn(),
}));
// Calendar selection is the input boundary; actual property consumers and native write queue remain real.
vi.mock("@/components/dropdowns/date-range", () => ({
  DateRangeDropdown: (props: ComponentProps<typeof DateRangeDropdown>) => (
    <>
      <button onClick={() => props.onSelect?.({ from: new Date(2026, 9, 20), to: new Date(2026, 9, 25) })}>
        range
      </button>
      <button onClick={() => props.onSelect?.(undefined)}>clear</button>
    </>
  ),
}));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let host: HTMLDivElement;
let renderer: ReturnType<typeof createRoot>;
const source = () => ({ ...issueDetailOf("x"), start_date: "2026-10-01", target_date: "2026-10-10" });
function Host({ kind }: { kind: "row" | "child" }) {
  const operations = useIssueOperations("ws", "A", "x");
  const displayProperties = { start_date: true, due_date: true };
  return kind === "row" ? (
    <IssueProperties
      issue={source()}
      displayProperties={displayProperties}
      isReadOnly={false}
      className=""
      activeLayout="list"
      updateIssue={(_project, id, body) => operations.update("ws", "A", id, body)}
    />
  ) : (
    <SubIssuesListItemProperties
      workspaceSlug="ws"
      parentIssueId="parent"
      issueId="x"
      issue={source()}
      displayProperties={displayProperties}
      canEdit
      updateSubIssue={(_ws, _project, _parent, id, body) => operations.update("ws", "A", id, body)}
    />
  );
}
beforeEach(async () => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  tab = await issueStoreTab();
  tab.app.issue.issues.acceptRead(source(), tab.app.issue.captureRead());
  host = document.createElement("div");
  document.body.append(host);
  renderer = createRoot(host);
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  host.remove();
  tab.app.dispose();
  vi.useRealTimers();
});
for (const kind of ["row", "child"] as const)
  for (const input of ["range", "clear"]) {
    it(`${kind} submits ${input} as one atomic range instead of an invalid intermediate date`, async () => {
      await act(async () => renderer.render(<Host kind={kind} />));
      await act(async () =>
        [...host.querySelectorAll("button")].find((button) => button.textContent === input)!.click()
      );
      await act(async () => until(() => tab.nerve.calls.length === 1, "range PATCH"));
      const expected =
        input === "range"
          ? { start_date: "2026-10-20", target_date: "2026-10-25" }
          : { start_date: null, target_date: null };
      expect(tab.nerve.calls[0]!.body).toEqual(expected);
      await act(async () =>
        tab.nerve.calls[0]!.answer(json(200, { ...source(), ...expected, updated_at: "2026-10-30T10:00:00Z" }))
      );
      expect(tab.nerve.calls.filter((call) => call.method === "PATCH")).toHaveLength(1);
      expect(tab.app.issue.issues.getIssueById("x")).toMatchObject(expected);
    });
  }
