/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useLayoutEffect, useRef, type RefObject } from "react";

/** A control a form shows refused: invalid, its message its description (M3 closeout, C11-1). */
const REFUSED = '[aria-invalid="true"]';

/**
 * Gives the focus to the field a form shows refused once it shows it so (M3 closeout, C11-1): after each attempt to
 * send the form (a submit whose rules failed, or one nerve refused under its fields), the first control in the form, in
 * the document's order, that is aria-invalid takes the focus. At its focus event, which a screen reader reads it on, it
 * is invalid and described by its message.
 *
 * attempt changes with each attempt and with nothing else, so that typing, which revalidates the fields, never moves
 * the focus: react-hook-form's submitCount, or the last refusal of a form that keeps its own. None (0, undefined) is
 * no attempt. A control already focused keeps the focus: focusing it again does nothing.
 *
 * The focus waits for the commit that shows the attempt's messages, and for what that commit renders in turn: propel's
 * FieldError gives its id to its Field in a layout effect, and the control is described only in the render that
 * follows, which React runs before it returns from the commit. The effect is a layout effect, so it runs in the commit,
 * and it focuses in a microtask, which runs once React has returned. react-hook-form's own focus (shouldFocusError,
 * setError's shouldFocus) comes before the commit: the forms turn it off.
 *
 * Gives the ref of the form's element, within which the control is looked for.
 */
export function useRefusedFocus<E extends HTMLElement>(attempt: unknown): RefObject<E | null> {
  const form = useRef<E>(null);
  useLayoutEffect(() => {
    if (!attempt) return;
    queueMicrotask(() => form.current?.querySelector<HTMLElement>(REFUSED)?.focus());
  }, [attempt]);
  return form;
}
