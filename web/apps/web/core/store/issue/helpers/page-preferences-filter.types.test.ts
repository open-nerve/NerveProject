/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A page's preferences hold no key of their own page, and the types say so (P7b spec 2.3, R18): a cycle's page takes
// no cycle_id and a module's page no module_id, from a literal or from any value typed as the project page's filters.
// The generated CycleIssueFilters and ModuleIssueFilters lack the key but do not refuse it, so a page's base, its
// converter and its service are checked against the web's own page types (re-74 F1).
import { expect, it } from "vitest";
import type { ApiClient, ProjectIssueFilters } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import type { CyclePageFilters, ModulePageFilters } from "@/lib/issue-filters";
import { toCycleFilters, toModuleFilters, toProjectFilters } from "@/lib/issue-filters";
import { CyclePreferencesService } from "@/services/planning/cycle-preferences.service";
import { ModulePreferencesService } from "@/services/planning/module-preferences.service";
import type { IIssueRootStore } from "../root.store";
import { PagePreferencesFilter } from "./page-preferences-filter.store";

const { CYCLE, MODULE, PROJECT } = EIssuesStoreType;

/** Never called: each page's base with its service and converter, as the root builds it, and the wrong pairs. */
function pages(root: IIssueRootStore, api: ApiClient) {
  const cycles = new CyclePreferencesService(api);
  const modules = new ModulePreferencesService(api);
  return [
    new PagePreferencesFilter(root, cycles, toCycleFilters, CYCLE),
    new PagePreferencesFilter(root, modules, toModuleFilters, MODULE),
    // @ts-expect-error a cycle's page's base converts with its own converter, not the project page's
    new PagePreferencesFilter<CyclePageFilters>(root, cycles, toProjectFilters, CYCLE),
    // @ts-expect-error the project page's base cannot send its filters through a cycle's page's service
    new PagePreferencesFilter<ProjectIssueFilters>(root, cycles, toProjectFilters, PROJECT),
    // @ts-expect-error as the root builds it: a cycle's page's service with the project page's converter
    new PagePreferencesFilter(root, cycles, toProjectFilters, CYCLE),
    // @ts-expect-error a module's page's base converts with its own converter, not the project page's
    new PagePreferencesFilter<ModulePageFilters>(root, modules, toProjectFilters, MODULE),
    // @ts-expect-error the project page's base cannot send its filters through a module's page's service
    new PagePreferencesFilter<ProjectIssueFilters>(root, modules, toProjectFilters, PROJECT),
    // @ts-expect-error as the root builds it: a module's page's service with the project page's converter
    new PagePreferencesFilter(root, modules, toProjectFilters, MODULE),
  ];
}

/** A value typed as the project page's filters, not a literal: no excess-property check stands in for the type. */
function held(filters: ProjectIssueFilters): [CyclePageFilters, ModulePageFilters] {
  // @ts-expect-error a value of the project page's filters may hold cycle_id
  const cycle: CyclePageFilters = filters;
  // @ts-expect-error a value of the project page's filters may hold module_id
  const module: ModulePageFilters = filters;
  return [cycle, module];
}

it("types each page's filters without its own key, for its base, its converter and its service", () => {
  expect(pages).toBeTypeOf("function");
  const filters: ProjectIssueFilters = { cycle_id: ["none"], module_id: ["none"], priority: ["high"] };
  expect(held(filters)).toEqual([filters, filters]);
  expect([toCycleFilters({ cycle_id__in: "none" }), toModuleFilters({ module_id__in: "none" })]).toEqual([{}, {}]);
});
