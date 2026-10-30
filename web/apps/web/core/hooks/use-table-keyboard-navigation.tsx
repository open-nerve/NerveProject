/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { isComposing } from "@nerve/utils";

function handleKeyboardNavigation(event: React.KeyboardEvent<HTMLTableElement>) {
  const cell = event.target;
  if (isComposing(event.nativeEvent) || !(cell instanceof HTMLTableCellElement)) return;
  const table = event.currentTarget;
  const row = cell.closest("tr");
  if (!row) return;
  const rows = Array.from(table.rows);
  let next: Element | null | undefined;
  switch (event.key) {
    case "ArrowRight":
      next = cell.nextElementSibling;
      break;
    case "ArrowLeft":
      next = cell.previousElementSibling;
      break;
    case "ArrowUp":
      next = rows[rows.indexOf(row) - 1]?.cells[cell.cellIndex];
      break;
    case "ArrowDown":
      next = rows[rows.indexOf(row) + 1]?.cells[cell.cellIndex];
      break;
    case "Enter":
    case " ": {
      const control = cell.querySelector<HTMLElement>("button:not(:disabled), a[href], .clickable:not(:disabled)");
      if (control) {
        event.preventDefault();
        control.focus();
        control.click();
      }
      return;
    }
    default:
      return;
  }
  if (!(next instanceof HTMLElement)) return;
  event.preventDefault();
  next.focus();
  next.scrollIntoView({ behavior: "smooth", block: "center", inline: "end" });
}

export const useTableKeyboardNavigation = () => handleKeyboardNavigation;
