/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { ISSUE_LAYOUTS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { getButtonStyling } from "@nerve/propel/button";
import { ChevronDownOutline } from "@makeplane/propel/icons";
import type { EIssueLayoutTypes } from "@nerve/types";
import { CustomMenu } from "@nerve/ui";
import { IssueLayoutIcon } from "../../layout-icon";

export function MobileLayoutSelection({
  layouts,
  onChange,
  activeLayout,
  disabled = false,
}: {
  layouts: EIssueLayoutTypes[];
  onChange: (layout: EIssueLayoutTypes) => void;
  activeLayout?: EIssueLayoutTypes;
  /** Stops the layouts (a view's page's draft being saved, P6a spec 2.7, R29); off unless given. */
  disabled?: boolean;
}) {
  const { t } = useTranslation();
  return (
    <CustomMenu
      ariaLabel={t("common.layout")}
      disabled={disabled}
      maxHeight="md"
      className="flex flex-grow justify-center text-13 text-secondary"
      placement="bottom-start"
      customButton={
        <span className={getButtonStyling("secondary", "lg")}>
          {activeLayout && <IssueLayoutIcon layout={activeLayout} size={14} strokeWidth={2} className="h-3.5 w-3.5" />}
          <ChevronDownOutline className="my-auto size-3 text-secondary" />
        </span>
      }
      // the button grows to its half of the small screen's row, from the page's edge: the keyboard's ring is inside its
      // edge, which the page's box would cut outside
      customButtonClassName="flex flex-grow justify-center text-secondary text-13 focus-visible:outline-1 focus-visible:-outline-offset-1 focus-visible:outline-accent-strong"
      closeOnSelect
    >
      {ISSUE_LAYOUTS.filter((l) => layouts.includes(l.key)).map((layout) => (
        <CustomMenu.MenuItem
          key={layout.key}
          onClick={() => {
            onChange(layout.key);
          }}
          className="flex items-center gap-2"
        >
          <IssueLayoutIcon layout={layout.key} className="h-3 w-3" />
          <div className="text-tertiary">{t(layout.i18n_label)}</div>
        </CustomMenu.MenuItem>
      ))}
    </CustomMenu>
  );
}
