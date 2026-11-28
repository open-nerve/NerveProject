/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRef, useState } from "react";
import { useNavigate, useParams } from "react-router";
// nerve imports
import type { View } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { AlertModalCore } from "@nerve/ui";
// hooks
import { useViews } from "@/hooks/store/use-views";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";
// store
import type { ViewContainer } from "@/store/views/view-container";
// local imports
import { useViewsPage } from "./actions-host";
import { pageViewParam, viewsPath } from "./view-paths";

/**
 * What the dialog says of each container's views (M7/P6b spec 2.11, R29), its keys whole: its title and its content,
 * and the deletion done.
 */
const SAYS = {
  project: {
    title: "project_views.delete_view.title",
    content: "project_views.delete_view.content",
    deleted: "project_views.action.delete.success",
  },
  workspace: {
    title: "workspace_views.delete_view.title",
    content: "workspace_views.delete_view.content",
    deleted: "workspace_views.action.delete.success",
  },
} as const;

type Props = {
  isOpen: boolean;
  handleClose: () => void;
  /** The view to delete, as it was when the dialog was asked for. */
  view: View | undefined;
  workspaceSlug: string;
  /** The view's container, a project or the workspace, whose views the page shows. */
  container: ViewContainer;
};

/**
 * The dialog that deletes a view of the container, a project's or a workspace's own (P6a spec 2.3, R17; M7/P6b spec
 * 2.11, R15), in the container's words. It closes only once nerve has deleted the view, says so and, when the page
 * shows the view (its own page, whose view is the route's param of the container, pageViewParam), goes to the
 * container's views; refused, it stays and says nerve's reason. While the deletion is out nothing closes it (Escape,
 * the backdrop, Cancel). Its follow-ups are those of the session it was sent in (followInSession) and of its page
 * (ViewActionsHost): once the page has gone (a Back, another project or workspace), nerve's late answer reaches the
 * store alone. The view's own page is leaving from the deletion's sending (pageLeaving), until nerve refuses it: it
 * shows nothing of the view gone before it goes (re-review n1).
 */
export function DeleteViewModal(props: Props) {
  const { isOpen, handleClose, view, workspaceSlug, container } = props;
  // router
  const navigate = useNavigate();
  const pageViewId = useParams()[pageViewParam(container)];
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { remove } = useViews();
  const toastRefusal = useRefusalToast();
  const { pageMounted, pageLeaving } = useViewsPage();
  // states
  const [busy, setBusy] = useState(false);
  const out = useRef(false);

  const submit = async () => {
    if (!view || out.current) return;
    out.current = true;
    setBusy(true);
    const shownByPage = pageViewId === view.id;
    if (shownByPage) pageLeaving.current = true;
    await followInSession(() => remove(view.id), {
      done: () => {
        if (!pageMounted.current) return;
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("toast.success"),
          message: t(SAYS[container.kind].deleted),
        });
        handleClose();
        if (shownByPage) return navigate(viewsPath(workspaceSlug, container));
      },
      failed: (error) => {
        pageLeaving.current = false;
        if (pageMounted.current) toastRefusal(error);
      },
    });
    out.current = false;
    if (pageMounted.current) setBusy(false);
  };

  return (
    <AlertModalCore
      handleClose={handleClose}
      handleSubmit={() => void submit()}
      isSubmitting={busy}
      isOpen={isOpen}
      title={t(SAYS[container.kind].title)}
      primaryButtonText={{ default: t("delete"), loading: t("deleting") }}
      secondaryButtonText={t("cancel")}
      content={t(SAYS[container.kind].content)}
    />
  );
}
