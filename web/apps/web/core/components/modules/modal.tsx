/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLayoutEffect, useRef, useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import type { ModuleCreate, ModuleUpdate } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
// hooks
import { useModules } from "@/hooks/store/use-modules";
// lib
import { useFormRefusal } from "@/lib/form-refusal";
import { followInSession } from "@/lib/in-session";
// local imports
import { ModuleForm, type ModuleFormField, type ModuleFormValues, type ModuleUnderField } from "./form";

/**
 * The fields of the form under which nerve's refusals show (422 validation_failed; a member it refuses, member_ids[i],
 * under the members'); others show in a toast, a name another module has (planning.module_name_taken) among them.
 */
const FIELDS: readonly ModuleFormField[] = ["name", "start_date", "target_date", "lead_id", "member_ids"];
const TAKEN: Readonly<Record<string, ModuleFormField>> = {};

/**
 * A new module as nerve takes it (ModuleCreate, a closed body): its name and its status, and its description, each
 * day, its lead and its members when it has them; never a null day (ModuleCreate's days are not nullable), nor its
 * project's id, which the address names.
 */
function creation({ name, description, start_date, target_date, status, lead_id, member_ids }: ModuleFormValues) {
  const body: ModuleCreate = {
    name,
    status,
    ...(description === "" ? {} : { description }),
    ...(start_date === null ? {} : { start_date }),
    ...(target_date === null ? {} : { target_date }),
    ...(lead_id === null ? {} : { lead_id }),
    ...(member_ids.length === 0 ? {} : { member_ids }),
  };
  return body;
}

/** Whether two sets of members are the same, whatever the order they were picked in. */
const sameMembers = (a: readonly string[], b: readonly string[]) =>
  a.length === b.length && a.every((id) => b.includes(id));

/**
 * A change as nerve takes it (ModuleUpdate): the fields changed since the form opened, nothing else; the days as a pair,
 * both when either changed (null for one cleared), as nerve checks them together; the members as the whole set the form
 * holds, which nerve takes in place of the module's (R6).
 */
function changes(initial: ModuleFormValues, values: ModuleFormValues): ModuleUpdate {
  const datesChanged = values.start_date !== initial.start_date || values.target_date !== initial.target_date;
  return {
    ...(values.name === initial.name ? {} : { name: values.name }),
    ...(values.description === initial.description ? {} : { description: values.description }),
    ...(datesChanged ? { start_date: values.start_date, target_date: values.target_date } : {}),
    ...(values.status === initial.status ? {} : { status: values.status }),
    ...(values.lead_id === initial.lead_id ? {} : { lead_id: values.lead_id }),
    ...(sameMembers(values.member_ids, initial.member_ids) ? {} : { member_ids: values.member_ids }),
  };
}

type Props = {
  isOpen: boolean;
  handleClose: () => void;
  /** The workspace of the projects a new module may go to, the route's. */
  workspaceSlug: string;
  /** The project a new module goes to, until the caller picks another. */
  projectId: string;
  /** The module to change; none, to make a new one. */
  moduleId?: string;
};

/**
 * The dialog that makes a module, or changes the module of moduleId (P6a spec 2.7). It closes only once nerve has made
 * the module or the change, and says so; refused, it stays with what it holds, nerve's reason under the field it names
 * (422) or in a toast (a name another module has, modules turned off, an archived module or project, a role that may
 * not, nerve busy; errorMessageKey). While what it sent is out, nothing closes it (Escape, the backdrop, Cancel). Its
 * follow-ups are those of the session and the page it was sent from (followInSession): once the dialog has gone with
 * its page (its host's, ModuleActionsHost or ProjectLevelModals), nerve's late answer reaches the store alone. Closed,
 * the dialog gives the focus back to what opened it.
 */
export const CreateUpdateModuleModal = observer(function CreateUpdateModuleModal(props: Props) {
  const { isOpen, handleClose, workspaceSlug, projectId, moduleId } = props;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { getModuleById, create, update } = useModules();
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
  const editing = moduleId !== undefined;
  const module = editing ? getModuleById(moduleId) : undefined;

  const submit = async (values: ModuleFormValues, initial: ModuleFormValues, underField: ModuleUnderField) => {
    if (out.current) return;
    out.current = true;
    setBusy(true);
    await followInSession(
      () =>
        moduleId === undefined
          ? create(values.project_id, creation(values))
          : update(moduleId, changes(initial, values)),
      {
        done: () => {
          if (!mounted.current) return;
          setToast({
            type: TOAST_TYPE.SUCCESS,
            title: editing
              ? t("project_module.action.update.success.title")
              : t("project_module.action.create.success.title"),
            message: editing
              ? t("project_module.action.update.success.description")
              : t("project_module.action.create.success.description"),
          });
          handleClose();
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
      <ModuleForm
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        module={module}
        editing={editing}
        busy={busy}
        onSubmit={submit}
        onCancel={handleClose}
      />
    </ModalCore>
  );
});
