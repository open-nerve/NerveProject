// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, createContext, useContext, type ComponentProps, type ReactNode } from "react";
import { createRoot, type Root } from "react-dom/client";
import { observer } from "mobx-react";
import {
  MemoryRouter,
  Route,
  Routes,
  useLocation,
  useNavigate,
  useNavigationType,
  useSearchParams,
} from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { IssueDetail } from "@nerve/api-client";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { useIssueDetailInput } from "@/hooks/use-issue-detail-input";
import { acme, detailOf, intakeIssueOf, intakeTab, loadList, open, pageOf } from "@/store/issue/intake/fake-intake";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import type { DeclineIssueModal } from "../modals/decline-issue-modal";
import type { DeleteInboxIssueModal } from "../modals/delete-issue-modal";
import type { IntakeDetail } from "./detail";
import type { IntakeMobileHeader } from "./header/mobile";
import type { IntakeHeader } from "./header/root";
import type { IntakeActions } from "./header/use-intake-actions";
import { InboxContentRoot } from "./root";

// The detail region of a project's intake page (P5a spec 2.5–2.8; re-94 m4): a region named so and focusable, around
// the selected work item, which each selection reads by a read of its own; nerve refusing it, or answering one not in
// the intake, clears the selection (replace) and says why once; any other failure keeps it, with a retry. The header,
// the detail and the modals are stand-ins here, the action model, the detail host, the stores and SWR are real; the
// work item's read (useIssueOperations' fetch) is held until a test answers it.

type Read = { id: string; answer: (detail: IssueDetail) => void; fail: (error: unknown) => void };
const fixture = vi.hoisted(() => ({
  reads: [] as Read[],
  swr: [] as [fetch: unknown, config: unknown][],
  /** Draws a stand-in, by its name, for the work item of issueId, its header's actions, or its modal's submission. */
  draw: (() => null) as (name: string, issueId?: string, actions?: IntakeActions, submit?: () => unknown) => ReactNode,
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/hooks/store/use-issue-detail", async (original) => ({
  ...(await original<typeof import("@/hooks/store/use-issue-detail")>()),
  useIssueDetail: () => tab.app.issue.issueDetail,
}));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/user", () => ({
  useUser: () => tab.app.user,
  useUserPermissions: () => tab.app.user.permission,
}));
vi.mock("@/lib/use-session-swr", async (original) => {
  const real = await original<typeof import("@/lib/use-session-swr")>();
  return {
    ...real,
    useSessionSWR: (...args: Parameters<typeof real.useSessionSWR>) => {
      fixture.swr.push([args[0], args[2]]);
      return real.useSessionSWR(...args);
    },
  };
});
vi.mock("@/components/issues/issue-detail/use-issue-operations", async (original) => ({
  ...(await original<typeof import("@/components/issues/issue-detail/use-issue-operations")>()),
  useIssueOperations: () => ({
    fetch: (_ws: string, _project: string, id: string) =>
      new Promise<void>((resolve, reject) =>
        fixture.reads.push({
          id,
          answer: (detail) => {
            tab.app.issue.issues.acceptRead(detail, tab.app.issue.captureRead());
            resolve();
          },
          fail: reject,
        })
      ),
  }),
}));
// the stand-ins below are drawn by fixture.draw, given once the file's imports are in
vi.mock("./detail", () => ({
  IntakeDetail: (props: ComponentProps<typeof IntakeDetail>) => fixture.draw("detail", props.issueId),
}));
vi.mock("./header/root", () => ({
  IntakeHeader: (props: ComponentProps<typeof IntakeHeader>) => fixture.draw("wide", props.issueId, props.actions),
}));
vi.mock("./header/mobile", () => ({
  IntakeMobileHeader: (props: ComponentProps<typeof IntakeMobileHeader>) =>
    fixture.draw("narrow", props.issueId, props.actions),
}));
vi.mock("../modals/decline-issue-modal", () => ({
  DeclineIssueModal: (props: ComponentProps<typeof DeclineIssueModal>) =>
    fixture.draw("decline", undefined, undefined, props.onSubmit),
}));
vi.mock("../modals/delete-issue-modal", () => ({
  DeleteInboxIssueModal: (props: ComponentProps<typeof DeleteInboxIssueModal>) =>
    fixture.draw("delete", undefined, undefined, props.onSubmit),
}));
vi.mock("../modals/snooze-issue-modal", () => ({ InboxIssueSnoozeModal: () => fixture.draw("snooze") }));
vi.mock("../modals/select-duplicate", () => ({ SelectDuplicateInboxIssueModal: () => fixture.draw("duplicate") }));
vi.mock("@/components/issues/issue-modal/modal", () => ({ CreateUpdateIssueModal: () => fixture.draw("accept") }));

