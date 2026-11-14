/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// M6 P7a spec 2.3, R4, R13: a work item's cycle and module options are what it may join, apart from the values it
// holds.
import { expect, it } from "vitest";
import type { Cycle, Module } from "@nerve/api-client";
import { cycleOf, moduleOf, projectOf } from "@/store/project/fake-projects";
import { cycleOptions, moduleOptions } from "./planning-directory";

const project = projectOf("WEB", "ws");
const draft = cycleOf(project, "Draft");
const upcoming = cycleOf(project, "Next", { status: "upcoming" });
const current = cycleOf(project, "Now", { status: "current" });
const ended = cycleOf(project, "Ended", { status: "completed" });
/** Archived after the project's list of the cycles that are not archived was read: the store's copy says so. */
const shelved = cycleOf(project, "Shelved", { status: "upcoming", archived_at: "2026-10-20T00:00:00Z" });
const held = new Map<string, Cycle>([draft, upcoming, current, ended, shelved].map((cycle) => [cycle.id, cycle]));
const cycleById = (cycleId: string) => held.get(cycleId);

it("gives No cycle first, then the cycles a work item may join, in the order of nerve's list", () => {
  expect(cycleOptions([current.id, draft.id, upcoming.id], cycleById)).toEqual([null, current, draft, upcoming]);
});

it("leaves out a cycle that has ended and one archived since the list was read: nerve refuses both", () => {
  expect(cycleOptions([ended.id, current.id, shelved.id, draft.id], cycleById)).toEqual([null, current, draft]);
});

it("leaves out a cycle the store no longer has", () => {
  expect(cycleOptions(["c-WEB-Deleted", draft.id], cycleById)).toEqual([null, draft]);
});

it("gives No cycle alone when no cycle can be joined, and nothing until the project's list is read", () => {
  expect(cycleOptions([ended.id], cycleById)).toEqual([null]);
  expect(cycleOptions([], cycleById)).toEqual([null]);
  expect(cycleOptions(undefined, cycleById)).toBeUndefined();
});

const backlog = moduleOf(project, "Backlog");
const going = moduleOf(project, "Going", { status: "in-progress" });
const done = moduleOf(project, "Done", { status: "completed" });
const dropped = moduleOf(project, "Dropped", { status: "cancelled" });
/** Archived after the project's list of the modules that are not archived was read: the store's copy says so. */
const stored = moduleOf(project, "Stored", { status: "completed", archived_at: "2026-10-20T00:00:00Z" });
const modules = new Map<string, Module>([backlog, going, done, dropped, stored].map((item) => [item.id, item]));
const moduleById = (moduleId: string) => modules.get(moduleId);

it("gives the modules a work item may join, whatever their status, in the order of nerve's list", () => {
  expect(moduleOptions([done.id, backlog.id, dropped.id, going.id], moduleById)).toEqual([
    done,
    backlog,
    dropped,
    going,
  ]);
});

it("leaves out a module archived since the list was read, and one the store no longer has", () => {
  expect(moduleOptions([going.id, stored.id, "m-WEB-Deleted", backlog.id], moduleById)).toEqual([going, backlog]);
});

it("gives no module option until the project's list is read", () => {
  expect(moduleOptions([], moduleById)).toEqual([]);
  expect(moduleOptions(undefined, moduleById)).toBeUndefined();
});
