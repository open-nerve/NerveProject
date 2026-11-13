/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, expect, it, vi } from "vitest";
import type { Project } from "@nerve/api-client";
import { emptyStores, stores } from "@/hooks/store/fake-store-hooks";
import { handed, response } from "@/lib/fake-session-swr";
import { projectOf } from "@/store/project/fake-projects";
import { workspaceOf } from "@/store/workspace/fake-workspaces";

// The project's wrapper fetches its cycles and its modules by the switch predicate the pages' gate reads (P6a spec
// 2.3, R3: planningOn, lib/planning-switch.ts), one predicate for both, under its own decision that the caller is a
// member of the project (nerve's read of it). The predicate is a stand-in here that answers what the test says, so
// the wrapper is seen to follow it, not the project's own fields; use-project-fetch.test.ts has the rest.

const gate = vi.hoisted(() => ({
  on: { cycles: true, modules: true } as Record<"cycles" | "modules", boolean>,
  asked: [] as string[],
}));
vi.mock("@/lib/planning-switch", () => ({
  planningOn: (project: Project, domain: "cycles" | "modules") => {
    gate.asked.push(`${domain} of ${project.id}`);
    return gate.on[domain];
  },
}));
vi.mock("@/lib/use-session-swr", () => import("@/lib/fake-session-swr"));
vi.mock("@/hooks/store/use-project", () => import("@/hooks/store/fake-store-hooks"));
vi.mock("@/hooks/store/use-project-preferences", () => import("@/hooks/store/fake-store-hooks"));
vi.mock("@/hooks/store/use-label", () => import("@/hooks/store/fake-store-hooks"));
vi.mock("@/hooks/store/use-member", () => import("@/hooks/store/fake-store-hooks"));
vi.mock("@/hooks/store/use-project-state", () => import("@/hooks/store/fake-store-hooks"));
vi.mock("@/hooks/store/use-cycles", () => import("@/hooks/store/fake-store-hooks"));
vi.mock("@/hooks/store/use-modules", () => import("@/hooks/store/fake-store-hooks"));
vi.mock("@/hooks/store/use-workspace", () => import("@/hooks/store/fake-store-hooks"));

const { useProjectFetch } = await import("./use-project-fetch");

const acme = workspaceOf("acme");

beforeEach(() => {
  handed.length = 0;
  response.current = {};
  emptyStores();
  stores.workspaces = [acme];
  stores.address = "acme";
  gate.asked.length = 0;
});

/** What the wrapper handed of a member's project as fields make it: the fetch of its cycles, and of its modules. */
function fetchedOf(fields: Partial<Project>) {
  const project = projectOf("WEB", acme.id, fields);
  stores.projects = [project];
  response.current = { data: project };
  useProjectFetch("acme", project.id);
  return { cycles: handed[5]?.[0] ?? null, modules: handed[6]?.[0] ?? null };
}

it.each<{ on: Record<"cycles" | "modules", boolean>; fields: Partial<Project> }>([
  { on: { cycles: false, modules: true }, fields: {} },
  { on: { cycles: true, modules: false }, fields: {} },
  { on: { cycles: true, modules: true }, fields: { cycle_view: false, module_view: false } },
])(
  "fetches a member's cycles and modules as the switch predicate says ($on), whatever the project's fields",
  ({ on, fields }) => {
    gate.on = on;
    expect(fetchedOf(fields)).toEqual({
      cycles: on.cycles ? ["PROJECT_CYCLES", "p-web"] : null,
      modules: on.modules ? ["PROJECT_MODULES", "p-web"] : null,
    });
    expect(gate.asked).toEqual(["cycles of p-web", "modules of p-web"]);
  }
);

it.each<{ when: string; fields: Partial<Project> }>([
  { when: "he sees it and is no member", fields: { member_role: null } },
  { when: "it is archived, though he is its member", fields: { archived_at: "2026-11-01T00:00:00Z" } },
])("fetches neither for a project not a member's ($when), whatever the predicate says", ({ fields }) => {
  gate.on = { cycles: true, modules: true };
  expect(fetchedOf(fields)).toEqual({ cycles: null, modules: null });
});
