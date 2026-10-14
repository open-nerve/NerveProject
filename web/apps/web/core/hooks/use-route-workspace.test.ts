/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { emptyStores, stores } from "@/hooks/store/fake-store-hooks";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import { useRouteWorkspace } from "./use-route-workspace";

// Which workspace a page of the workspace is about: the hook runs as a plain function, with a stand-in for the route's
// params and fake-store-hooks.ts's for the caller's list, which holds acme and beta, while the router store still
// names acme, the workspace the caller was on (StoreWrapper's effect writes it after the render).

const acme = workspaceOf("acme");
const beta = workspaceOf("beta");
const route = vi.hoisted((): { params: { workspaceSlug?: string } } => ({ params: {} }));
vi.mock("react-router", () => ({ useParams: () => route.params }));
vi.mock("@/hooks/store/use-workspace", () => import("@/hooks/store/fake-store-hooks"));

beforeEach(() => {
  emptyStores();
  stores.workspaces = [acme, beta];
  stores.address = "acme";
  route.params = {};
});

describe("useRouteWorkspace", () => {
  it("is the workspace the route names, as the caller's list holds it, not the one the router store names", () => {
    route.params = { workspaceSlug: "beta" };
    expect(useRouteWorkspace()).toBe(beta);
  });

  it("is none for a route that names no workspace, or one the caller's list does not hold", () => {
    expect(useRouteWorkspace()).toBeNull();
    route.params = { workspaceSlug: "gamma" };
    expect(useRouteWorkspace()).toBeNull();
  });
});
