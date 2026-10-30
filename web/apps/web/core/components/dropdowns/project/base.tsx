/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Logo } from "@nerve/propel/emoji-icon-picker";
import { ProjectsOutline } from "@makeplane/propel/icons";
import { CustomSearchSelect } from "@nerve/ui";
import { cn, sortBySelectedFirst } from "@nerve/utils";
// nerve imports
import type { Project } from "@nerve/api-client";
// local imports
import { DropdownButton } from "../buttons";
import { BUTTON_VARIANTS_WITH_TEXT } from "../constants";
import type { TDropdownProps } from "../types";

type Props = TDropdownProps & {
  getProjectById: (projectId: string | null | undefined) => Project | undefined;
  projectIds: string[];
  renderCondition?: (projectId: string) => boolean;
} & (
    | {
        multiple: false;
        onChange: (val: string) => void;
        value: string | null;
      }
    | {
        multiple: true;
        onChange: (val: string[]) => void;
        value: string[];
      }
  );

/** A project's logo, as the dropdown's button shows it. */
const renderProjectLogo = (logoProps: Project["logo_props"]) => (
  <span className="grid h-4 w-4 flex-shrink-0 place-items-center">
    <Logo logo={logoProps} size={14} />
  </span>
);

export const ProjectDropdownBase = observer(function ProjectDropdownBase(props: Props) {
  const { t } = useTranslation();
  const {
    buttonClassName,
    buttonContainerClassName,
    buttonVariant,
    className = "",
    disabled = false,
    getProjectById,
    hideIcon = false,
    multiple,
    onChange,
    placeholder = t("common.project"),
    placement,
    projectIds,
    renderCondition,
    showTooltip = false,
    tabIndex,
    value,
  } = props;
  // store hooks
  const options = projectIds?.map((projectId) => {
    const projectDetails = getProjectById(projectId);
    if (!projectDetails || (renderCondition && !renderCondition(projectId))) return;
    return {
      value: projectId,
      query: `${projectDetails?.name}`,
      content: (
        <div className="flex items-center gap-2">
          {projectDetails?.logo_props && (
            <span className="grid h-4 w-4 flex-shrink-0 place-items-center">
              <Logo logo={projectDetails?.logo_props} size={12} />
            </span>
          )}
          <span className="flex-grow truncate">{projectDetails?.name}</span>
        </div>
      ),
    };
  });

  const sortedOptions = sortBySelectedFirst(
    options.filter((option) => option !== undefined),
    value
  );
  // what the button says: the project picked, how many are, or the placeholder
  const getDisplayName = () => {
    if (Array.isArray(value)) {
      const firstProject = getProjectById(value[0]);
      return value.length ? (value.length === 1 ? firstProject?.name : `${value.length} projects`) : placeholder;
    } else {
      return value ? (getProjectById(value)?.name ?? placeholder) : placeholder;
    }
  };

  // the logos of the projects picked; with several to pick and none picked yet, the icon of projects
  const getProjectIcon = () => {
    if (Array.isArray(value)) {
      return (
        <div className="flex items-center gap-0.5">
          {value.length > 0 ? (
            value.map((projectId) => {
              const projectDetails = getProjectById(projectId);
              return projectDetails?.logo_props ? (
                <span key={projectId}>{renderProjectLogo(projectDetails.logo_props)}</span>
              ) : null;
            })
          ) : (
            <ProjectsOutline className="size-3 text-tertiary" />
          )}
        </div>
      );
    } else {
      const projectDetails = getProjectById(value);
      return projectDetails?.logo_props ? renderProjectLogo(projectDetails.logo_props) : null;
    }
  };

  const content = (
    <DropdownButton
      presentation
      className={cn("group-data-open:bg-layer-transparent-active", buttonClassName)}
      isActive={false}
      tooltipHeading={placeholder}
      tooltipContent={getDisplayName() ?? placeholder}
      showTooltip={showTooltip}
      variant={buttonVariant}
    >
      <span className="sr-only">{placeholder}: </span>
      {!hideIcon && getProjectIcon()}
      <span className={BUTTON_VARIANTS_WITH_TEXT.includes(buttonVariant) ? "max-w-40 truncate" : "sr-only"}>
        {getDisplayName()}
      </span>
    </DropdownButton>
  );
  const shared = {
    options: sortedOptions,
    disabled,
    tabIndex,
    customButton: content,
    className: cn("h-full shrink", className),
    optionsClassName: "w-48",
    customButtonClassName: cn("group h-full max-w-full hover:bg-transparent", buttonContainerClassName),
    placement,
    searchPlaceholder: t("common.search.label"),
    noResultsMessage: t("no_matching_results"),
    loadingMessage: t("common.loading"),
  };
  return multiple ? (
    <CustomSearchSelect {...shared} multiple value={value} onChange={(ids: string[]) => onChange(ids)} />
  ) : (
    <CustomSearchSelect {...shared} value={value} onChange={(id: string) => onChange(id)} />
  );
});
