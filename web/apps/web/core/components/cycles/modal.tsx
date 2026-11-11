/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLayoutEffect, useRef, useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import type { CycleCreate, CycleUpdate } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
// hooks
import { useCycles } from "@/hooks/store/use-cycles";
// lib
import { useFormRefusal } from "@/lib/form-refusal";
import { followInSession } from "@/lib/in-session";
// local imports
import { CycleForm, type CycleFormField, type CycleFormValues, type CycleUnderField } from "./form";

/** The fields of the form under which nerve's refusals show (422 validation_failed); others show in a toast. */
const FIELDS: readonly CycleFormField[] = ["name", "start_date", "end_date"];
/** No code of nerve's names a field's value as another cycle's: an overlap of days is the whole cycle's, in a toast. */
const TAKEN: Readonly<Record<string, CycleFormField>> = {};

/**
 * A new cycle as nerve takes it (CycleCreate, a closed body): its name, its description when it has one, and each day
 * picked. A draft has neither day, never a null one; a day given alone is nerve's to refuse, under the other's field.
 */
function creation({ name, description, start_date, end_date }: CycleFormValues): CycleCreate {
  return {
    name,
    ...(description === "" ? {} : { description }),
    ...(start_date === null ? {} : { start_date }),
    ...(end_date === null ? {} : { end_date }),
  };
}

/**
 * A change as nerve takes it (CycleUpdate): the fields changed since the form opened, nothing else; the days as a pair,
 * both when either changed (both null, to make the cycle a draft again), as nerve checks them together.
 */
function changes(initial: CycleFormValues, values: CycleFormValues): CycleUpdate {
  const datesChanged = values.start_date !== initial.start_date || values.end_date !== initial.end_date;
  return {
    ...(values.name === initial.name ? {} : { name: values.name }),
    ...(values.description === initial.description ? {} : { description: values.description }),
    ...(datesChanged ? { start_date: values.start_date, end_date: values.end_date } : {}),
  };
}

type CycleModalProps = {
  isOpen: boolean;
  handleClose: () => void;
  /** The workspace of the projects a new cycle may go to, the route's. */
  workspaceSlug: string;
  /** The project a new cycle goes to, until the caller picks another. */
  projectId: string;
  /** The cycle to change; none, to make a new one. */
  cycleId?: string;
};

/**
 * The dialog that makes a cycle, or changes the cycle of cycleId (P5a spec 2.6). It closes only once nerve has made
 * the cycle or the change, and says so; refused, it stays with what it holds, nerve's reason under the field it names
 * (422) or in a toast (an overlap of days, cycles turned off, an archived project, a role that may not, nerve busy).
 * While what it sent is out, nothing closes it (Escape, the backdrop, Cancel). Its follow-ups are those of the session
 * and the page it was sent from (followInSession): once the dialog has gone with its project's page (its host's key,
 * a Back), nerve's late answer reaches the store alone. Closed, the dialog gives the focus back to what opened it.
 */
export const CycleCreateUpdateModal = observer(function CycleCreateUpdateModal(props: CycleModalProps) {
  const { isOpen, handleClose, workspaceSlug, projectId, cycleId } = props;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { getCycleById, create, update } = useCycles();
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
  const editing = cycleId !== undefined;
  const cycle = editing ? getCycleById(cycleId) : undefined;

  const submit = async (values: CycleFormValues, initial: CycleFormValues, underField: CycleUnderField) => {
    if (out.current) return;
    out.current = true;
    setBusy(true);
    await followInSession(
      () =>
        cycleId === undefined ? create(values.project_id, creation(values)) : update(cycleId, changes(initial, values)),
      {
        done: () => {
          if (!mounted.current) return;
          setToast({
            type: TOAST_TYPE.SUCCESS,
            title: editing
              ? t("project_cycles.action.update.success.title")
              : t("project_cycles.action.create.success.title"),
            message: editing
              ? t("project_cycles.action.update.success.description")
              : t("project_cycles.action.create.success.description"),
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
      <CycleForm
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        cycle={cycle}
        editing={editing}
        busy={busy}
        onSubmit={submit}
        onCancel={handleClose}
      />
    </ModalCore>
  );
});
