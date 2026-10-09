/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect } from "react";
import type { Control, FieldArrayWithId, FormState } from "react-hook-form";
import { useFieldArray, useForm } from "react-hook-form";
// nerve imports
import type { InvitationCreate, WorkspaceInvitation, WorkspaceInvitationsCreate } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// components
import { invitationRefusal } from "@/components/workspace/invite-modal/refusal";
// lib
import { followInSession } from "@/lib/in-session";

/** A row as the form adds it: an address to type, invited as a member. */
const newRow = (): InvitationCreate => ({ email: "", role: 15 });

/** The form as it opens: one row. */
const opened = (): WorkspaceInvitationsCreate => ({ invitations: [newRow()] });

type TUseWorkspaceInvitationProps = {
  /** Sends the invitations of the form, nerve's WorkspaceInvitationsCreate, and gives the invitations nerve made. */
  invite: (data: WorkspaceInvitationsCreate) => Promise<WorkspaceInvitation[]>;
  /** What the form's place does with the invitations sent: the modal closes, the onboarding shows their links. */
  onSent: (invitations: WorkspaceInvitation[]) => void;
};

type TUseWorkspaceInvitationReturn = {
  control: Control<WorkspaceInvitationsCreate>;
  fields: FieldArrayWithId<WorkspaceInvitationsCreate, "invitations", "id">[];
  formState: FormState<WorkspaceInvitationsCreate>;
  remove: (index: number) => void;
  onFormSubmit: () => void;
  /** Empties the form: one row again. */
  clear: () => void;
  appendField: () => void;
};

/**
 * The invitation form (M3 design 2, W4): its rows, each an address and a role, which it sends together. Sent, the form
 * empties, says so and gives nerve's invitations to onSent; refused, it shows why under each row nerve names, else in a
 * toast (invitationRefusal). It follows the answer only in the session the invitations were sent in (M3 design 7.1).
 */
export const useWorkspaceInvitationActions = (props: TUseWorkspaceInvitationProps): TUseWorkspaceInvitationReturn => {
  const { invite, onSent } = props;
  const { t } = useTranslation();
  // form info
  const { control, reset, handleSubmit, formState, setError } = useForm<WorkspaceInvitationsCreate>({
    defaultValues: opened(),
  });

  const { fields, append, remove } = useFieldArray({
    control,
    name: "invitations",
  });

  const appendField = () => {
    append(newRow());
  };

  const onSubmitForm = (data: WorkspaceInvitationsCreate) =>
    followInSession(() => invite(data), {
      done: (invitations) => {
        reset(opened());
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("toast.success"),
          message: t("workspace_settings.settings.members.invitations_sent_successfully"),
        });
        onSent(invitations);
      },
      failed: (error) => {
        const refusal = invitationRefusal(error, data.invitations.length);
        if (refusal.kind === "toast") {
          setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t(refusal.message) });
          return;
        }
        for (const { index, message } of refusal.rows) {
          setError(`invitations.${index}.email`, { type: "server", message: t(message) });
        }
      },
    });

  useEffect(() => {
    if (fields.length === 0) append([newRow()]);
  }, [fields, append]);

  return {
    control,
    fields,
    formState,
    remove,
    onFormSubmit: handleSubmit(onSubmitForm),
    clear: () => reset(opened()),
    appendField,
  };
};
