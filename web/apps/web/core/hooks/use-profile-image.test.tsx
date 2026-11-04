// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, StrictMode } from "react";
import { createRoot, type Root } from "react-dom/client";
import { observer } from "mobx-react";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { StoreContext } from "@/lib/store-context";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { assetTab } from "@/store/asset/fake-assets";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import { loadProject, loadProjects } from "@/store/project/fake-projects";
import type { ProfileImageSource } from "@/store/asset/profile-images";
import type { RootStore } from "@/store/root.store";
import { useProfileImage, type ProfileImageView } from "./use-profile-image";

vi.mock("@/lib/auth/api-client", () => ({
  tokenManager: {
    start: async () => {},
    state: { status: "signed-in", loginId: "image-session" },
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
const Probe = observer(function Probe({ source }: { source: ProfileImageSource }) {
  view = useProfileImage(source);
  return (
    <>
      {view.url ? (
        <img src={view.url} alt="profile" onError={view.onError} onLoad={view.onLoad} />
      ) : (
        <span>placeholder</span>
      )}
      {view.loading && <span>loading</span>}
      {view.error !== undefined && <span>failed</span>}
    </>
  );
});
let config: { provider: () => Map<string, never>; revalidateOnMount: boolean };
const source = (): ProfileImageSource => ({ kind: "workspace", workspaceID: tab.workspace.id });
const expiry = (after = 60_000) => new Date(Date.now() + after).toISOString();
const render = async (image: ProfileImageSource = source(), count = 1, strict = false) => {
  const children = (
    <SWRConfig value={config}>
      <StoreContext.Provider value={tab.app}>
        {Array.from({ length: count }, (_, index) => (
          <Probe key={index} source={image} />
        ))}
      </StoreContext.Provider>
    </SWRConfig>
  );
  await act(async () => renderer.render(strict ? <StrictMode>{children}</StrictMode> : children));
};
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers({ now: Date.parse("2026-11-04T10:00:00Z") });
  tab = await assetTab();
  tab.workspace = { ...tab.workspace, logo_url: "/static/logo", logo_url_expires_at: expiry() };
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace]);
  tab.nerve.calls.length = 0;
  tab.nerve.respond = (call) => {
    if (call.path === "/api/v0/workspaces") call.answer(json(200, { data: [tab.workspace] }));
  };
  container = document.createElement("div");
  document.body.append(container);
  renderer = createRoot(container);
  config = { provider: () => new Map<string, never>(), revalidateOnMount: true };
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  tab.app.dispose();
  container.remove();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
const shown = () => act(async () => until(() => !!container.querySelector("img"), "authorized image"));

it("render catches expiry even when the background timer has not fired and then refreshes once", async () => {
  await render();
  await shown();
  tab.nerve.respond = undefined;
  vi.setSystemTime(Date.now() + 60_001);
  await render(source(), 2);
  expect(container.querySelector("img")).toBeNull();
  await act(async () => until(() => tab.nerve.calls.length === 2, "render expiry refresh"));
  await act(async () =>
    tab.nerve.calls[1]!.answer(json(200, { data: [{ ...tab.workspace, logo_url_expires_at: "invalid" }] }))
  );
  expect(container.querySelector("img")).toBeNull();
  expect(container.textContent).toContain("failed");
  await act(async () => vi.advanceTimersByTimeAsync(180_000));
  expect(tab.nerve.calls).toHaveLength(2);
});

it("real SWR coalesces mounted images, expires all pixels and refreshes once", async () => {
  await render(source(), 3);
  await shown();
  expect(container.querySelectorAll("img")).toHaveLength(3);
  expect(tab.nerve.calls).toHaveLength(1);
  tab.nerve.respond = undefined;
  await act(async () => vi.advanceTimersByTimeAsync(60_001));
  expect(container.querySelectorAll("img")).toHaveLength(0);
  expect(tab.nerve.calls).toHaveLength(2);
  await act(async () =>
    tab.nerve.calls[1]!.answer(
      json(200, { data: [{ ...tab.workspace, logo_url: "/static/fresh", logo_url_expires_at: expiry() }] })
    )
  );
  await shown();
  expect([...container.querySelectorAll("img")].map((img) => img.getAttribute("src"))).toEqual(
    Array(3).fill("/static/fresh")
  );
});

it.each(["visibilitychange", "focus", "online"])(
  "%s catches delayed background expiry and invalid refreshed expiry does not loop",
  async (event) => {
    await render();
    await shown();
    vi.setSystemTime(Date.now() + 60_001);
    await act(async () => (event === "visibilitychange" ? document : window).dispatchEvent(new Event(event)));
    expect(container.querySelector("img")).toBeNull();
    await act(async () => until(() => tab.nerve.calls.length === 2, "one expiry read"));
    await act(async () => vi.advanceTimersByTimeAsync(180_000));
    expect(tab.nerve.calls).toHaveLength(2);
    expect(container.querySelector("img")).toBeNull();
  }
);

it("one error refresh, old callback ignored, repeated bad image remains absent until explicit retry", async () => {
  let count = 0;
  tab.nerve.respond = (call) =>
    call.answer(json(200, { data: [{ ...tab.workspace, logo_url: `/static/image-${++count}` }] }));
  await render();
  await shown();
  const oldError = view.onError;
  await act(async () => view.onError());
  await shown();
  expect(container.querySelector("img")?.getAttribute("src")).toBe("/static/image-2");
  await act(async () => oldError());
  expect(container.querySelector("img")?.getAttribute("src")).toBe("/static/image-2");
  await act(async () => view.onError());
  expect(container.querySelector("img")).toBeNull();
  await act(async () => vi.advanceTimersByTimeAsync(180_000));
  expect(tab.nerve.calls).toHaveLength(2);
  tab.workspace = { ...tab.workspace, logo_url_expires_at: expiry() };
  await act(async () => view.retry());
  await shown();
  expect(container.querySelector("img")?.getAttribute("src")).toBe("/static/image-3");
});

it("source changes hide old pixels before their new proof and unmount releases StrictMode resources and timers", async () => {
  const other = workspaceOf("other", { logo_url: "/static/other", logo_url_expires_at: expiry() });
  tab.nerve.respond = undefined;
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace, other]);
  tab.nerve.calls.length = 0;
  tab.nerve.respond = (call) => call.answer(json(200, { data: [tab.workspace, other] }));
  await render(source(), 1, true);
  await shown();
  tab.nerve.respond = undefined;
  await render({ kind: "workspace", workspaceID: other.id }, 1, true);
  expect(container.querySelector("img")).toBeNull();
  await act(async () => until(() => tab.nerve.calls.length === 2, "changed source read"));
  await act(async () => tab.nerve.calls[1]!.answer(json(200, { data: [tab.workspace, other] })));
  await shown();
  expect(container.querySelector("img")?.getAttribute("src")).toBe("/static/other");
  await render(source(), 0, true);
  await act(async () => vi.advanceTimersByTimeAsync(180_000));
  expect(tab.nerve.calls).toHaveLength(2);
  tab.nerve.respond = (call) =>
    call.answer(json(200, { data: [{ ...tab.workspace, logo_url_expires_at: expiry() }, other] }));
  await render(source(), 1, true);
  await shown();
  expect(tab.nerve.calls).toHaveLength(3);
});

