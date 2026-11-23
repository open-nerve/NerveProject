// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { runInAction } from "mobx";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { IssueDetail, IssueIntake, IssueLink, Project } from "@nerve/api-client";
import { json } from "@/lib/auth/fake-nerve";
import { StoreContext } from "@/lib/store-context";
import type { RootStore } from "@/store/root.store";
import { issueLinkOf } from "@/store/issue/fake-issues";
import { acme, detailOf, intakeTab } from "@/store/issue/intake/fake-intake";
import { loadArchivedProjects, loadProjects, projectOf } from "@/store/project/fake-projects";
import { waitForPickerDOM } from "@/components/assets/fake-image-picker";
import { IssueDetailHost } from "@/components/issues/detail-host";
import { IntakeDetail } from "./detail";

// The attachments and links of the selected work item of a project's intake (P5b spec 2.3, R10, R11), as its detail
// draws them inside the detail host: its attachments by the intake's rule (the detail's, intakeEditableBy, the content
// view's canWrite the second layer), its links by the contract's issue.update (an admin or a member, whatever its
// status, its intake on or off, the project and the work item not archived; a guest reads them), and none of the work
// item detail's sub-work item and relation buttons, which nerve refuses for a work item in the intake. The stores, the
// content view, the attachments, the links and their modal are real; the detail's other parts are stand-ins.

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", async () => ({
  rootStore: {},
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/hooks/editor/use-issue-images", () => ({ useIssueImages: () => ({ images: undefined }) }));
vi.mock("@/components/issues/title-input", () => ({ IssueTitleInput: () => null }));
vi.mock("@/components/editor/rich-text/description-input", () => ({ DescriptionInput: () => null }));
vi.mock("@/components/issues/issue-detail/reactions", () => ({ IssueReaction: () => null }));
vi.mock("@/components/issues/issue-detail/description-versions", () => ({ IssueDescriptionVersions: () => null }));
vi.mock("@/components/issues/issue-detail/issue-activity", () => ({ IssueActivity: () => null }));
vi.mock("./issue-properties", () => ({ InboxIssueContentProperties: () => null }));

const LINK: IssueLink = { ...issueLinkOf("l-1", "i-x"), title: "Spec", metadata: { title: "Spec page" } };
const ATTACHMENTS = "/api/v0/issues/i-x/attachments";
const LINKS = "/api/v0/issues/i-x/links";

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
  tab = await intakeTab();
  runInAction(() => {
    tab.app.instance.config = {
      product: "Nerve",
      version: "test",
      commit: "test",
      api_version: "v0",
      signup_enabled: true,
      workspace_creation_enabled: true,
      file_size_limit: 10,
    };
  });
  // nerve lists the work item's attachments (none) and its links (one); every other request waits
  tab.nerve.respond = (call) => {
    if (call.method === "GET" && call.path === ATTACHMENTS) call.answer(json(200, { data: [], next_cursor: null }));
    else if (call.method === "GET" && call.path === LINKS) call.answer(json(200, { data: [LINK] }));
  };
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
const intakeOf = (status: IssueIntake["status"]): IssueIntake => ({
  status,
  snoozed_till: status === "snoozed" ? "2099-01-01T00:00:00Z" : null,
  duplicate_to_id: null,
});

/**
 * The detail of i-x, in p-web's intake as status says and submitted by creator, inside its detail host, shown to the
 * caller as role; the project's own fields and the work item's as project and issue say.
 */
