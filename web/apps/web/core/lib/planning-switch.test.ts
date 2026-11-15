/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// M6 P7b spec 2.5, R8: the words of a work item list's options that name a project's planning, each its domain's (the
// display properties of the cycle and the modules, the groupings by the cycle and the module, the filters by them),
// offered only while the project is known and has the domain turned on; every other word always.
import { expect, it } from "vitest";
import { listWordOn, type ListWord, type PlanningDomain } from "./planning-switch";

const switches = (cycles: boolean, modules: boolean) => ({ cycle_view: cycles, module_view: modules });

it.each<{ word: ListWord; kind: string; domain: PlanningDomain }>([
  { word: "cycle", kind: "the display property and the grouping", domain: "cycles" },
  { word: "modules", kind: "the display property", domain: "modules" },
  { word: "module", kind: "the grouping", domain: "modules" },
  { word: "cycle_id", kind: "the filter", domain: "cycles" },
  { word: "module_id", kind: "the filter", domain: "modules" },
])("offers $word, $kind, only while the project has its $domain on", ({ word, domain }) => {
  const cycles = domain === "cycles";
  expect(listWordOn(switches(true, true), word)).toBe(true);
  expect(listWordOn(switches(cycles, !cycles), word)).toBe(true);
  expect(listWordOn(switches(!cycles, cycles), word)).toBe(false);
  expect(listWordOn(switches(false, false), word)).toBe(false);
  expect(listWordOn(undefined, word)).toBe(false);
});

it.each<ListWord>([
  "state",
  "priority",
  "labels",
  "assignees",
  null,
  "label_id",
  "assignee_id",
  "key",
  "sub_issue_count",
])("offers %s whatever the project has on, and while the project is unknown", (word) => {
  expect(listWordOn(switches(false, false), word)).toBe(true);
  expect(listWordOn(undefined, word)).toBe(true);
});
