// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { Route, Routes } from "react-router";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Workspace, WorkspaceMember } from "@nerve/api-client";
import type { RootStore } from "@/store/root.store";
import { profilePage } from "@/components/assets/fake-profile-page";
import { waitForPickerDOM } from "@/components/assets/fake-image-picker";
import { answered, json } from "@/lib/auth/fake-nerve";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import { membershipOf } from "@/store/member/workspace/fake-members";
import { userOf } from "@/store/member/fake-users";
import { ButtonAvatars } from "./avatar";
import { NameColumn } from "@/components/workspace/settings/member-columns";
vi.mock("@/lib/auth/api-client", () => ({
  tokenManager: {
    start: async () => {},
    state: { status: "signed-in", loginId: "member-images" },
    subscribe: () => () => {},
  },
  publicClient: {},
}));
vi.mock("@/lib/store-context", async () => ({
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
// Keep native image events visible in jsdom; the authorization/read/store path is real.
vi.mock("@makeplane/propel/components/avatar", () => ({
  Avatar: ({ src, alt, fallback }: { src?: string; alt?: string; fallback?: string }) => (
    <span aria-label={alt}>{src ? <img src={src} alt={alt} /> : fallback}</span>
  ),
}));
let tab: Awaited<ReturnType<typeof profilePage>>;
let here: WorkspaceMember;
let elsewhere: Workspace;
const members = (workspace: Workspace, rows: WorkspaceMember[]) =>
  act(async () =>
    answered(
      tab.nerve,
      () => tab.app.memberRoot.workspace.fetchWorkspaceMembers(workspace),
      ["GET", `/api/v0/workspaces/${workspace.slug}/members`],
      { data: rows },
      "members"
    )
  );
const renders = (workspace: Workspace = tab.workspace, suspended = false) =>
  tab.render(
    <Routes location={`/${workspace.slug}/projects`}>
      <Route
        path="/:workspaceSlug/projects"
        element={
          <>
            <ButtonAvatars userIds={here.member.id} />
            <ButtonAvatars userIds={[here.member.id]} />
            <NameColumn
              rowData={{ ...here, is_active: !suspended }}
              workspaceSlug={workspace.slug}
              isAdmin={false}
              currentUser={tab.app.user.data}
              setRemoveMemberModal={() => {}}
            />
          </>
        }
      />
    </Routes>
  );
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  tab = await profilePage();
  elsewhere = workspaceOf("elsewhere");
  here = membershipOf(
    "Bob",
    {
      workspace_id: tab.workspace.id,
      member: {
        ...userOf("Bob"),
        avatar_url: "/static/here.png",
        avatar_url_expires_at: new Date(Date.now() + 60_000).toISOString(),
      },
    },
    tab.workspace.slug
  );
  tab.nerve.respond = undefined;
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace, elsewhere]);
  await members(tab.workspace, [here]);
  await members(elsewhere, [
    { ...here, workspace_id: elsewhere.id, member: { ...here.member, avatar_url: "/static/elsewhere.png" } },
  ]);
  tab.nerve.calls.length = 0;
  tab.nerve.respond = (call) =>
    call.path.endsWith("/members") ? call.answer(json(200, { data: [here] })) : tab.answer(call);
});
afterEach(async () => {
  await tab.dispose();
  vi.restoreAllMocks();
  vi.useRealTimers();
});
it("three consumers reauthorize one current workspace list and ignore another workspace's global avatar", async () => {
  await renders();
  await waitForPickerDOM(
    () => document.querySelectorAll('img[src="/static/here.png"]').length === 3,
    "three current images"
  );
  expect(document.querySelector('img[src="/static/elsewhere.png"]')).toBeNull();
  expect(tab.nerve.calls.filter((call) => call.path.endsWith("/members"))).toHaveLength(1);
});
it("a guest has no other-member images and sends no members read despite the old cache", async () => {
  tab.nerve.respond = undefined;
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: 5 }, elsewhere]);
  tab.nerve.calls.length = 0;
  await renders();
  expect(document.querySelector("img")).toBeNull();
  expect(tab.nerve.calls).toHaveLength(0);
});
it("an absent membership after changing route does not inherit the old workspace or global picture", async () => {
  await renders();
  tab.nerve.respond = undefined;
  await members(elsewhere, []);
  tab.nerve.calls.length = 0;
  await renders(elsewhere);
  expect(document.querySelector("img")).toBeNull();
  expect(tab.nerve.calls).toHaveLength(0);
});
it("expiry removes all old images and coalesces their next reauthorization", async () => {
  await renders();
  await waitForPickerDOM(() => document.querySelectorAll('img[src="/static/here.png"]').length === 3, "initial images");
  tab.nerve.respond = (call) =>
    call.path.endsWith("/members")
      ? call.answer(
          json(200, {
            data: [
              {
                ...here,
                member: {
                  ...here.member,
                  avatar_url: "/static/refreshed.png",
                  avatar_url_expires_at: new Date(Date.now() + 60_000).toISOString(),
                },
              },
            ],
          })
        )
      : tab.answer(call);
  await act(async () => vi.advanceTimersByTimeAsync(60_001));
  await waitForPickerDOM(
    () => document.querySelectorAll('img[src="/static/refreshed.png"]').length === 3,
    "refreshed images"
  );
  expect(document.querySelector('img[src="/static/here.png"]')).toBeNull();
  expect(tab.nerve.calls.filter((call) => call.path.endsWith("/members"))).toHaveLength(2);
});
it("the workspace's inactive row keeps its deactivated marker", async () => {
  await renders(tab.workspace, true);
  await waitForPickerDOM(
    () => document.querySelectorAll('img[src="/static/here.png"]').length === 2,
    "active consumers only"
  );
  expect(document.querySelectorAll('a[href$="/profile/u-Bob"]')).toHaveLength(0);
});
