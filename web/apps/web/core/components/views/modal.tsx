/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLayoutEffect, useRef, useState } from "react";
import { isEqual } from "lodash-es";
import { observer } from "mobx-react";
import { useNavigate } from "react-router";
// nerve imports
import type { ViewCreate, ViewUpdate } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
// hooks
import { useViews } from "@/hooks/store/use-views";
// lib
import { useFormRefusal } from "@/lib/form-refusal";
import { followInSession } from "@/lib/in-session";
// local imports
import { ProjectViewForm, type ViewFormField, type ViewFormValues, type ViewUnderField } from "./form";
import { NEW_VIEW_CONTENT, type ViewContent } from "./view-content";

/** The fields of the form under which nerve's refusals show (422 validation_failed); others show in a toast. */
const FIELDS: readonly ViewFormField[] = ["name"];
/** No code of nerve's names a field's value as another view's: views' names need not differ. */
const TAKEN: Readonly<Record<string, ViewFormField>> = {};

/** A new view as nerve takes it (ViewCreate): who it is, as the form holds it, and what it keeps besides. */
function creation({ name, description, logo_props }: ViewFormValues, content: ViewContent): ViewCreate {
  return { name, description, logo_props, ...content };
}

/**
 * A change as nerve takes it (ViewUpdate, each field given replacing the view's): the fields changed since the form
 * opened, nothing else, so never the view's filters nor its display, which its page changes (R28).
 */
function changes(initial: ViewFormValues, values: ViewFormValues): ViewUpdate {
  return {
    ...(values.name === initial.name ? {} : { name: values.name }),
    ...(values.description === initial.description ? {} : { description: values.description }),
    ...(isEqual(values.logo_props, initial.logo_props) ? {} : { logo_props: values.logo_props }),
  };
}

type Props = {
  isOpen: boolean;
  handleClose: () => void;
  /** The workspace of the project, the route's. */
  workspaceSlug: string;
  /** The project a new view goes to, the route's. */
  projectId: string;
  /** The view to change; none, to make a new one. */
  viewId?: string;
  /** What a new view keeps besides who it is (Save as: its page's); none, what "Add view" gives (NEW_VIEW_CONTENT). */
  content?: ViewContent;
};

/**
 * The dialog that makes a view of the project ("Add view", the empty list's New, create_view, Save as) or changes the
 * view of viewId (P6a spec 2.4, R11, R28, R30), as M6's dialogs of a cycle and a module: it holds who the view is (its
 * icon, name and description) and nothing else; a new view keeps the content it was given, or "Add view"'s. It closes
 * only once nerve has made the view or the change, and says so, a new view's page then shown; refused, it stays with
 * what it holds, nerve's reason under the name (422) or in a toast. While what it sent is out, nothing closes it
 * (Escape, the backdrop, Cancel); else each does at once, through its handleClose (ModalCore), and an Escape the icon
 * picker or an input method takes is theirs. Its follow-ups are those of the session and the page it was sent from
 * (followInSession): once the dialog has gone with its page (its host's, ViewActionsHost, ProjectLevelModals or the
 * filters' HOC), nerve's late answer reaches the store alone. Closed, the dialog gives the focus back to what opened it.
 */
export const CreateUpdateProjectViewModal = observer(function CreateUpdateProjectViewModal(props: Props) {
  const { isOpen, handleClose, workspaceSlug, projectId, viewId, content = NEW_VIEW_CONTENT } = props;
  // router
  const navigate = useNavigate();
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { getViewById, create, update } = useViews();
  const showRefusal = useFormRefusal(FIELDS, TAKEN);
  // states
  const [busy, setBusy] = useState(false);
  const out = useRef(false);
  const mounted = useRef(false);
  useLayoutEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  // derived values
  const editing = viewId !== undefined;
  const view = editing ? getViewById(viewId) : undefined;

  const submit = async (values: ViewFormValues, initial: ViewFormValues, underField: ViewUnderField) => {
    if (out.current) return;
    out.current = true;
    setBusy(true);
    await followInSession(
      () =>
        viewId === undefined
          ? create(projectId, creation(values, content))
          : update(viewId, () => changes(initial, values)),
      {
        done: (answer) => {
          if (!mounted.current) return;
          setToast({
            type: TOAST_TYPE.SUCCESS,
            title: t("toast.success"),
            message: editing ? t("project_views.action.update.success") : t("project_views.action.create.success"),
          });
          handleClose();
          if (!editing) return navigate(`/${workspaceSlug}/projects/${projectId}/views/${answer.id}`);
        },
        failed: (error) => {
          if (mounted.current) showRefusal(error, underField);
        },
      }
    );
    out.current = false;
    if (mounted.current) setBusy(false);
  };

  return (
    <ModalCore
      isOpen={isOpen}
      handleClose={busy ? undefined : handleClose}
      position={EModalPosition.TOP}
      width={EModalWidth.XXL}
    >
      <ProjectViewForm view={view} editing={editing} busy={busy} onSubmit={submit} onCancel={handleClose} />
    </ModalCore>
  );
});
