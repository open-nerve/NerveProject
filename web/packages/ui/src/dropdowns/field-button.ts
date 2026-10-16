/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ISelectFieldProps } from "./helper";

/**
 * What a select's button says as one of a form's fields (M3 closeout, C11-1): the message that says why the value is
 * refused describes it, and it says the value is refused (aria-invalid), by which its form finds it to give it the
 * focus. The props the button spreads.
 */
export function fieldButtonProps({ describedBy, invalid = false }: ISelectFieldProps) {
  return { "aria-describedby": describedBy, "aria-invalid": invalid || undefined };
}
