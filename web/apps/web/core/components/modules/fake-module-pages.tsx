/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, useRef, type ReactNode } from "react";
import { observable, runInAction } from "mobx";
import { observer } from "mobx-react";
import { MemoryRouter, Route, Routes, useParams } from "react-router";
import { vi } from "vitest";
import type { Module } from "@nerve/api-client";
import { Where, settle } from "@/components/common/fake-quick-actions";
import { useModules } from "@/hooks/store/use-modules";
import { heldChange, type HeldChange } from "@/lib/auth/fake-tab";
import { ModuleActionsHost } from "./actions-host";
import { ModuleQuickActions } from "./quick-actions";

// The pages of a project's modules as a module's quick actions are on them (P6a spec 2.6, 2.7), for their tests: its
// list (its modules not archived), a module's own page and its archive, each the page of its ModuleActionsHost with a
// row for each module it shows, the module's quick actions in it; and another page, elsewhere. The router's address is
// the test's to read and move (where, fake-quick-actions.tsx, with what a test does on the pages). The modules are a
// stand-in store's (standInModules), which a test file gives the pages by mocking @/hooks/store/use-modules with it.

/**
 * A stand-in for the module store over modules: it gives them by id and lists them as their project's two lists do;
 * its changes (an archive, a restore, a deletion) go to nerve, which settles the last one sent when the test says
 * (answer, refuse); once answered, the store writes it, which takes the module's row away, then the change resolves.
 */
export function standInModules(modules: Module[]) {
  const held = observable.map(
    modules.map((module) => [module.id, module]),
    { deep: false }
  );
  const sent: HeldChange<void>[] = [];
  /** The module of id once nerve has answered its change. */
  const answered = async (id: string): Promise<Module> => {
    const change = heldChange<void>();
    sent.push(change);
    await change.sent;
    const module = held.get(id);
    if (!module) throw new Error("Module not found");
    return module;
  };
  const moved = (archived_at: string | null) =>
    vi.fn(async (id: string): Promise<Module> => {
      const module = { ...(await answered(id)), archived_at };
      runInAction(() => held.set(id, module));
      return module;
    });
  const listed = (archived: boolean) =>
    [...held.values()].filter((module) => (module.archived_at !== null) === archived).map((module) => module.id);
  return {
    getModuleById: (id: string | null | undefined) => (id ? held.get(id) : undefined),
    getProjectModuleIds: () => listed(false),
    getArchivedModuleIds: () => listed(true),
    archive: moved("2026-11-11T09:00:00Z"),
    unarchive: moved(null),
    remove: vi.fn(async (id: string): Promise<void> => {
      await answered(id);
      runInAction(() => held.delete(id));
    }),
    /** nerve answers the change sent last. */
    answer: async () => {
      await act(async () => sent.at(-1)?.answer(undefined));
      await settle();
    },
    /** nerve refuses the change sent last, by error. */
    refuse: async (error: unknown) => {
      await act(async () => sent.at(-1)?.refuse(error));
      await settle();
    },
  };
}

/** A row of a page: the module's name, and its quick actions, whose right-click menu is the row's. */
const Row = observer(function Row({ moduleId }: { moduleId: string }) {
  const { workspaceSlug = "", projectId = "" } = useParams();
  const { getModuleById } = useModules();
  const ref = useRef<HTMLDivElement>(null);
  const module = getModuleById(moduleId);
  if (!module) return null;
  return (
    <div ref={ref} data-row={moduleId}>
      <span>{module.name}</span>
      <ModuleQuickActions parentRef={ref} moduleId={moduleId} projectId={projectId} workspaceSlug={workspaceSlug} />
    </div>
  );
});

/** A list of the project's modules: those not archived, or its archive. */
const List = observer(function List({ archived }: { archived: boolean }) {
  const { projectId } = useParams();
  const { getProjectModuleIds, getArchivedModuleIds } = useModules();
  const ids = (archived ? getArchivedModuleIds : getProjectModuleIds)(projectId) ?? [];
  return ids.map((id) => <Row key={id} moduleId={id} />);
});

function OwnPage() {
  const { moduleId = "" } = useParams();
  return <Row moduleId={moduleId} />;
}

function Page({ children }: { children: ReactNode }) {
  const { workspaceSlug = "", projectId = "" } = useParams();
  return (
    <ModuleActionsHost workspaceSlug={workspaceSlug} projectId={projectId}>
      {children}
    </ModuleActionsHost>
  );
}

/** The pages of the project's modules, at path. */
export function ModulePages({ path }: { path: string }) {
  return (
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route
          path=":workspaceSlug/projects/:projectId/modules"
          element={
            <Page>
              <List archived={false} />
            </Page>
          }
        />
        <Route
          path=":workspaceSlug/projects/:projectId/modules/:moduleId"
          element={
            <Page>
              <OwnPage />
            </Page>
          }
        />
        <Route
          path=":workspaceSlug/projects/:projectId/archives/modules"
          element={
            <Page>
              <List archived />
            </Page>
          }
        />
        <Route path="*" element={<p>elsewhere</p>} />
      </Routes>
      <Where />
    </MemoryRouter>
  );
}
