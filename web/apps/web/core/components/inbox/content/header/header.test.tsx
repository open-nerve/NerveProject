// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { observer } from "mobx-react";
import { runInAction } from "mobx";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { IssueIntake, Project } from "@nerve/api-client";
import { toasts } from "@/lib/fake-toast";
import { acme, beta, detailOf, intakeTab } from "@/store/issue/intake/fake-intake";
import { loadArchivedProjects, loadProjects, projectOf } from "@/store/project/fake-projects";
import { loadWorkspaces } from "@/store/workspace/fake-workspaces";
import { IntakePageStateProvider } from "../../page-state";
import { IntakeMobileHeader } from "./mobile";
import { IntakeHeader } from "./root";
import type { IntakeActions } from "./intake-actions";
import { useIntakeActions } from "./use-intake-actions";

// The header of the selected work item of a project's intake, wide and narrow, and the action model both read (P5a
// spec 2.7, 2.8, R11, R15, R20, R27): what the caller is offered, by his role in the route's project (the permission
// store's own, over acme and its project WEB as nerve lists them to u-me), the work item's status and the project; the
// status's primary action; copying the work item's link (the clipboard's write a stand-in the test lets or refuses).

const held = vi.hoisted(() => ({ copy: vi.fn(), actions: undefined as IntakeActions | undefined }));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/utils", async (original) => ({
  ...(await original<typeof import("@nerve/utils")>()),
  copyUrlToClipboard: held.copy,
}));
vi.mock("@/hooks/store/use-issue-detail", async (original) => ({
  ...(await original<typeof import("@/hooks/store/use-issue-detail")>()),
  useIssueDetail: () => tab.app.issue.issueDetail,
}));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/use-intake", () => ({ useIntake: () => tab.store }));
vi.mock("@/hooks/store/user", () => ({
  useUser: () => tab.app.user,
  useUserPermissions: () => tab.app.user.permission,
}));

