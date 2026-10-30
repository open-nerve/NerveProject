/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { Label, LabelCreate, LabelUpdate } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// hooks
import { useLabel } from "@/hooks/store/use-label";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { fieldErrorKeys } from "@/lib/error-messages";
import { useFormRefusal } from "@/lib/form-refusal";
import { followInSession } from "@/lib/in-session";

/** The field of a label's form under which nerve's refusals show. */
type LabelFormField = "name";
const FIELDS: readonly LabelFormField[] = ["name"];
/** nerve's code for a name another label of the project has, in any case, under the name (M3 design 3.16). */
const TAKEN: Readonly<Record<string, LabelFormField>> = { "project.label_name_taken": "name" };

/** Where a label's form says a message of nerve's refusal: under the field (its setError). */
export type LabelUnderField = (field: LabelFormField, message: string) => void;

/**
 * The changes of a project's labels that its settings page makes (M3 design 2 P7, 7.6), each through the label store,
 * one at a time (v0 design 7.7). The page follows each only in the session it was sent in (M3 design 7.1): done, the
 * caller's follow-up of success, once nerve has made the change; nerve's refusal of a creation or a change under the
 * form's name it names (a name taken, in any case), else in a toast, as every refusal of a move, a removal from its
 * group or a deletion is. A move to a place nerve does not allow, which the page's drop rules keep from but another
 * tab's change can make, is said as such: nerve refuses its parent_id for a parent of another project or none, the
 * label itself, a parent under another, or a label with labels under it (3.16: labels have two levels). A removal
 * from its group sends no parent, so nerve never refuses it so. Each settles once the page has followed the change.
 */
export type LabelCreated = (label: Label) => void | Promise<void>;
export type IssueLabelCreation = (data: LabelCreate, underField: LabelUnderField, done: LabelCreated) => Promise<void>;

export function useLabelChanges(captureScope?: () => () => boolean) {
  const { createLabel, updateLabel, updateLabelPosition, deleteLabel } = useLabel();
  const { t } = useTranslation();
  const failed = useRefusalToast();
  const showRefusal = useFormRefusal(FIELDS, TAKEN);
  const placeRefused = (error: unknown) => {
    if (fieldErrorKeys(error).parent_id === undefined) {
      failed(error);
      return;
    }
    setToast({
      type: TOAST_TYPE.ERROR,
      title: t("toast.error"),
      message: t("project_settings.labels.parent_not_allowed"),
    });
  };

  return {
    /** Creates a label at the top of the project of projectId, last of its labels. */
    create: (projectId: string, data: LabelCreate, underField: LabelUnderField, done: LabelCreated) => {
      const current = captureScope?.() ?? (() => true);
      return followInSession(() => createLabel(projectId, data), {
        done: (label) => {
          if (current()) return done(label);
        },
        failed: (error) => {
          if (current()) showRefusal(error, underField);
        },
      });
    },
    /** Changes the label of labelId to what data names. */
    update: (labelId: string, data: LabelUpdate, underField: LabelUnderField, done: () => void) =>
      followInSession(() => updateLabel(labelId, data), { done, failed: (error) => showRefusal(error, underField) }),
    /** Moves the label of labelId where it was dropped (LabelStore.updateLabelPosition). */
    move: (labelId: string, parentId: string | null, droppedOnId: string | undefined, dropAtEndOfList: boolean) =>
      followInSession(() => updateLabelPosition(labelId, parentId, droppedOnId, dropAtEndOfList), {
        failed: placeRefused,
      }),
    /** Moves the label of labelId to the top of its project's labels. */
    removeFromGroup: (labelId: string) => followInSession(() => updateLabel(labelId, { parent_id: null }), { failed }),
    /** Deletes the label of labelId, and the labels under it. */
    remove: (labelId: string, done: () => void) => followInSession(() => deleteLabel(labelId), { done, failed }),
  };
}
