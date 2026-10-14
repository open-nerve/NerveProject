/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import { useRouteWorkspace } from "./use-route-workspace";

// Which workspace a page of the workspace is about: the hook runs as a plain function, with stand-ins for the route's
// params and for the caller's list, which holds acme and beta, while the router store (not read) names neither.

const acme = workspaceOf("acme");
const beta = workspaceOf("beta");
const route = vi.hoisted((): { params: { workspaceSlug?: string } } => ({ params: {} }));
vi.mock("react-router", () => ({ useParams: () => route.params }));
vi.mock("@/hooks/store/use-workspace", () => ({
  useWorkspace: () => ({
    getWorkspaceBySlug: (slug: string) => [acme, beta].find((workspace) => workspace.slug === slug) ?? null,
  }),
}));

beforeEach(() => {
  route.params = {};
});

describe("useRouteWorkspace", () => {
  it("is the workspace the route names, as the caller's list holds it", () => {
    route.params = { workspaceSlug: "beta" };
    expect(useRouteWorkspace()).toBe(beta);
  });

  it("is none for a route that names no workspace, or one the caller's list does not hold", () => {
    expect(useRouteWorkspace()).toBeNull();
    route.params = { workspaceSlug: "gamma" };
    expect(useRouteWorkspace()).toBeNull();
  });
});
