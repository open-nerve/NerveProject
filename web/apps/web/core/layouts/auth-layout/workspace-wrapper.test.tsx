/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { emptyShown, shown } from "@/lib/fake-controls";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import type { WorkspaceAccess } from "./use-workspace-fetch";
import { WorkspaceAuthWrapper } from "./workspace-wrapper";

// What the workspace wrapper shows for each decision of useWorkspaceFetch (M3 design 7.2, 8.3): the wrapper renders on
// the server, in a router, with that decision, and stand-ins for the buttons (fake-controls.ts), which keep the props
// they were given.

const decided = vi.hoisted(() => {
  const access: { current: WorkspaceAccess } = { current: { kind: "loading" } };
  return access;
});
vi.mock("./use-workspace-fetch", () => ({ useWorkspaceFetch: () => decided.current }));
vi.mock("@/hooks/store/user", () => ({ useUser: () => ({ signOut: vi.fn(), data: { email: "ann@example.com" } }) }));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const PAGES = "<p>the pages of the workspace</p>";

/** Renders the wrapper of the address's workspace, its page a paragraph, with the decision access: gives the markup. */
function render(access: WorkspaceAccess): string {
  decided.current = access;
  return renderToStaticMarkup(
    <MemoryRouter>
      <WorkspaceAuthWrapper>
        <p>the pages of the workspace</p>
      </WorkspaceAuthWrapper>
    </MemoryRouter>
  );
}

beforeEach(() => {
  emptyShown();
});

describe("WorkspaceAuthWrapper", () => {
  it("shows that nerve cannot be reached, retried when the page asks by the list's retry", () => {
    const retry = vi.fn();
    const markup = render({ kind: "unavailable", retry });
    expect(markup).toContain("auth.session_unavailable.title");
    expect(markup).not.toContain(PAGES);
    expect(shown.buttons.map((button) => button.onClick)).toEqual([retry]);
  });

  it("shows a wait while the caller's workspaces are not there", () => {
    const markup = render({ kind: "loading" });
    expect(markup).toContain("animate-pulse");
    expect(markup).not.toContain(PAGES);
  });

  it.each([
    {
      has: "others",
      hasWorkspaces: true,
      links: ["workspace_not_found.go_home", "workspace_not_found.visit_profile"],
      not: "workspace_not_found.create_workspace",
    },
    {
      has: "none",
      hasWorkspaces: false,
      links: ["workspace_not_found.create_workspace"],
      not: "workspace_not_found.go_home",
    },
  ])("shows that the workspace is not found, with where to go when he has $has", ({ hasWorkspaces, links, not }) => {
    const markup = render({ kind: "not-found", hasWorkspaces });
    expect(markup).toContain("workspace_not_found.title");
    for (const link of links) expect(markup).toContain(link);
    expect(markup).not.toContain(not);
    expect(markup).not.toContain(PAGES);
  });

  it("renders the workspace's pages when it is the caller's", () => {
    expect(render({ kind: "ready", workspace: workspaceOf("acme") })).toBe(PAGES);
  });
});
