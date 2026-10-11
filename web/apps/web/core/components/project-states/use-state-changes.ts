/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { StateCreate, StateGroup, StateUpdate } from "@nerve/api-client";
// hooks
import { useProjectState } from "@/hooks/store/use-project-state";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { useFormRefusal } from "@/lib/form-refusal";
import { followInSession } from "@/lib/in-session";

/** The fields of a state's form under which nerve's refusals show, in the order their messages show. */
type StateFormField = "name" | "description";
const FIELDS: readonly StateFormField[] = ["name", "description"];
/** nerve's code for a name another state of the project has, under the name (M3 design 3.17). */
const TAKEN: Readonly<Record<string, StateFormField>> = { "project.state_name_taken": "name" };

/** Where a state's form says a message of nerve's refusal: under the field (its setError). */
export type StateUnderField = (field: StateFormField, message: string) => void;

/**
 * The changes of a project's states that its settings page makes (M3 design 2 P6, 7.6), each through the state store,
 * one at a time (v0 design 7.7). The page follows each only in the session it was sent in (M3 design 7.1): done, the
 * caller's follow-up of success, once nerve has made the change; nerve's refusal of a creation or a change under the
 * form's fields it names (a name taken, under the name), else in a toast, as every refusal of a move, a deletion or a
 * new default is (a default or a group's only state, which the page offers no deletion of, refused all the same when
 * another tab changed them meanwhile). Each settles once the page has followed the change.
 */
export function useStateChanges() {
  const { createState, updateState, moveState, deleteState, markStateAsDefault } = useProjectState();
  const failed = useRefusalToast();
  const showRefusal = useFormRefusal(FIELDS, TAKEN);

  return {
    /** Creates a state in the project of projectId, last of its group. */
    create: (projectId: string, data: StateCreate, underField: StateUnderField, done: () => void) =>
      followInSession(() => createState(projectId, data), { done, failed: (error) => showRefusal(error, underField) }),
    /** Changes the state of stateId to what data names. */
    update: (stateId: string, data: StateUpdate, underField: StateUnderField, done: () => void) =>
      followInSession(() => updateState(stateId, data), { done, failed: (error) => showRefusal(error, underField) }),
    /** Moves the state of stateId into group, before the state of droppedOnId, or after it (StateStore.moveState). */
    move: (stateId: string, group: StateGroup, droppedOnId: string, after: boolean) =>
      followInSession(() => moveState(stateId, group, droppedOnId, after), { failed }),
    /** Deletes the state of stateId. */
    remove: (stateId: string, done: () => void) => followInSession(() => deleteState(stateId), { done, failed }),
    /** Makes the state of stateId its project's default. */
    markDefault: (stateId: string) => followInSession(() => markStateAsDefault(stateId), { failed }),
  };
}
