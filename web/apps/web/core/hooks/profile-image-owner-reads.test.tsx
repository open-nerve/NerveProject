// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { observer } from "mobx-react";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { StoreContext } from "@/lib/store-context";
import { answered, json, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { assetTab } from "@/store/asset/fake-assets";
import type { ProfileImageSource } from "@/store/asset/profile-images";
import { loadWorkspaces } from "@/store/workspace/fake-workspaces";
import { loadProjects } from "@/store/project/fake-projects";
import { accountOf, userOf } from "@/store/member/fake-users";
import { membershipOf } from "@/store/member/workspace/fake-members";
import { profileOf, seedAccount } from "@/store/user/fake-account";
import type { RootStore } from "@/store/root.store";
import { useProfileImage, type ProfileImageView } from "./use-profile-image";
import { useWorkspacesFetch } from "./use-workspaces-fetch";
import { useWorkspaceMembersFetch } from "./use-workspace-members-fetch";
import { useCurrentUserFetch } from "./use-current-user-fetch";
import { useProjectRead } from "./use-project-read";

vi.mock("@/lib/auth/api-client", () => ({
  tokenManager: {
    start: async () => {},
    state: { status: "signed-in", loginId: "owner-read-session" },
    subscribe: () => () => {},
  },
  publicClient: {},
}));
vi.mock("@/lib/store-context", async () => ({
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
let tab: Awaited<ReturnType<typeof assetTab>>;
let renderer: Root;
let container: HTMLDivElement;
let view: ProfileImageView;
let cache: Map<string, never>;
const Image = observer(function Image({ source }: { source: ProfileImageSource }) {
  view = useProfileImage(source);
  return view.url ? <img src={view.url} alt="profile" /> : <span>placeholder</span>;
});
// The page's canonical hooks and actual host calls; none of these fetchers is an image test double.
function PageRead({ source }: { source: ProfileImageSource }) {
  useWorkspacesFetch(source.kind === "workspace");
  useWorkspaceMembersFetch(source.kind === "member" ? tab.workspace : null);
  useCurrentUserFetch(source.kind === "self");
  useProjectRead(source.kind === "project" ? source.projectID : undefined);
  return null;
}
async function render(source: ProfileImageSource, page: boolean, image = true) {
  await act(async () =>
    renderer.render(
      <SWRConfig value={{ provider: () => cache, revalidateOnMount: true }}>
        <StoreContext.Provider value={tab.app}>
          {page && <PageRead source={source} />}
          {image && <Image source={source} />}
        </StoreContext.Provider>
      </SWRConfig>
    )
  );
  await act(async () => vi.advanceTimersByTimeAsync(0));
}
const expiry = () => new Date(Date.now() + 60_000).toISOString();
const user = () => ({ ...accountOf("caller"), avatar_url: "/static/image", avatar_url_expires_at: expiry() });
const member = () =>
  membershipOf(
    "bob",
    {
      workspace_id: tab.workspace.id,
      member: { ...userOf("bob"), avatar_url: "/static/image", avatar_url_expires_at: expiry() },
    },
    tab.workspace.slug
  );
const project = () => ({ ...tab.project, cover_image_url: "/static/image", cover_image_url_expires_at: expiry() });
const sourceOf = (kind: ProfileImageSource["kind"]): ProfileImageSource =>
  kind === "self"
    ? { kind, image: "avatar" }
    : kind === "workspace"
      ? { kind, workspaceID: tab.workspace.id }
      : kind === "project"
        ? { kind, workspaceID: tab.workspace.id, projectID: tab.project.id }
        : { kind, workspaceID: tab.workspace.id, memberID: "u-bob" };
function path(source: ProfileImageSource) {
  return source.kind === "self"
    ? "/api/v0/me"
    : source.kind === "workspace"
      ? "/api/v0/workspaces"
      : source.kind === "project"
        ? `/api/v0/projects/${source.projectID}`
        : `/api/v0/workspaces/${tab.workspace.slug}/members`;
}
function body(source: ProfileImageSource) {
  return source.kind === "self"
    ? user()
    : source.kind === "workspace"
      ? { data: [tab.workspace] }
      : source.kind === "project"
        ? project()
        : { data: [member()] };
}
function fetchOwner(source: ProfileImageSource): Promise<unknown> {
  return source.kind === "self"
    ? tab.app.user.fetchCurrentUser()
    : source.kind === "workspace"
      ? tab.app.workspaceRoot.fetchWorkspaces()
      : source.kind === "project"
        ? tab.app.projectRoot.project.fetchProject(source.projectID)
        : tab.app.memberRoot.workspace.fetchWorkspaceMembers(tab.workspace);
}
const queries = (source: ProfileImageSource) => tab.nerve.calls.filter((call) => call.path === path(source));
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers({ now: Date.parse("2026-11-04T10:00:00Z") });
  tab = await assetTab();
  tab.workspace = { ...tab.workspace, logo_url: "/static/image", logo_url_expires_at: expiry() };
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace]);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [project()]);
  await seedAccount(tab.nerve, tab.app.user, user());
  await answered(
    tab.nerve,
    () => tab.app.memberRoot.workspace.fetchWorkspaceMembers(tab.workspace),
    ["GET", `/api/v0/workspaces/${tab.workspace.slug}/members`],
    { data: [member()] },
    "fixture member"
  );
  tab.nerve.calls.length = 0;
  tab.nerve.respond = (call) => {
    if (call.path === "/api/v0/me/profile") call.answer(json(200, profileOf()));
  };
  cache = new Map<string, never>();
  container = document.createElement("div");
  document.body.append(container);
  renderer = createRoot(container);
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  tab.app.dispose();
  container.remove();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
it.each(["self", "workspace", "project", "member"] as const)(
  "PC3 %s page and image share one owner request",
  async (kind) => {
    const source = sourceOf(kind);
    await render(source, true);
    await act(async () => until(() => queries(source).length > 0 && view.loading, "both views reading"));
    await act(async () => vi.advanceTimersByTimeAsync(0));
    const count = queries(source).length;
    await act(async () => {
      for (const call of queries(source)) call.answer(json(200, body(source)));
    });
    expect(count).toBe(1);
    await act(async () => until(() => !!container.querySelector("img"), "shared current image"));
  }
);
it.each(["self", "workspace", "project", "member"] as const)(
  "PC3 a %s page joining an in-flight image read does not fetch again",
  async (kind) => {
    const source = sourceOf(kind);
    await render(source, false);
    await act(async () => until(() => queries(source).length === 1, "image request"));
    await render(source, true);
    const count = queries(source).length;
    await act(async () => {
      for (const call of queries(source)) call.answer(json(200, body(source)));
    });
    expect(count).toBe(1);
    await act(async () => until(() => !!container.querySelector("img"), "successful shared answer"));
  }
);
it.each(["self", "workspace", "project", "member"] as const)(
  "PC3 a newer successful %s owner read replaces old image proof without stranding it",
  async (kind) => {
    const source = sourceOf(kind);
    await render(source, false);
    await act(async () => until(() => queries(source).length === 1, "old image request"));
    const next = track(fetchOwner(source));
    await act(async () => until(() => queries(source).length === 2, "new host request"));
    await act(async () => queries(source)[1]!.answer(json(200, body(source))));
    await act(async () => until(() => next.settled, "new host answer"));
    await act(async () => queries(source)[0]!.answer(json(200, body(source))));
    await act(async () => until(() => !view.loading, "old image settled"));
    expect(view.error).toBeUndefined();
    expect(container.querySelector("img")?.getAttribute("src")).toBe("/static/image");
  }
);
it.each(["self", "workspace", "project", "member"] as const)(
  "PC3 a late failed %s owner read cannot erase newer successful proof",
  async (kind) => {
    const source = sourceOf(kind);
    await render(source, false);
    const next = track(fetchOwner(source));
    await act(async () => until(() => queries(source).length === 2, "new host request"));
    await act(async () => queries(source)[1]!.answer(json(200, body(source))));
    await act(async () => until(() => next.settled, "new host answer"));
    await act(async () => queries(source)[0]!.answer(problem(403, "forbidden")));
    expect(view.error).toBeUndefined();
    expect(container.querySelector("img")?.getAttribute("src")).toBe("/static/image");
  }
);
it.each(["self", "workspace", "project", "member"] as const)(
  "PC3 a failed newer %s owner read cannot be authorized by an old success; explicit retry recovers",
  async (kind) => {
    const source = sourceOf(kind);
    await render(source, false);
    const next = track(fetchOwner(source));
    await act(async () => until(() => queries(source).length === 2, "new host request"));
    await act(async () => queries(source)[1]!.answer(problem(403, "forbidden")));
    await act(async () => until(() => next.settled, "new refusal"));
    await act(async () => queries(source)[0]!.answer(json(200, body(source))));
    expect(view.error).toBeDefined();
    expect(container.querySelector("img")).toBeNull();
    await act(async () => vi.advanceTimersByTimeAsync(10_000));
    expect(queries(source)).toHaveLength(2);
    const retry = track(view.retry());
    await act(async () => until(() => queries(source).length === 3, "manual retry"));
    await act(async () => queries(source)[2]!.answer(json(200, body(source))));
    await act(async () => until(() => retry.settled, "retry success"));
    expect(view.error).toBeUndefined();
    expect(container.querySelector("img")?.getAttribute("src")).toBe("/static/image");
  }
);
it.each(["self", "workspace", "project", "member"] as const)(
  "PC3 completed %s SWR cache cannot authorize a newly mounted image",
  async (kind) => {
    const source = sourceOf(kind);
    await render(source, false);
    await act(async () => queries(source)[0]!.answer(json(200, body(source))));
    expect(container.querySelector("img")).not.toBeNull();
    await render(source, false, false);
    await render(source, false);
    expect(container.querySelector("img")).toBeNull();
    await act(async () => until(() => queries(source).length === 2, "fresh mount read"));
    await act(async () => queries(source)[1]!.answer(json(200, body(source))));
    expect(container.querySelector("img")?.getAttribute("src")).toBe("/static/image");
  }
);
it("PC3 a restored member scope starts a new read despite an old same-key SWR request", async () => {
  const source = sourceOf("member");
  await render(source, false);
  const old = queries(source)[0]!;
  await act(async () => loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: 5 }]));
  expect(container.querySelector("img")).toBeNull();
  await act(async () => loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace]));
  await act(async () => vi.advanceTimersByTimeAsync(0));
  expect(queries(source)).toHaveLength(2);
  await act(async () => old.answer(json(200, body(source))));
  expect(container.querySelector("img")).toBeNull();
  await act(async () => queries(source)[1]!.answer(json(200, body(source))));
  expect(container.querySelector("img")?.getAttribute("src")).toBe("/static/image");
});
it("PC3 workspace logo authorization survives unrelated project and membership role changes", async () => {
  const source = sourceOf("workspace");
  await render(source, false);
  await act(async () => loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []));
  await act(async () => loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: 5 }]));
  await act(async () => queries(source)[0]!.answer(json(200, body(source))));
  expect(view.error).toBeUndefined();
  expect(container.querySelector("img")?.getAttribute("src")).toBe("/static/image");
});
it.each(["self", "workspace", "project", "member"] as const)(
  "PC3 reconnect cannot retry an invalid %s capability behind the explicit retry control",
  async (kind) => {
    const source = sourceOf(kind);
    await render(source, true);
    const invalid =
      source.kind === "self"
        ? { ...user(), avatar_url_expires_at: "invalid" }
        : source.kind === "workspace"
          ? { data: [{ ...tab.workspace, logo_url_expires_at: "invalid" }] }
          : source.kind === "project"
            ? { ...project(), cover_image_url_expires_at: "invalid" }
            : { data: [{ ...member(), member: { ...member().member, avatar_url_expires_at: "invalid" } }] };
    await act(async () => queries(source)[0]!.answer(json(200, invalid)));
    expect(view.error).toBeDefined();
    expect(container.querySelector("img")).toBeNull();
    await act(async () => {
      window.dispatchEvent(new Event("online"));
      window.dispatchEvent(new Event("focus"));
      document.dispatchEvent(new Event("visibilitychange"));
      await vi.advanceTimersByTimeAsync(10);
    });
    expect(queries(source)).toHaveLength(1);
    expect(view.error).toBeDefined();
    const retry = track(view.retry());
    await act(async () => until(() => queries(source).length === 2, "explicit capability retry"));
    await act(async () => queries(source)[1]!.answer(json(200, body(source))));
    await act(async () => until(() => retry.settled, "explicit retry success"));
    expect(container.querySelector("img")?.getAttribute("src")).toBe("/static/image");
  }
);
