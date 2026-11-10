/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { createContext, useContext, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from "react";
// nerve imports
import type { Cycle } from "@nerve/api-client";
// local imports
import { ArchiveCycleModal } from "./archived-cycles/modal";
import { CycleDeleteModal } from "./delete-modal";
import { CycleCreateUpdateModal } from "./modal";

/** The dialogs of a cycle that its quick actions open: the one that changes it, archives it or deletes it. */
type CycleDialog = "edit" | "archive" | "delete";

/** What a cycle's quick actions have of the page they are on (CycleActionsHost). */
type CyclePage = {
  /** Opens the page's dialog of the cycle. */
  openDialog: (dialog: CycleDialog, cycle: Cycle) => void;
  /** Whether the page is still mounted: a change sent from it is followed while it is, though its row has gone. */
  pageMounted: { readonly current: boolean };
};

const CyclePageContext = createContext<CyclePage | undefined>(undefined);

/** The cycles' page the caller is on (CycleActionsHost). */
export function useCyclePage(): CyclePage {
  const page = useContext(CyclePageContext);
  if (page === undefined) throw new Error("A cycle's quick actions require the CycleActionsHost of their page");
  return page;
}

type Props = {
  workspaceSlug: string;
  projectId: string;
  children: ReactNode;
};

/**
 * The host, on a page of a project's cycles (their list, their archive, a cycle's own page), of what a cycle's quick
 * actions open and follow (P5a spec 2.5): the dialogs that change, archive or delete a cycle, and the page's lifetime,
 * by which its restore is followed. They are the page's, not the row's: a deletion, an archive or a restore takes the
 * cycle's row away as nerve confirms it (a change of days too, that makes a cycle current), before the page follows
 * nerve's answer; the host stays, so its dialog closes and says so, and the page goes where the change takes it. Each
 * follows the answer only in the session it was sent in (followInSession), and while the page is the one shown:
 * mounted by the page, the host goes with it (a Back, another project), and nerve's late answer then reaches the store
 * alone.
 */
export function CycleActionsHost(props: Props) {
  const { workspaceSlug, projectId, children } = props;
  // states
  const [shown, setShown] = useState<{ dialog: CycleDialog; cycle: Cycle; open: boolean }>();
  const mounted = useRef(false);
  useLayoutEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);

  // the page as its rows have it, the same for its lifetime
  const page = useMemo<CyclePage>(
    () => ({ openDialog: (dialog, cycle) => setShown({ dialog, cycle, open: true }), pageMounted: mounted }),
    []
  );
  // a dialog closed keeps its cycle while it fades out
  const close = () => setShown((current) => current && { ...current, open: false });
  const isOpen = (dialog: CycleDialog) => shown?.open === true && shown.dialog === dialog;

  return (
    <CyclePageContext.Provider value={page}>
      {children}
      <CycleCreateUpdateModal
        isOpen={isOpen("edit")}
        handleClose={close}
        projectId={projectId}
        cycleId={shown?.cycle.id}
      />
      <ArchiveCycleModal
        isOpen={isOpen("archive")}
        handleClose={close}
        cycle={shown?.cycle}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
      />
      <CycleDeleteModal
        isOpen={isOpen("delete")}
        handleClose={close}
        cycle={shown?.cycle}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
      />
    </CyclePageContext.Provider>
  );
}