async function shown(
  role: Project["member_role"],
  status: IssueIntake["status"],
  creator = "u-other",
  project: Partial<Project> = {},
  issue: Partial<IssueDetail> = {}
) {
  const web = projectOf("WEB", acme.id, { intake_view: true, member_role: role, ...project });
  const store = tab.app.projectRoot.project;
  if (web.archived_at) await loadArchivedProjects(tab.nerve, store, acme, [web]);
  else await loadProjects(tab.nerve, store, acme, [web]);
  tab.app.issue.issues.acceptRead(
    detailOf("x", { intake: intakeOf(status), created_by_id: creator, ...issue }),
    tab.app.issue.captureRead()
  );
  if (issue.link_count) await tab.app.issue.issueDetail.link.fetchLinks("acme", "p-web", "i-x");
  tab.nerve.calls.length = 0;
  await act(async () =>
    root.render(
      <StoreContext.Provider value={tab.app}>
        <IssueDetailHost workspaceSlug="acme" projectId="p-web" issueId="i-x">
          <IntakeDetail
            workspaceSlug="acme"
            projectId="p-web"
            issueId="i-x"
            isSubmitting="saved"
            setTitleStatus={() => {}}
            setDescriptionStatus={() => {}}
          />
        </IssueDetailHost>
      </StoreContext.Provider>
    )
  );
}
const buttons = () => [...document.querySelectorAll("button")];
/** The button whose text is name, the name it is given. */
const named = (name: string) => buttons().find((button) => button.textContent === name);
/** The trigger of a collapsible whose title says title. */
const trigger = (title: string) => buttons().find((button) => button.textContent?.includes(title));
/** The buttons of the links' header: its trigger, and the control at its end, if any. */
const linksHeader = () => [...(trigger("common.links")?.parentElement?.querySelectorAll("button") ?? [])];

/** The caller opens the work item's attachments, and they are read. */
async function openAttachments() {
  const title = trigger("content_files.title");
  expect(title, "the attachments").toBeDefined();
  await act(async () => title!.click());
  await waitForPickerDOM(() => !!named("content_files.refresh") && !named("content_files.refresh")!.disabled, "files");
  // the picker comes with its upload session, a render after the attachments
  await act(async () => vi.advanceTimersByTimeAsync(10));
}
const picker = () => document.querySelector('input[type="file"]');

describe("the intake detail's attachments (P5b spec 2.3, R9)", () => {
  it.each([
    { who: "an admin", role: ADMIN, status: "pending", picker: true },
    { who: "a guest who submitted it", role: GUEST, mine: true, status: "pending", picker: true },
    { who: "a member who submitted it", role: MEMBER, mine: true, status: "snoozed", picker: true },
    // the four ways the intake's rule says no
    { who: "a member", role: MEMBER, status: "pending", picker: false },
    { who: "an admin", role: ADMIN, status: "declined", picker: false },
    { who: "an admin of a project whose intake is off", role: ADMIN, off: true, status: "pending", picker: false },
    { who: "an admin of an archived project", role: ADMIN, archived: true, status: "pending", picker: false },
  ] as const)("lets $who, of a work item $status, add attachments: $picker", async (cell) => {
    await shown(cell.role, cell.status, "mine" in cell ? "u-me" : "u-other", {
      ...("archived" in cell ? { archived_at: "2026-11-01T00:00:00Z" } : {}),
      ...("off" in cell ? { intake_view: false } : {}),
    });
    await openAttachments();
    // the attachments are shown, read, to everyone; a picker to those who may add one
    expect(tab.nerve.to(ATTACHMENTS)).toHaveLength(1);
    expect(document.body.textContent).toContain("content_files.empty");
    expect(!!picker()).toBe(cell.picker);
  });

  it("keeps the attachments collapsed, unread, until the caller opens them", async () => {
    await shown(ADMIN, "pending");
    expect(trigger("content_files.title")).toBeDefined();
    expect(named("content_files.refresh")).toBeUndefined();
    expect(tab.nerve.to(ATTACHMENTS)).toHaveLength(0);
  });
});

