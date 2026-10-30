/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import type { TContextMenuItem } from "./root";
import { cn } from "../../utils";

export function ContextMenuItem({ item, handleClose }: { item: TContextMenuItem; handleClose: () => void }) {
  return (
    <button
      type="button"
      role="menuitem"
      className={cn(
        "flex w-full items-center gap-2 rounded-sm px-1 py-1.5 text-left text-11 text-secondary select-none hover:bg-layer-transparent-hover focus:bg-layer-transparent-hover",
        { "text-placeholder": item.disabled },
        item.className
      )}
      disabled={item.disabled}
      onClick={(event) => {
        event.preventDefault();
        event.stopPropagation();
        if (item.closeOnClick !== false) handleClose();
        item.action();
      }}
    >
      {item.customContent ?? (
        <>
          {item.icon && <item.icon className={cn("h-3 w-3", item.iconClassName)} />}
          <div className="flex-1">
            <h5>{item.title}</h5>
            {item.description && (
              <p className={cn("whitespace-pre-line text-tertiary", { "text-placeholder": item.disabled })}>
                {item.description}
              </p>
            )}
          </div>
        </>
      )}
    </button>
  );
}
