// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, createElement, StrictMode } from "react";
import { createRoot, type Root } from "react-dom/client";
import { SWRConfig, type Cache } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { InvitationPreview } from "@nerve/api-client";
import { FakeNerve, json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { waitForPickerDOM } from "@/components/assets/fake-image-picker";
import { AuthHeader } from "@/components/account/auth-forms/auth-header";
import { EAuthModes } from "@/helpers/authentication.helper";
const wire = vi.hoisted(() => ({ nerve: undefined as FakeNerve | undefined }));
vi.mock("@/lib/auth/api-client", async () => ({
  publicClient: (await import("@nerve/api-client")).createClient({
    baseUrl: "http://nerve.test",
    fetch: (request: Request) => nerve().fetch(request),
  }),
}));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
let renderer: Root, host: HTMLDivElement, cache: Cache<InvitationPreview>;
const preview = (url = "/static/public.png", expires = Date.now() + 60_000): InvitationPreview => ({
  id: "invitation",
  role: 15,
  declined: false,
  workspace_name: "Acme",
  workspace_slug: "acme",
  workspace_logo_url: url,
  workspace_logo_url_expires_at: new Date(expires).toISOString(),
});
const render = (id: string | null = "invitation", token: string | null = "nrv_inv_public", strict = false) =>
  act(async () => {
    const header = createElement(AuthHeader, { invitationId: id, token, authMode: EAuthModes.SIGN_IN });
    renderer.render(
      createElement(
        SWRConfig,
        { value: { provider: () => cache } },
        strict ? createElement(StrictMode, null, header) : header
      )
    );
  });
const count = (n: number) => act(async () => until(() => nerve().calls.length >= n, `${n} invitation reads`));
const answer = (index: number, value: InvitationPreview) =>
  act(async () => requiredCall(index).answer(json(200, value)));
const image = () => host.querySelector('img[alt="aria_labels.projects_sidebar.workspace_logo"]');
function nerve() {
  if (!wire.nerve) throw new Error("Invitation network is not initialized");
  return wire.nerve;
}
function requiredCall(index: number) {
  const found = nerve().calls[index];
  if (!found) throw new Error(`Missing invitation read ${index}`);
  return found;
}
function picture() {
  const found = image();
  if (!found) throw new Error("Missing invitation picture");
  return found;
}
beforeEach(() => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  wire.nerve = new FakeNerve();
  cache = new Map();
  host = document.createElement("div");
  document.body.append(host);
  renderer = createRoot(host);
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  host.remove();
  vi.restoreAllMocks();
  vi.useRealTimers();
});
it("keys only by invitation and token, never sends session authorization, and skips incomplete links", async () => {
  await render(null, "token");
  await render("invitation", null);
  expect(nerve().calls).toHaveLength(0);
  await render();
  await count(1);
  expect(nerve().calls[0]).toMatchObject({
    path: "/api/v0/workspace-invitations/invitation",
    query: { token: "nrv_inv_public" },
    authorization: null,
  });
  await answer(0, preview());
  expect(image()?.getAttribute("src")).toBe("/static/public.png");
  await render("invitation", "nrv_inv_new");
  await count(2);
  expect(image()).toBeNull();
  expect(nerve().calls[1]?.query).toEqual({ token: "nrv_inv_new" });
  expect([...cache.keys()].filter((key) => key.includes("INVITATION_PREVIEW"))).toHaveLength(2);
});
it("expiry removes the old picture and reads a fresh public capability once", async () => {
  await render();
  await count(1);
  await answer(0, preview());
  await act(async () => picture().dispatchEvent(new Event("load")));
  await act(async () => vi.advanceTimersByTimeAsync(60_001));
  await count(2);
  expect(image()).toBeNull();
  await answer(1, preview("/static/renewed.png"));
  expect(image()?.getAttribute("src")).toBe("/static/renewed.png");
  expect(nerve().calls).toHaveLength(2);
});
it("background time is checked on visibility/focus without waiting for a throttled timer", async () => {
  await render();
  await count(1);
  await answer(0, preview());
  vi.setSystemTime(Date.now() + 60_001);
  await act(async () => {
    window.dispatchEvent(new Event("focus"));
    document.dispatchEvent(new Event("visibilitychange"));
  });
  await count(2);
  expect(image()).toBeNull();
  expect(nerve().calls).toHaveLength(2);
});
it("already invalid refreshed expiry stops automatically and offers explicit retry", async () => {
  await render();
  await count(1);
  await answer(0, preview("/static/expired.png", Date.now() - 1));
  await count(2);
  await answer(1, preview("/static/also-expired.png", Date.now() - 2));
  await act(async () => vi.advanceTimersByTimeAsync(120_000));
  expect(image()).toBeNull();
  expect(nerve().calls).toHaveLength(2);
  const retry = [...host.querySelectorAll("button")].find(
    (button) => button.textContent === "image_upload.retry_image"
  );
  expect(retry).toBeDefined();
  if (!retry) throw new Error("Missing image retry");
  await act(async () => retry.click());
  await count(3);
  await answer(2, preview("/static/retried.png"));
  expect(image()?.getAttribute("src")).toBe("/static/retried.png");
});
it("image failures have one automatic retry; an old node cannot spoil the newer picture", async () => {
  await render();
  await count(1);
  await answer(0, preview());
  const old = picture();
  await act(async () => old.dispatchEvent(new Event("error")));
  await count(2);
  await answer(1, preview("/static/new.png"));
  await act(async () => old.dispatchEvent(new Event("error")));
  expect(image()?.getAttribute("src")).toBe("/static/new.png");
  await act(async () => picture().dispatchEvent(new Event("error")));
  expect(image()).toBeNull();
  await act(async () => vi.advanceTimersByTimeAsync(120_000));
  expect(nerve().calls).toHaveLength(2);
});
it("replacing the link aborts its old request, including a return to the same original link", async () => {
  await render();
  await count(1);
  const old = requiredCall(0);
  await render("other", "nrv_inv_other");
  await count(2);
  expect(old.aborted()).toBe(true);
  await render();
  await count(3);
  expect(requiredCall(1).aborted()).toBe(true);
  await answer(2, preview("/static/current.png"));
  await answer(0, preview("/static/late.png"));
  expect(image()?.getAttribute("src")).toBe("/static/current.png");
});
it("unmount aborts the preview and leaves no timer-driven read", async () => {
  await render();
  await count(1);
  await act(async () => renderer.render(null));
  expect(requiredCall(0).aborted()).toBe(true);
  await act(async () => vi.advanceTimersByTimeAsync(120_000));
  expect(nerve().calls).toHaveLength(1);
});
it("a refused link stops retries and removes its old capability", async () => {
  await render();
  await count(1);
  await answer(0, preview());
  await act(async () => picture().dispatchEvent(new Event("error")));
  await count(2);
  await act(async () => requiredCall(1).answer(problem(404, "workspace.invitation_not_found")));
  await act(async () => {
    vi.advanceTimersByTime(120_000);
    window.dispatchEvent(new Event("focus"));
  });
  expect(image()).toBeNull();
  expect(nerve().calls).toHaveLength(2);
});
it("StrictMode replay still completes a current public preview", async () => {
  nerve().respond = (call) => call.answer(json(200, preview()));
  await render("invitation", "nrv_inv_public", true);
  await waitForPickerDOM(() => image()?.getAttribute("src") === "/static/public.png", "strict preview");
  expect(nerve().calls.every((call) => call.authorization === null)).toBe(true);
});

it("only a successful current image load renews automatic retry, including an unchanged capability", async () => {
  const unchanged = preview();
  await render();
  await count(1);
  await answer(0, unchanged);
  await act(async () => picture().dispatchEvent(new Event("error")));
  await count(2);
  await answer(1, unchanged);
  expect(image()?.getAttribute("src")).toBe(unchanged.workspace_logo_url);
  await act(async () => picture().dispatchEvent(new Event("load")));
  await act(async () => picture().dispatchEvent(new Event("error")));
  await count(3);
  await answer(2, preview("/static/healthy.png"));
  expect(image()?.getAttribute("src")).toBe("/static/healthy.png");
});