/** The detail, drawn while its work item is held; it says whether it is inside the detail host. */
const Detail = observer(function Detail({ issueId }: Pick<ComponentProps<typeof IntakeDetail>, "issueId">) {
  const input = useIssueDetailInput();
  return tab.app.issue.issues.getIssueDetailById(issueId) ? <p data-body={issueId} data-host={!!input} /> : null;
});
/** A header whose buttons take each action the model offers, drawn while the work item is held. */
const Header = observer(function Header({
  issueId,
  actions,
}: Pick<ComponentProps<typeof IntakeHeader>, "issueId" | "actions">) {
  const name = useContext(HeaderName);
  if (!tab.app.issue.issues.getIssueDetailById(issueId)) return null;
  return (
    <div data-header={name}>
      {actions.offered.map((action) => (
        <button key={action} type="button" onClick={() => actions.take(action)}>
          {`${name} ${action}`}
        </button>
      ))}
    </div>
  );
});
/** Which header a stand-in draws: the wide or the narrow. */
const HeaderName = createContext("");
fixture.draw = (name, issueId, actions, submit) => {
  if (name === "detail") return <Detail issueId={issueId!} />;
  if (actions)
    return (
      <HeaderName.Provider value={name}>
        <Header issueId={issueId!} actions={actions} />
      </HeaderName.Provider>
    );
  // a modal, whose one button submits it
  return (
    <div role="dialog" aria-label={name}>
      <button type="button" onClick={() => void submit?.()}>
        {`confirm ${name}`}
      </button>
    </div>
  );
};

let tab: Awaited<ReturnType<typeof intakeTab>>;
let host: HTMLDivElement, root: Root, cache: Map<string, unknown>;
const held: { where?: { search: string; type: string }; navigate?: ReturnType<typeof useNavigate> } = {};
globalThis.IS_REACT_ACT_ENVIRONMENT = true;

function Page() {
  const [params] = useSearchParams();
  const location = useLocation();
  held.where = { search: location.search, type: useNavigationType() };
  held.navigate = useNavigate();
  const selected = params.get("inboxIssueId");
  return selected ? (
    <InboxContentRoot
      workspaceSlug="acme"
      projectId="p-web"
      inboxIssueId={selected}
      isMobileSidebar={false}
      setIsMobileSidebar={() => {}}
    />
  ) : (
    <p data-nothing />
  );
}
const address = (id: string) => `/acme/projects/p-web/intake?currentTab=open&inboxIssueId=${id}`;

beforeEach(async () => {
  vi.useFakeTimers();
  toasts.length = 0;
  fixture.reads = [];
  fixture.swr = [];
  tab = await intakeTab();
  await loadProjects(tab.nerve, tab.app.projectRoot.project, acme, [
    projectOf("WEB", acme.id, { intake_view: true, member_role: 20 }),
  ]);
  tab.nerve.calls.length = 0;
  cache = new Map();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  await act(async () =>
    root.render(
      <SWRConfig value={{ provider: () => cache as never }}>
        <MemoryRouter initialEntries={[address("i-x")]}>
          <Routes>
            <Route path="/:workspaceSlug/projects/:projectId/intake" element={<Page />} />
          </Routes>
        </MemoryRouter>
      </SWRConfig>
    )
  );
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  tab.app.dispose();
  vi.useRealTimers();
});

/** The selection's read of id, once it is out (the n-th of that work item's). */
async function readOf(id: string, n = 1) {
  await act(async () => until(() => fixture.reads.filter((read) => read.id === id).length >= n, `read ${n} of ${id}`));
  return fixture.reads.filter((read) => read.id === id)[n - 1]!;
}
const settled = () => act(async () => void (await vi.advanceTimersByTimeAsync(50)));
const region = () => host.querySelector<HTMLElement>('section[aria-label="inbox_issue.a11y.detail"]');
const body = () => host.querySelector<HTMLElement>("[data-body]");
const status = () => host.querySelector('[role="status"]');
const alert = () => host.querySelector('[role="alert"]');
const modal = (name: string) => host.querySelector(`[role="dialog"][aria-label="${name}"]`);
const click = (text: string) =>
  act(async () => [...host.querySelectorAll("button")].find((button) => button.textContent === text)!.click());
