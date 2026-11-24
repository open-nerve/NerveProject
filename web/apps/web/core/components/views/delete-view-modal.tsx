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
// local imports
import { useViewsPage } from "./actions-host";

type Props = {
  isOpen: boolean;
  handleClose: () => void;
  /** The view to delete, as it was when the dialog was asked for. */
  view: View | undefined;
  workspaceSlug: string;
  projectId: string;
};

/**
 * The dialog that deletes a view (P6a spec 2.3, R17). It closes only once nerve has deleted the view, says so and, when
 * the page shows the view (its own page), goes to the project's views; refused, it stays and says nerve's reason. While
 * the deletion is out nothing closes it (Escape, the backdrop, Cancel). Its follow-ups are those of the session it was
 * sent in (followInSession) and of its page (ViewActionsHost): once the page has gone (a Back, another project), nerve's
 * late answer reaches the store alone.
 */
export function DeleteProjectViewModal(props: Props) {
  const { isOpen, handleClose, view, workspaceSlug, projectId } = props;
  // router
  const navigate = useNavigate();
  const { viewId: pageViewId } = useParams();
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { remove } = useViews();
  const toastRefusal = useRefusalToast();
  const { pageMounted } = useViewsPage();
  // states
  const [busy, setBusy] = useState(false);
  const out = useRef(false);

  const submit = async () => {
    if (!view || out.current) return;
    out.current = true;
    setBusy(true);
    const shownByPage = pageViewId === view.id;
    await followInSession(() => remove(view.id), {
      done: () => {
        if (!pageMounted.current) return;
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("toast.success"),
          message: t("project_views.action.delete.success"),
        });
        handleClose();
        if (shownByPage) return navigate(`/${workspaceSlug}/projects/${projectId}/views`);
      },
      failed: (error) => {
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
      title={t("project_views.delete_view.title")}
      primaryButtonText={{ default: t("delete"), loading: t("deleting") }}
      secondaryButtonText={t("cancel")}
      content={t("project_views.delete_view.content")}
    />
  );
}
