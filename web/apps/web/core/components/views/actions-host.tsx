/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { createContext, useContext, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from "react";
// nerve imports
import type { View } from "@nerve/api-client";
// local imports
import { DeleteProjectViewModal } from "./delete-view-modal";

/** The dialogs of a view that its page opens: the one its quick actions open to delete it. */
type ViewDialog = "delete";

/** What a view's quick actions have of the page they are on (ViewActionsHost). */
type ViewsPage = {
  /** Opens the page's dialog of the view. */
  openDialog: (dialog: ViewDialog, view: View) => void;
  /** Whether the page is still mounted: a change sent from it is followed while it is, though its row has gone. */
  pageMounted: { readonly current: boolean };
};

const ViewsPageContext = createContext<ViewsPage | undefined>(undefined);

/** The page of a project's views the caller is on (ViewActionsHost). */
export function useViewsPage(): ViewsPage {
  const page = useContext(ViewsPageContext);
  if (page === undefined) throw new Error("A view's quick actions require the ViewActionsHost of their page");
  return page;
}

type Props = {
  workspaceSlug: string;
  projectId: string;
  children: ReactNode;
};

/**
 * The host, on a page of a project's views, of what a view's quick actions open and follow (P6a spec 2.3, R17, as
 * ModuleActionsHost for modules): the dialog that deletes a view, and the page's lifetime. They are the page's, not the
 * row's: a deletion takes the view's row away as nerve confirms it, before the page follows nerve's answer; the host
 * stays, so its dialog closes and says so. Each follows the answer only in the session it was sent in
 * (followInSession), and while the page is the one shown: mounted by the page, the host goes with it (a Back, another
 * project), and nerve's late answer then reaches the store alone.
 */
export function ViewActionsHost(props: Props) {
  const { workspaceSlug, projectId, children } = props;
  // states
  const [shown, setShown] = useState<{ dialog: ViewDialog; view: View; open: boolean }>();
  const mounted = useRef(false);
  useLayoutEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);

  // the page as its rows have it, the same for its lifetime
  const page = useMemo<ViewsPage>(
    () => ({ openDialog: (dialog, view) => setShown({ dialog, view, open: true }), pageMounted: mounted }),
    []
  );
  // a dialog closed keeps its view while it fades out
  const close = () => setShown((current) => current && { ...current, open: false });
  const isOpen = (dialog: ViewDialog) => shown?.open === true && shown.dialog === dialog;

  return (
    <ViewsPageContext.Provider value={page}>
      {children}
      <DeleteProjectViewModal
        isOpen={isOpen("delete")}
        handleClose={close}
        view={shown?.view}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
      />
    </ViewsPageContext.Provider>
  );
}