const select = (id: string) => act(async () => void held.navigate!(address(id)));
const selected = () => new URLSearchParams(held.where!.search).get("inboxIssueId");
/** The work item answered and shown. */
async function shown(id = "i-x", n = 1) {
  (await readOf(id, n)).answer(detailOf(id.slice(2)));
  await settled();
  expect(body()?.dataset.body).toBe(id);
}

describe("InboxContentRoot, the detail region", () => {
  it("is a region named the detail and focusable, around its read's loading and then the work item, in the detail host", async () => {
    await readOf("i-x");
    expect(region()?.getAttribute("tabindex")).toBe("-1");
    expect(region()?.contains(status())).toBe(true);
    expect(status()?.textContent).toBe("common.loading");
    await shown();
    expect(region()?.contains(body())).toBe(true);
    expect(body()?.dataset.host).toBe("true");
    expect(status()).toBeNull();
  });

  it("reads the selected work item by its own key, its mount read off, once each time it is selected", async () => {
    await shown();
    const identity = tab.app.issue.permissionReads.identity(acme.id, ["p-web"], true);
    const reads = fixture.swr.filter(([fetch]) => (fetch as string[] | null)?.[0] === "intake-issue");
    expect(reads.length).toBeGreaterThan(0);
    for (const [fetch, config] of reads) {
      expect(fetch).toEqual(["intake-issue", identity, "acme", "p-web", "i-x"]);
      expect(config).toEqual({ revalidateOnMount: false });
    }
    await select("i-y");
    await shown("i-y");
    // selected again, the work item is read again, though SWR holds its answer
    await select("i-x");
    await shown("i-x", 2);
    expect(fixture.reads.map((read) => read.id)).toEqual(["i-x", "i-y", "i-x"]);
  });

  it("clears the selection (replace) and says why once its own read is refused, never by a refusal SWR keeps (re-94 m4)", async () => {
    (await readOf("i-x")).fail(refusal(404, "issue.not_found"));
    await settled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.issue_not_found" }]);
    expect(held.where).toEqual({ search: "?currentTab=open", type: "REPLACE" });
    expect(host.querySelector("[data-nothing]")).not.toBeNull();

    // selected again: SWR keeps the refusal, the selection waits for its own read, which gives the work item
    await select("i-x");
    await readOf("i-x", 2);
    await settled();
    expect(selected()).toBe("i-x");
    expect(status()?.textContent).toBe("common.loading");
    expect(toasts).toHaveLength(1);
    await shown("i-x", 2);
    expect(toasts).toHaveLength(1);
    expect(held.where?.type).toBe("PUSH");
  });

  it.each<[string, (read: Read) => void]>([
    ["is refused as not found", (read) => read.fail(refusal(404, "issue.not_found"))],
    ["answers it out of the intake", (read) => read.answer(detailOf("x", { intake: null }))],
  ])(
    "takes the work item out of the intake's lists once its read %s, so that the page selects another row",
    async (_, settle) => {
      await act(
        async () =>
          void (await loadList(tab.nerve, tab.store, "p-web", open, pageOf([intakeIssueOf("x"), intakeIssueOf("y")])))
      );
      settle(await readOf("i-x"));
      await settled();
      expect(held.where).toEqual({ search: "?currentTab=open", type: "REPLACE" });
      expect(tab.store.getList("p-web", open)).toEqual(pageOf([intakeIssueOf("y")], null, 1));
    }
  );

  it("clears the selection (replace) and says why once its read answers a work item not in the intake", async () => {
    (await readOf("i-x")).answer(detailOf("x", { intake: null }));
    await settled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.issue_intake_not_found" }]);
    expect(held.where).toEqual({ search: "?currentTab=open", type: "REPLACE" });
  });

  it("shows any other failure with its retry, keeping the selection; the retry reads again", async () => {
    (await readOf("i-x")).fail(refusal(503, "service.unavailable"));
    await settled();
    expect(region()?.contains(alert())).toBe(true);
    expect(selected()).toBe("i-x");
    expect(toasts).toEqual([]);
    await click("common.retry");
    await shown("i-x", 2);
    expect(alert()).toBeNull();
    expect(selected()).toBe("i-x");
  });

  it("is a detail of its own for each selection: a modal open on one is not open on the next", async () => {
    await shown();
    await click("wide decline");
    expect(modal("decline")).not.toBeNull();
    await select("i-y");
    expect(modal("decline")).toBeNull();
    await shown("i-y");
    expect(modal("decline")).toBeNull();
  });

  it("keeps the region and an open modal while a read again withdraws the header and the detail (R34)", async () => {
    await shown();
    await click("wide decline");
    const again = tab.app.issue.issues.refreshAuthoritative("i-x");
    await act(async () => until(() => tab.nerve.to("/api/v0/issues/i-x").length === 1, "the read again"));
    expect(body()).toBeNull();
    expect(host.querySelector("[data-header]")).toBeNull();
    expect(region()).not.toBeNull();
    expect(modal("decline")).not.toBeNull();
    tab.nerve
      .to("/api/v0/issues/i-x")[0]!
      .answer(json(200, detailOf("x", { intake: { status: "declined", snoozed_till: null, duplicate_to_id: null } })));
    await act(async () => void (await again));
    expect(body()?.dataset.body).toBe("i-x");
    expect(modal("decline")).not.toBeNull();
  });

  it("clears the selection (replace) and says why once a later read finds the work item gone", async () => {
    await shown();
    const again = tab.app.issue.issues.refreshAuthoritative("i-x");
    await act(async () => until(() => tab.nerve.to("/api/v0/issues/i-x").length === 1, "the read again"));
    tab.nerve.to("/api/v0/issues/i-x")[0]!.answer(problem(404, "issue.not_found"));
    await act(async () => void (await again));
    await settled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.issue_not_found" }]);
    expect(held.where).toEqual({ search: "?currentTab=open", type: "REPLACE" });
  });

  it("clears the selection (replace) and says nerve's refusal once a later read is refused", async () => {
    await shown();
    const again = tab.app.issue.issues.refreshAuthoritative("i-x").catch(() => undefined);
    await act(async () => until(() => tab.nerve.to("/api/v0/issues/i-x").length === 1, "the read again"));
    tab.nerve.to("/api/v0/issues/i-x")[0]!.answer(problem(403, "forbidden"));
    await act(async () => void (await again));
    await settled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.forbidden" }]);
    expect(held.where).toEqual({ search: "?currentTab=open", type: "REPLACE" });
  });

  it("clears the selection (replace) and says why once a deletion confirmed elsewhere drops the work item", async () => {
    await shown();
    // a deletion that is not this detail's, confirmed without a read again: the work item is dropped, not missing
    const store = tab.app.issue;
    const deleted = store.writes.deleteIssue("i-x", store.scope("acme", "p-web"));
    await act(async () => until(() => tab.nerve.calls.length === 1, "the delete"));
    expect(tab.nerve.calls[0]).toMatchObject({ method: "DELETE", path: "/api/v0/issues/i-x" });
    tab.nerve.calls[0]!.answer(noContent());
    await act(async () => void (await deleted));
    await settled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.issue_not_found" }]);
    expect(held.where).toEqual({ search: "?currentTab=open", type: "REPLACE" });
  });

  it("clears the selection (replace) once the caller's delete is confirmed, never saying it cannot read it", async () => {
    await shown();
    await click("wide delete");
    await click("confirm delete");
    await act(async () => until(() => tab.nerve.calls.length === 1, "the delete"));
    expect(tab.nerve.calls[0]).toMatchObject({ method: "DELETE", path: "/api/v0/intake-issues/i-x" });
    tab.nerve.calls[0]!.answer(noContent());
    await act(async () => until(() => tab.nerve.calls.length === 2, "the read again"));
    expect(tab.nerve.calls[1]).toMatchObject({ method: "GET", path: "/api/v0/issues/i-x" });
    tab.nerve.calls[1]!.answer(problem(404, "issue.not_found"));
    await settled();
    expect(held.where).toEqual({ search: "?currentTab=open", type: "REPLACE" });
    expect(toasts).toEqual([]);
  });
});
