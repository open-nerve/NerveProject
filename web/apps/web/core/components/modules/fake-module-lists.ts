/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { observable, runInAction } from "mobx";
import { vi } from "vitest";
import type { Module, ModuleLink, ModuleListItem, ModuleUpdate, Progress } from "@nerve/api-client";
import { heldChange, type HeldChange } from "@/lib/auth/fake-tab";

// The modules of a project's pages of modules (its list, its archive, a module's preview; P6a spec 2.2, 2.6) as they
// read and change them, for their tests: the module store is a stand-in a test file gives the pages by mocking
// @/hooks/store/use-modules with it (listedModules). Its reads and its changes of a module go to nerve, which settles
// each when the test says.

/** A list of a project's the store holds: its modules that are not archived, or its archive. */
type List = "modules" | "archive";
/** Whom a change of a module's members adds and removes, as a page gives it to the store. */
type MembersChange = { add: string[]; remove: string[] };

/**
 * A stand-in for the module store as the pages of a project's modules read it: each of a project's two lists as nerve
 * last answered its read (show sets one as if it had); a module's row, of either list, by its id, and the module, its
 * own read once nerve has answered one, else its row; a module's progress and its links as nerve last answered their
 * reads. Each read (a project's two lists, a module, its progress, its links) goes to nerve, which settles the last one
 * of its kind when the test says (answer, refuse); what nerve answers is in the store before the read resolves, as the
 * store does. A change of a module (its fields, its members, its restore) goes to nerve too, which settles the last one
 * sent when the test says (sent): answered, the store writes the module nerve gives into its rows, which keep their
 * counts, and into its own read (a restored module's row goes from its project's archive to the front of its list),
 * then the change resolves.
 */
export function listedModules() {
  const lists = observable.map<string, ModuleListItem[]>({}, { deep: false });
  const modules = observable.map<string, Module>({}, { deep: false });
  const progress = observable.map<string, Progress>({}, { deep: false });
  const links = observable.map<string, ModuleLink[]>({}, { deep: false });
  const reads: { what: string; of: string; nerve: HeldChange<unknown> }[] = [];
  const sent: HeldChange<Module>[] = [];
  const read = (what: string, keep?: (of: string, answer: unknown) => void) =>
    vi.fn(async (of: string) => {
      const nerve = heldChange<unknown>();
      reads.push({ what, of, nerve });
      const answer = await nerve.sent;
      if (keep) runInAction(() => keep(of, answer));
      return answer;
    });
  const listed = (list: List) => (of: string, answer: unknown) =>
    lists.set(`${list} ${of}`, answer as ModuleListItem[]);
  const listOf = (list: List, projectId: string | null | undefined) => lists.get(`${list} ${projectId}`);
  const rowOf = (moduleId: string | null | undefined) => [...lists.values()].flat().find((row) => row.id === moduleId);
  const changed = async () => {
    const change = heldChange<Module>();
    sent.push(change);
    const answer = await change.sent;
    runInAction(() => {
      modules.set(answer.id, answer);
      for (const [key, rows] of lists) {
        const at = rows.findIndex((row) => row.id === answer.id);
        const row = rows[at];
        if (row) lists.set(key, rows.with(at, { ...row, ...answer }));
      }
    });
    return answer;
  };
  const restored = async () => {
    const change = heldChange<Module>();
    sent.push(change);
    const answer = await change.sent;
    runInAction(() => {
      const archive = listOf("archive", answer.project_id) ?? [];
      const row = archive.find((one) => one.id === answer.id);
      if (!row) return;
      lists.set(
        `archive ${answer.project_id}`,
        archive.filter((one) => one !== row)
      );
      lists.set(`modules ${answer.project_id}`, [
        { ...row, ...answer },
        ...(listOf("modules", answer.project_id) ?? []),
      ]);
    });
    return answer;
  };
  return {
    getProjectModuleIds: (projectId: string | null | undefined) => listOf("modules", projectId)?.map((row) => row.id),
    getArchivedModuleIds: (projectId: string | null | undefined) => listOf("archive", projectId)?.map((row) => row.id),
    getModuleRow: rowOf,
    getModuleById: (moduleId: string | null | undefined) =>
      (moduleId ? modules.get(moduleId) : undefined) ?? rowOf(moduleId),
    getModuleProgress: (moduleId: string | null | undefined) => (moduleId ? progress.get(moduleId) : undefined),
    getModuleLinks: (moduleId: string | null | undefined) => (moduleId ? links.get(moduleId) : undefined),
    fetchModules: read("modules", listed("modules")),
    fetchArchivedModules: read("archive", listed("archive")),
    fetchModule: read("module", (of, answer) => modules.set(of, answer as Module)),
    fetchProgress: read("progress", (of, answer) => progress.set(of, answer as Progress)),
    fetchLinks: read("links", (of, answer) => links.set(of, answer as ModuleLink[])),
    update: vi.fn<(moduleId: string, body: ModuleUpdate) => Promise<Module>>(changed),
    changeMembers: vi.fn<(moduleId: string, change: MembersChange) => Promise<Module>>(changed),
    unarchive: vi.fn<(moduleId: string) => Promise<Module>>(restored),
    sent,
    /** The project's list is rows, as if nerve had answered its read. */
    show: (list: List, projectId: string, rows: ModuleListItem[]) =>
      runInAction(() => lists.set(`${list} ${projectId}`, rows)),
    /** What the pages have read so far, each as "what of id". */
    asked: () => reads.map((one) => `${one.what} of ${one.of}`),
    /** nerve answers the last read of what. */
    answer: (what: string, body: unknown) =>
      act(async () => reads.findLast((one) => one.what === what)?.nerve.answer(body)),
    /** nerve refuses the last read of what. */
    refuse: (what: string, error: unknown) =>
      act(async () => reads.findLast((one) => one.what === what)?.nerve.refuse(error)),
  };
}
