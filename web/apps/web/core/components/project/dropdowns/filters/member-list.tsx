/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import { ROLE_DETAILS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { getButtonStyling } from "@nerve/propel/button";
import { ChevronDownOutline } from "@makeplane/propel/icons";
import { EUserWorkspaceRoles } from "@nerve/types";
import { cn } from "@nerve/utils";
// components
import { FilterHeader, FilterOption, FiltersDropdown } from "@/components/issues/issue-layouts/filters";

/** A role the members are filtered by: its value in the filter, and the key of its name. */
interface IRoleOption {
  value: string;
  i18n_label: string;
}

type Props = {
  appliedFilters: string[] | null;
  handleUpdate: (role: string) => void;
  memberType: "project" | "workspace";
};

// a project's roles have the workspace's values (EUserProjectRoles), and the same names
const PROJECT_ROLE_OPTIONS: IRoleOption[] = [
  EUserWorkspaceRoles.ADMIN,
  EUserWorkspaceRoles.MEMBER,
  EUserWorkspaceRoles.GUEST,
].map((role) => ({ value: String(role), i18n_label: ROLE_DETAILS[role].i18n_title }));

const WORKSPACE_ROLE_OPTIONS: IRoleOption[] = [
  ...PROJECT_ROLE_OPTIONS,
  { value: "suspended", i18n_label: "workspace_settings.settings.members.suspended" },
];

// Role filter group component
const RoleFilterGroup = observer(function RoleFilterGroup({
  appliedFilters,
  handleUpdate,
  memberType,
}: {
  appliedFilters: string[] | null;
  handleUpdate: (role: string) => void;
  memberType: "project" | "workspace";
}) {
  const [isExpanded, setIsExpanded] = useState(true);
  const { t } = useTranslation();
  const appliedFiltersCount = appliedFilters?.length ?? 0;
  const roleOptions = memberType === "project" ? PROJECT_ROLE_OPTIONS : WORKSPACE_ROLE_OPTIONS;

  return (
    <div className="space-y-2">
      <FilterHeader
        title={`${t("common.roles")}${appliedFiltersCount > 0 ? ` (${appliedFiltersCount})` : ""}`}
        isPreviewEnabled={isExpanded}
        handleIsPreviewEnabled={() => setIsExpanded(!isExpanded)}
      />

      {isExpanded && (
        <div className="space-y-1">
          {roleOptions.map((role) => {
            const isSelected = appliedFilters?.includes(role.value) ?? false;
            return (
              <FilterOption
                key={`role-${role.value}`}
                isChecked={isSelected}
                title={t(role.i18n_label)}
                onClick={() => handleUpdate(role.value)}
              />
            );
          })}
        </div>
      )}
    </div>
  );
});

/**
 * The members' filter, a popover as the projects' filters are. Opened, it gives the focus to its roles' heading; Tab
 * reaches the roles, Enter or Space toggles one, the popover staying open. Escape closes it, the focus going back to
 * its button; Tab past its last role closes it too, the focus moving on. The popover's button is its one control, what
 * it shows only looking like a button.
 */
export const MemberListFiltersDropdown = observer(function MemberListFiltersDropdown(props: Props) {
  const { appliedFilters, handleUpdate, memberType } = props;
  const { t } = useTranslation();

  const appliedFiltersCount = appliedFilters?.length ?? 0;

  return (
    <FiltersDropdown
      placement="bottom-start"
      focusPanel
      menuButton={
        <span className={cn(getButtonStyling("secondary", "lg"), "relative gap-2")}>
          <span>{t("common.filters")}</span>
          <ChevronDownOutline className="h-3 w-3" />
          {appliedFiltersCount > 0 && (
            <span className="absolute -top-1 -right-1 h-2 w-2 rounded-full bg-accent-primary" />
          )}
        </span>
      }
    >
      <div className="p-2.5">
        <RoleFilterGroup appliedFilters={appliedFilters} handleUpdate={handleUpdate} memberType={memberType} />
      </div>
    </FiltersDropdown>
  );
});
