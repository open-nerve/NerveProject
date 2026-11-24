/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { createContext, useContext, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from "react";
// nerve imports
import type { View } from "@nerve/api-client";
// local imports
import { DeleteProjectViewModal } from "./delete-view-modal";
import { CreateUpdateProjectViewModal } from "./modal";

/** The dialogs of a view that its page opens: those of its quick actions, which change it or delete it. */
type ViewDialog = "edit" | "delete";

/** What a page of a project's views has of its host (ViewActionsHost): its header, its list and its views' rows. */
type ViewsPage = {
  /** Opens the page's dialog of a new view ("Add view", the empty list's New). */
  openCreate: () => void;
  /** Opens the page's dialog of the view. */
  openDialog: (dialog: ViewDialog, view: View) => void;
  /** Whether the page is still mounted: a change sent from it is followed while it is, though its row has gone. */
  pageMounted: { readonly current: boolean };
};

const ViewsPageContext = createContext<ViewsPage | undefined>(undefined);

/** The page of a project's views the caller is on (ViewActionsHost). */
export function useViewsPage(): ViewsPage {
  const page = useContext(ViewsPageContext);
  if (page === undefined) throw new Error("A page of a project's views requires its ViewActionsHost");
  return page;
}

type Props = {
  workspaceSlug: string;
  projectId: string;
  children: ReactNode;
};

/**
 * The host, on a page of a project's views, of the dialogs it opens and follows (P6a spec 2.3, 2.4, R17, as
 * ModuleActionsHost for modules): the dialog of a new view, which its header's "Add view" and its empty list's New
 * open; those of a view's quick actions, which change it or delete it; and the page's lifetime. They are the page's,
 * not the row's: a deletion takes the view's row away as nerve confirms it (a change too, a rename the page's search no
 * longer finds), before the page follows nerve's answer; the host stays, so its dialog closes and says so. Each follows
 * the answer only in the session it was sent in (followInSession), and while the page is the one shown: mounted by the
 * page, the host goes with it (a Back, another project), and nerve's late answer then reaches the store alone.
 */
export function ViewActionsHost(props: Props) {
  const { workspaceSlug, projectId, children } = props;
  // states
  const [shown, setShown] = useState<{ dialog: "create" | ViewDialog; view?: View; open: boolean }>();
  const mounted = useRef(false);
  useLayoutEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);

  // the page as its header, its list and its rows have it, the same for its lifetime
  const page = useMemo<ViewsPage>(
    () => ({
      openCreate: () => setShown({ dialog: "create", open: true }),
      openDialog: (dialog, view) => setShown({ dialog, view, open: true }),
      pageMounted: mounted,
    }),
    []
  );
  // a dialog closed keeps its view while it fades out
  const close = () => setShown((current) => current && { ...current, open: false });
  const isOpen = (dialog: "create" | ViewDialog) => shown?.open === true && shown.dialog === dialog;

  return (
    <ViewsPageContext.Provider value={page}>
      {children}
      <CreateUpdateProjectViewModal
        isOpen={isOpen("create") || isOpen("edit")}
        handleClose={close}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        viewId={shown?.dialog === "edit" ? shown.view?.id : undefined}
      />
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
