// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ReactNode } from "react";
import { Route, Routes } from "react-router";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { RootStore } from "@/store/root.store";
import { profilePage } from "@/components/assets/fake-profile-page";
import { waitForPickerDOM } from "@/components/assets/fake-image-picker";
import { answered, json } from "@/lib/auth/fake-nerve";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import { membershipOf } from "@/store/member/workspace/fake-members";
import { userOf } from "@/store/member/fake-users";
import { DescriptionVersionsRoot } from "./root";
import { EditorMentionsRoot } from "@/components/editor/embeds/mentions/root";
vi.mock("@/lib/auth/api-client", () => ({
  tokenManager: {
    start: async () => {},
    state: { status: "signed-in", loginId: "authors" },
    subscribe: () => () => {},
  },
  publicClient: {},
}));
vi.mock("@/lib/store-context", async () => ({
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
// Only open/closed presentation is elided. Host resolution, member facts, capabilities and images are real.
function Frame({ children }: { children: ReactNode }) {
  return <div>{children}</div>;
}
vi.mock("@nerve/ui", () => ({ CustomMenu: Object.assign(Frame, { MenuItem: Frame }) }));
vi.mock("./modal", () => ({ DescriptionVersionsModal: () => null }));
vi.mock("@nerve/propel/popover", () => ({ Popover: Object.assign(Frame, { Button: Frame, Panel: Frame }) }));
let tab: Awaited<ReturnType<typeof profilePage>>;
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  tab = await profilePage();
});
afterEach(async () => {
  await tab.dispose();
  vi.restoreAllMocks();
  vi.useRealTimers();
});
it("history dropdown and embedded mention use their host workspace even when the surrounding route and member cache name another", async () => {
  const other = workspaceOf("other");
  const member = membershipOf(
    "Bob",
    {
      workspace_id: tab.workspace.id,
      member: {
        ...userOf("Bob"),
        avatar_url: "/static/host.png",
        avatar_url_expires_at: new Date(Date.now() + 60_000).toISOString(),
      },
    },
    tab.workspace.slug
  );
  tab.nerve.respond = undefined;
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace, other]);
  const directory = (workspace: typeof other) =>
    answered(
      tab.nerve,
      () => tab.app.memberRoot.workspace.fetchWorkspaceMembers(workspace),
      ["GET", `/api/v0/workspaces/${workspace.slug}/members`],
      {
        data: [
          {
            ...member,
            workspace_id: workspace.id,
            member: {
              ...member.member,
              avatar_url: workspace.id === other.id ? "/static/wrong-route.png" : "/static/host.png",
            },
          },
        ],
      },
      "directory"
    );
  await directory(tab.workspace);
  await directory(other);
  tab.nerve.calls.length = 0;
  tab.nerve.respond = (call) => call.answer(json(200, { data: [member] }));
  await tab.render(
    <Routes location={`/${other.slug}/projects`}>
      <Route
        path="/:workspaceSlug/projects"
        element={
          <>
            <DescriptionVersionsRoot
              workspaceSlug={tab.workspace.slug}
              projectId="A"
              scopeKey="current"
              versions={[{ id: "version", ownerId: member.member.id, lastSavedAt: new Date().toISOString() }]}
              entityInformation={{
                id: "issue",
                createdAt: new Date(),
                createdByDisplayName: "Bob",
                isRestoreDisabled: false,
              }}
              retryList={async () => {}}
              readDescription={async () => "saved"}
              handleRestore={async () => {}}
            />
            <EditorMentionsRoot
              entity_name="user_mention"
              entity_identifier={member.member.id}
              workspaceID={tab.workspace.id}
              workspaceSlug={tab.workspace.slug}
              projectId={undefined}
            />
          </>
        }
      />
    </Routes>
  );
  await waitForPickerDOM(
    () => document.querySelectorAll('img[src="/static/host.png"]').length === 2,
    "host author images"
  );
  expect(document.querySelector('img[src="/static/wrong-route.png"]')).toBeNull();
  expect([...document.querySelectorAll("a")].map((a) => a.getAttribute("href"))).toEqual([
    `/${tab.workspace.slug}/profile/${member.member.id}`,
    `/${tab.workspace.slug}/profile/${member.member.id}`,
  ]);
  expect(tab.nerve.calls.map((call) => call.path)).toEqual([`/api/v0/workspaces/${tab.workspace.slug}/members`]);
  tab.nerve.respond = undefined;
  await act(async () => loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: 5 }, other]));
  expect(document.querySelector('img[src="/static/host.png"]')).toBeNull();
});
