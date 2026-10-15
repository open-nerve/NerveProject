/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useCallback } from "react";
import type { ISelectFieldProps } from "./helper";

/**
 * A select's button as one of a form's fields (M3 closeout, C11-1): it is Popper's reference (setReference) and the
 * form's (buttonRef: react-hook-form's field.ref, by which a rule failing on submit focuses it); the message that
 * says why the value is refused describes it, and it says the value is refused (aria-invalid). setButton is the
 * button's ref, described the props it spreads.
 */
export function useFieldButton(
  setReference: (button: HTMLButtonElement | null) => void,
  { buttonRef, describedBy, invalid = false }: ISelectFieldProps
) {
  const setButton = useCallback(
    (button: HTMLButtonElement | null) => {
      setReference(button);
      buttonRef?.(button);
    },
    [setReference, buttonRef]
  );
  const described = { "aria-describedby": describedBy, "aria-invalid": invalid || undefined };
  return { setButton, described };
}