describe("the intake detail's links (P5b spec 2.3, R10)", () => {
  const STATUSES = [
    { status: "pending", intake: "on" },
    { status: "snoozed", intake: "on" },
    { status: "declined", intake: "on" },
    { status: "duplicate", intake: "on" },
    { status: "accepted", intake: "on" },
    { status: "pending", intake: "off" },
  ] as const;
  it.each(
    [ADMIN, MEMBER].flatMap((role) =>
      STATUSES.map(({ status, intake }) => ({
        who: role === ADMIN ? "an admin" : "a member who did not submit it",
        role,
        status,
        intake,
      }))
    )
  )("lets $who add a link to a work item $status, its intake $intake", async ({ role, status, intake }) => {
    await shown(role, status, "u-other", intake === "off" ? { intake_view: false } : {});
    expect(named("issue.add.link")).toBeDefined();
  });

  it("lets a guest who submitted the work item read its links, and add, change or delete none", async () => {
    await shown(GUEST, "pending", "u-me", {}, { link_count: 1 });
    expect(named("issue.add.link")).toBeUndefined();
    expect(document.body.textContent).toContain("Spec");
    // no control at the end of the links' header, and the link's menu (change, delete) disabled
    expect(linksHeader()).toHaveLength(1);
    expect(document.querySelector<HTMLButtonElement>('button[aria-label="common.item_options"]')?.disabled).toBe(true);
  });

  it("lets an admin change and delete its links, from the header's control and the link's menu", async () => {
    await shown(ADMIN, "declined", "u-other", {}, { link_count: 1 });
    expect(linksHeader()).toHaveLength(2);
    expect(document.querySelector<HTMLButtonElement>('button[aria-label="common.item_options"]')?.disabled).toBe(false);
  });

  it.each([
    { what: "an archived project", project: { archived_at: "2026-11-01T00:00:00Z" }, issue: {} },
    { what: "an archived work item", project: {}, issue: { archived_at: "2026-11-02T00:00:00Z" } },
  ])("lets an admin of $what add no link", async ({ project, issue }) => {
    await shown(ADMIN, "accepted", "u-other", project, { ...issue, link_count: 1 });
    expect(named("issue.add.link")).toBeUndefined();
    expect(linksHeader()).toHaveLength(1);
  });

  it.each([
    { links: 0, shown: false },
    { links: 1, shown: true },
  ])("shows the links' collapsible while the work item has links ($links): $shown", async ({ links, shown: has }) => {
    await shown(ADMIN, "pending", "u-other", {}, { link_count: links });
    expect(!!trigger("common.links")).toBe(has);
    expect(document.body.textContent?.includes("Spec")).toBe(has);
  });

  it("opens the link's modal from Add link, closes it, and sends a link to the work item's links", async () => {
    await shown(MEMBER, "declined");
    expect(named("issue.add.link")).toBeDefined();
    await act(async () => named("issue.add.link")!.click());
    expect(document.querySelector('[role="dialog"]')).not.toBeNull();
    await act(async () => named("common.cancel")!.click());
    await waitForPickerDOM(() => document.querySelector('[role="dialog"]') === null, "the modal closed");
    await act(async () => named("issue.add.link")!.click());
    const url = document.querySelector<HTMLInputElement>("#url")!;
    await act(async () => {
      Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(url, "example.test/spec");
      url.dispatchEvent(new Event("input", { bubbles: true }));
    });
    await act(async () =>
      document.querySelector("form")!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
    );
    await waitForPickerDOM(() => tab.nerve.to(LINKS).some((call) => call.method === "POST"), "the link sent");
    const sent = tab.nerve.to(LINKS).filter((call) => call.method === "POST");
    expect(sent.map((call) => call.body)).toEqual([{ url: "example.test/spec", title: null }]);
  });
});

describe("the intake detail's other widgets (P5b spec 2.3, R11)", () => {
  it("offers no sub-work item and no relation, which nerve refuses for a work item in the intake", async () => {
    await shown(ADMIN, "pending");
    expect(named("issue.add.link")).toBeDefined();
    expect(buttons().filter((button) => /issue\.add\.(sub_issue|relation)/.test(button.textContent ?? ""))).toEqual([]);
  });
});
