/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { createContext, useContext, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from "react";
// nerve imports
import type { Module } from "@nerve/api-client";
// local imports
import { ArchiveModuleModal } from "./archived-modules/modal";
import { DeleteModuleModal } from "./delete-module-modal";
import { CreateUpdateModuleModal } from "./modal";

/** The dialogs of a module that its page opens: those of its quick actions, which change it, archive it or delete it. */
type ModuleDialog = "edit" | "archive" | "delete";

/** What a module's quick actions have of the page they are on (ModuleActionsHost). */
type ModulePage = {
  /** Opens the page's dialog of the module. */
  openDialog: (dialog: ModuleDialog, module: Module) => void;
  /** Whether the page is still mounted: a change sent from it is followed while it is, though its row has gone. */
  pageMounted: { readonly current: boolean };
};

const ModulePageContext = createContext<ModulePage | undefined>(undefined);

/** The modules' page the caller is on (ModuleActionsHost). */
export function useModulePage(): ModulePage {
  const page = useContext(ModulePageContext);
  if (page === undefined) throw new Error("A module's quick actions require the ModuleActionsHost of their page");
  return page;
}

type Props = {
  workspaceSlug: string;
  projectId: string;
  children: ReactNode;
};

/**
 * The host, on a page of a project's modules (their list, their archive, a module's own page), of what a module's
 * quick actions open and follow (P6a spec 2.7, as CycleActionsHost for cycles): the dialogs that change, archive or
 * delete a module, and the page's lifetime, by which its restore is followed. They are the page's, not the row's: a
 * deletion, an archive or a restore takes the module's row away as nerve confirms it (a change too, a rename the page's
 * search no longer finds, say), before the page follows nerve's answer; the host stays, so its dialog closes and says
 * so, and the page goes where the change takes it. Each follows the answer only in the session it was sent in
 * (followInSession), and while the page is the one shown: mounted by the page, the host goes with it (a Back, another
 * project), and nerve's late answer then reaches the store alone.
 */
export function ModuleActionsHost(props: Props) {
  const { workspaceSlug, projectId, children } = props;
  // states
  const [shown, setShown] = useState<{ dialog: ModuleDialog; module: Module; open: boolean }>();
  const mounted = useRef(false);
  useLayoutEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);

  // the page as its rows have it, the same for its lifetime
  const page = useMemo<ModulePage>(
    () => ({ openDialog: (dialog, module) => setShown({ dialog, module, open: true }), pageMounted: mounted }),
    []
  );
  // a dialog closed keeps its module while it fades out
  const close = () => setShown((current) => current && { ...current, open: false });
  const isOpen = (dialog: ModuleDialog) => shown?.open === true && shown.dialog === dialog;

  return (
    <ModulePageContext.Provider value={page}>
      {children}
      <CreateUpdateModuleModal
        isOpen={isOpen("edit")}
        handleClose={close}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        moduleId={shown?.module.id}
      />
      <ArchiveModuleModal
        isOpen={isOpen("archive")}
        handleClose={close}
        module={shown?.module}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
      />
      <DeleteModuleModal
        isOpen={isOpen("delete")}
        handleClose={close}
        module={shown?.module}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
      />
    </ModulePageContext.Provider>
  );
}
