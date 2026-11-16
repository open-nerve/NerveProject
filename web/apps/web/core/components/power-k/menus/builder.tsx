/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { Command } from "cmdk";
import { useTranslation } from "@nerve/i18n";
// local imports
import { PowerKModalCommandItem } from "../ui/modal/command-item";
import { PowerKMenuEmptyState } from "./empty-state";

/**
 * A key of the command palette's texts (power-k.json), never a text itself (P7c spec 2.3); that a menu's key is
 * defined is tools/i18n-keys.mjs's check, which reads it at the menu.
 */
type PowerKTextKey = `power_k.${string}`;

type Props<T> = {
  heading?: string;
  items: T[];
  onSelect: (item: T) => void;
  getIcon?: (item: T) => React.ComponentType<{ className?: string }>;
  getIconNode?: (item: T) => React.ReactNode;
  getKey: (item: T) => string;
  getLabel: (item: T) => React.ReactNode;
  /**
   * The item's value to cmdk, which keeps the list's selection by it: an item whose name another may share is valued
   * by its id, and found by its getKeywords.
   */
  getValue: (item: T) => string;
  /** The texts the palette's search finds the item by, in place of its value (powerKFilter); by its value when none. */
  getKeywords?: (item: T) => string[];
  isSelected?: (item: T) => boolean;
  /** The key of what the menu says when it has no items; the empty state's own (no results) when none is given. */
  emptyTextKey?: PowerKTextKey;
};

export function PowerKMenuBuilder<T>({
  heading,
  items,
  onSelect,
  getIcon,
  getIconNode,
  getKey,
  getLabel,
  getValue,
  getKeywords,
  isSelected,
  emptyTextKey,
}: Props<T>) {
  const { t } = useTranslation();

  if (items.length === 0) return <PowerKMenuEmptyState emptyText={emptyTextKey && t(emptyTextKey)} />;

  return (
    <Command.Group heading={heading}>
      {items.map((item) => (
        <PowerKModalCommandItem
          key={getKey(item)}
          icon={getIcon?.(item)}
          iconNode={getIconNode?.(item)}
          value={getValue(item)}
          keywords={getKeywords?.(item)}
          label={getLabel(item)}
          isSelected={isSelected?.(item)}
          onSelect={() => onSelect(item)}
        />
      ))}
    </Command.Group>
  );
}
