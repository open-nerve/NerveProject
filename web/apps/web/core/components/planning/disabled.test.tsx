// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { PlanningDomain } from "@/lib/planning-switch";
import { callerInWeb, its, type Caller } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import { PlanningDisabled } from "./disabled";

// A project's cycles or modules turned off (P5a spec 2.5, P6a spec 2.5): the domain's own text and picture, and the
// way to the project's features of the domain, which only its admins take. The caller's role is the permission
// store's own (fake-permissions.ts).

const held = vi.hoisted(() => ({ permissions: undefined as IUserPermissionStore | undefined }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => held.permissions }));

let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
});

const text = () => host.textContent ?? "";
const button = (name: string) => [...host.querySelectorAll("button")].find((one) => one.textContent === name);

/** The caller of a role in acme and in web, shown web's domain turned off. */
async function shown(caller: Caller, domain: PlanningDomain) {
  const { permissions, acme, web } = await callerInWeb(caller.workspaceRole, caller.projectRole, caller.guestViewAll);
  held.permissions = permissions;
  await act(async () =>
    root.render(
      <MemoryRouter initialEntries={[`/${acme.slug}/projects/${web.id}/${domain}`]}>
        <Routes>
          <Route
            path=":workspaceSlug/projects/:projectId/:domain"
            element={<PlanningDisabled workspaceSlug={acme.slug} projectId={web.id} domain={domain} />}
          />
          <Route
            path=":workspaceSlug/settings/projects/:projectId/features/cycles"
            element={<p>the cycles' features</p>}
          />
          <Route
            path=":workspaceSlug/settings/projects/:projectId/features/modules"
            element={<p>the modules' features</p>}
          />
        </Routes>
      </MemoryRouter>
    )
  );
}

it.each([
  { domain: "cycles", key: "cycle", picture: "/disabled-feature/cycles-", features: "the cycles' features" },
  { domain: "modules", key: "module", picture: "/disabled-feature/modules-", features: "the modules' features" },
] as const)(
  "says the project's $domain are turned off, by their own text and picture, and takes its admin to their features",
  async ({ domain, key, picture, features }) => {
    await shown(its.admin, domain);
    expect(text()).toContain(`disabled_project.empty_state.${key}.title`);
    expect(text()).toContain(`disabled_project.empty_state.${key}.description`);
    expect(host.querySelector("img")?.getAttribute("src")).toContain(picture);
    const manage = button(`disabled_project.empty_state.${key}.primary_button.text`);
    expect(manage?.disabled).toBe(false);
    await act(async () => manage?.click());
    expect(text()).toBe(features);
  }
);

it.each([
  { caller: its.admin, may: true },
  { caller: its.memberAdmin, may: true },
  { caller: its.member, may: false },
  { caller: its.guest, may: false },
  { caller: its.guestSeesAll, may: false },
])("offers $caller.who the way to the modules' features: $may", async ({ caller, may }) => {
  await shown(caller, "modules");
  const manage = button("disabled_project.empty_state.module.primary_button.text");
  expect(manage?.disabled).toBe(!may);
  await act(async () => manage?.click());
  expect(text() === "the modules' features").toBe(may);
});