it("actual ProjectStore directory loss retires old nonmember image; recovery starts a new proof", async () => {
  const project = {
    ...tab.project,
    member_role: null,
    cover_image_url: "/static/cover",
    cover_image_url_expires_at: expiry(),
  };
  tab.nerve.respond = undefined;
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [project]);
  await loadProject(tab.nerve, tab.app.projectRoot.project, project);
  tab.nerve.calls.length = 0;
  const cover: ProfileImageSource = { kind: "project", workspaceID: tab.workspace.id, projectID: project.id };
  await render(cover);
  await act(async () => until(() => tab.nerve.calls.length === 1, "old cover"));
  const old = tab.nerve.calls[0]!;
  await act(async () => loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []));
  expect(container.querySelector("img")).toBeNull();
  await act(async () => loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [project]));
  const details = () => tab.nerve.to(`/api/v0/projects/${project.id}`);
  await act(async () => until(() => details().length === 2, "new cover lifetime"));
  await act(async () => old.answer(json(200, project)));
  expect(container.querySelector("img")).toBeNull();
  await act(async () => details()[1]!.answer(json(200, { ...project, cover_image_url: "/static/fresh" })));
  await shown();
  expect(container.querySelector("img")?.getAttribute("src")).toBe("/static/fresh");
});

it("owner failure stops and retry remains a read; null image does not read until its owner supplies a URL", async () => {
  tab.nerve.respond = undefined;
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, logo_url: null }]);
  tab.nerve.calls.length = 0;
  await render();
  expect(tab.nerve.calls).toHaveLength(0);
  await act(async () => loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace]));
  await act(async () => until(() => tab.nerve.calls.length === 2, "new nonnull capability read"));
  await act(async () => tab.nerve.calls[1]!.answer(problem(403, "forbidden")));
  expect(container.querySelector("img")).toBeNull();
  expect(container.textContent).toContain("failed");
  await act(async () => vi.advanceTimersByTimeAsync(180_000));
  expect(tab.nerve.calls).toHaveLength(2);
  tab.nerve.respond = (call) => call.answer(json(200, { data: [{ ...tab.workspace, logo_url_expires_at: expiry() }] }));
  await act(async () => view.retry());
  await shown();
  expect(tab.nerve.calls.every((call) => call.method === "GET")).toBe(true);
});