let tab: Awaited<ReturnType<typeof intakeTab>>;
let host: HTMLDivElement, root: Root;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(async () => {
  vi.useFakeTimers();
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  toasts.length = 0;
  held.copy.mockReset();
  tab = await intakeTab();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  tab.app.dispose();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

const ADMIN = 20 as const;
const MEMBER = 15 as const;
const GUEST = 5 as const;
const settle = () => act(async () => void (await vi.advanceTimersByTimeAsync(100)));
const intakeOf = (status: IssueIntake["status"], snoozedTill: string | null = null): IssueIntake => ({
  status,
  snoozed_till: snoozedTill,
  duplicate_to_id: null,
});
const later = () => new Date(Date.now() + 3 * 86_400_000).toISOString();
const earlier = () => new Date(Date.now() - 86_400_000).toISOString();

/** The caller's role in WEB, the project's own fields, and the work item i-x of its intake, submitted by creator. */
async function given(
  role: Project["member_role"],
  intake: IssueIntake,
  creator = "u-other",
  project: Partial<Project> = {}
) {
  const web = projectOf("WEB", acme.id, { intake_view: true, member_role: role, ...project });
  const ops = projectOf("OPS", acme.id, { intake_view: true, member_role: GUEST });
  const store = tab.app.projectRoot.project;
  // an archived project is listed apart, as nerve lists it
  await loadProjects(tab.nerve, store, acme, web.archived_at ? [ops] : [web, ops]);
  if (web.archived_at) await loadArchivedProjects(tab.nerve, store, acme, [web]);
  tab.app.issue.issues.acceptRead(
    detailOf("x", { intake, created_by_id: creator, sequence_id: 7 }),
    tab.app.issue.captureRead()
  );
}

/** The selection the action model is given: its moves and the detail's region are the page's, not this test's. */
const selection = {
  leave: () => {},
  neighbour: () => undefined,
  leaving: { current: false },
  region: { current: null },
  navigation: () => null,
};
const Wide = observer(function Wide() {
  const actions = useIntakeActions("acme", "p-web", "i-x", selection);
  held.actions = actions;
  return <IntakeHeader projectId="p-web" issueId="i-x" actions={actions} isSubmitting="saved" selection={selection} />;
});
const Narrow = observer(function Narrow() {
  const actions = useIntakeActions("acme", "p-web", "i-x", selection);
  held.actions = actions;
  return <IntakeMobileHeader issueId="i-x" actions={actions} isSubmitting="saved" selection={selection} />;
});
const draw = (view: "wide" | "narrow") =>
  act(async () =>
    root.render(
      <MemoryRouter>
        <IntakePageStateProvider>{view === "wide" ? <Wide /> : <Narrow />}</IntakePageStateProvider>
      </MemoryRouter>
    )
  );

/** What the header offers, its buttons' and its menu's: the actions' labels, in order. */
async function offeredIn(view: "wide" | "narrow") {
  await draw(view);
  const menu = host.querySelector<HTMLButtonElement>('button[aria-label="common.options"]');
  if (menu)
    await act(async () => {
      menu.focus();
      menu.click();
    });
  await settle();
  return [...document.querySelectorAll("button, a")]
    .map((control) => control.textContent ?? "")
    .filter((text) => text.startsWith("inbox_issue.actions."))
    .map((text) => text.slice("inbox_issue.actions.".length))
    .toSorted();
}
const TRIAGE = ["accept", "decline", "mark_as_duplicate", "copy", "delete"];
// opening the work item is offered on one accepted alone (P5b spec 2.4, R15)
const READ_ACCEPTED = ["copy", "open"];
const READ = ["copy"];

describe("the header's action model (spec 2.8)", () => {
  it.each([
    { who: "an admin", role: ADMIN, status: "pending", offers: [...TRIAGE, "snooze"] },
    { who: "an admin", role: ADMIN, status: "snoozed", till: "later", offers: [...TRIAGE, "unsnooze", "resnooze"] },
    // a snooze whose day has come is pending
    { who: "an admin", role: ADMIN, status: "snoozed", till: "earlier", offers: [...TRIAGE, "snooze"] },
    { who: "an admin", role: ADMIN, status: "accepted", offers: READ_ACCEPTED },
    { who: "an admin", role: ADMIN, status: "declined", offers: READ },
    { who: "an admin", role: ADMIN, status: "duplicate", offers: READ },
    { who: "a member who submitted it", role: MEMBER, mine: true, status: "pending", offers: [...READ, "delete"] },
    {
      who: "a member who submitted it",
      role: MEMBER,
      mine: true,
      status: "snoozed",
      till: "later",
      offers: [...READ, "delete"],
    },
    { who: "a member who submitted it", role: MEMBER, mine: true, status: "declined", offers: READ },
    { who: "a member", role: MEMBER, status: "pending", offers: READ },
    { who: "a guest who submitted it", role: GUEST, mine: true, status: "pending", offers: READ },
    { who: "a guest", role: GUEST, status: "snoozed", till: "later", offers: READ },
    { who: "an admin of an archived project", role: ADMIN, archived: true, status: "pending", offers: READ },
  ] as const)("offers $who, on a work item $status $till, the same actions in both headers", async (cell) => {
    const offers = [...cell.offers].toSorted();
    const till = "till" in cell ? (cell.till === "later" ? later() : earlier()) : null;
    const project = "archived" in cell ? { archived_at: "2026-11-01T00:00:00Z" } : {};
    await given(cell.role, intakeOf(cell.status, till), "mine" in cell ? "u-me" : "u-other", project);
    expect(await offeredIn("wide")).toEqual(offers);
    expect(held.actions?.primary).toBe(offers.includes("accept") ? "accept" : undefined);
    expect(await offeredIn("narrow")).toEqual(offers);
  });

  it("takes a workspace admin who is the project's member for its admin, as the permission store does", async () => {
    await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...acme, role: ADMIN }, beta]);
    await given(MEMBER, intakeOf("pending"));
    expect(await offeredIn("wide")).toEqual([...TRIAGE, "snooze"].toSorted());
    expect(held.actions?.primary).toBe("accept");
  });

  it("reads the route's project, not the router's: its identifier and the caller's role in it", async () => {
    await given(ADMIN, intakeOf("pending"));
    runInAction(() => tab.app.router.setQuery({ workspaceSlug: "acme", projectId: "p-ops" }));
    expect(await offeredIn("wide")).toEqual([...TRIAGE, "snooze"].toSorted());
    expect(host.querySelector("h3")?.textContent).toBe("WEB-7");
  });

  it("says the work item's status by its name, a snooze the days it has to go", async () => {
    await given(ADMIN, intakeOf("declined"));
    await draw("wide");
    expect(host.textContent).toContain("inbox_issue.status.declined.title");
    await given(ADMIN, intakeOf("snoozed", later()));
    await settle();
    expect(host.textContent).toContain("inbox_issue.status.snoozed.description");
  });
});

describe("copying the work item's link", () => {
  it("copies its address by its project's identifier, and says so", async () => {
    held.copy.mockResolvedValue(undefined);
    await given(MEMBER, intakeOf("pending"));
    await draw("wide");
    await act(async () =>
      [...host.querySelectorAll("button")].find((b) => b.textContent === "inbox_issue.actions.copy")!.click()
    );
    await settle();
    expect(held.copy).toHaveBeenCalledExactlyOnceWith("/acme/browse/WEB-7");
    expect(toasts).toEqual([
      { type: "success", title: "common.link_copied", message: "common.link_copied_to_clipboard" },
    ]);
  });

  it("says so when the browser does not let the page write the clipboard", async () => {
    held.copy.mockRejectedValue(new Error("denied"));
    await given(MEMBER, intakeOf("pending"));
    await draw("narrow");
    const menu = host.querySelector<HTMLButtonElement>('button[aria-label="common.options"]')!;
    await act(async () => menu.click());
    await settle();
    await act(async () =>
      [...document.querySelectorAll("button")].find((b) => b.textContent === "inbox_issue.actions.copy")!.click()
    );
    await settle();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "something_went_wrong_please_try_again" }]);
  });
});
