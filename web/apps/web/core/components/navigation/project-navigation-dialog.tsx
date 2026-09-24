/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
import { CloseOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Checkbox } from "@makeplane/propel/components/checkbox";
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
import { cn } from "@nerve/utils";
// hooks
import { useProjectNavigationPreferences } from "@/hooks/use-navigation-preferences";

type TProjectNavigationDialogProps = {
  isOpen: boolean;
  onClose: () => void;
};

export const ProjectNavigationDialog = observer(function ProjectNavigationDialog(props: TProjectNavigationDialogProps) {
  const { isOpen, onClose } = props;
  const { t } = useTranslation();

  // store hooks
  const {
    preferences: projectPreferences,
    updateNavigationMode,
    updateShowLimitedProjects,
    updateLimitedProjectsCount,
  } = useProjectNavigationPreferences();

  // local state for limited projects count input
  const [projectCountInput, setProjectCountInput] = useState(projectPreferences.limitedProjectsCount.toString());

  // Prevent typing invalid characters in number input
  // oxlint-disable-next-line unicorn/consistent-function-scoping
  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    // Block: e, E, +, -, .
    if (["e", "E", "+", "-", "."].includes(e.key)) {
      e.preventDefault();
    }
  };

  // Handle project count input change
  const handleProjectCountChange = (value: string) => {
    // Strip any non-digit characters
    const cleanedValue = value.replace(/\D/g, "");
    setProjectCountInput(cleanedValue);

    // Parse and validate the value
    const numValue = parseInt(cleanedValue, 10);

    // If valid number, enforce minimum of 1
    if (!isNaN(numValue)) {
      const validValue = Math.max(1, numValue);
      updateLimitedProjectsCount(validValue);
    }
  };

  return (
    <ModalCore isOpen={isOpen} handleClose={onClose} position={EModalPosition.CENTER} width={EModalWidth.XXL}>
      <div className="flex max-h-[90vh] flex-col rounded-lg bg-surface-1">
        {/* Header */}
        <div className="flex justify-between px-6 pt-4">
          <div>
            <h2 className="text-18 font-semibold text-primary">{t("projects")}</h2>
          </div>
          <button
            onClick={onClose}
            className="flex size-5 flex-shrink-0 items-center justify-center rounded-sm text-placeholder hover:bg-layer-1"
            aria-label={t("close")}
          >
            <CloseOutline className="size-4" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 space-y-4 overflow-y-auto px-6 py-4">
          <div className="rounded-md border border-subtle bg-surface-2 px-2 py-2">
            <div className="space-y-3">
              {/* Navigation Mode Radio Buttons */}
              <div className="space-y-2">
                {/* oxlint-disable-next-line jsx_a11y/label-has-associated-control */}
                <label className="flex cursor-pointer gap-2 rounded-md px-2 py-1.5 hover:bg-surface-2">
                  <input
                    type="radio"
                    name="navigation-mode"
                    value="ACCORDION"
                    checked={projectPreferences.navigationMode === "ACCORDION"}
                    onChange={() => updateNavigationMode("ACCORDION")}
                    className="mt-1 size-4 text-accent-primary focus:ring-accent-strong"
                  />
                  <div className="flex-1">
                    <div className="text-13 text-primary">{t("accordion_navigation_control")}</div>
                    <div className="text-11 text-secondary">
                      Feature tabs will appear as nested items under project and acts as accordion.
                    </div>
                  </div>
                </label>

                {/* oxlint-disable-next-line jsx_a11y/label-has-associated-control */}
                <label className="flex cursor-pointer gap-2 rounded-md px-2 py-1.5 hover:bg-surface-2">
                  <input
                    type="radio"
                    name="navigation-mode"
                    value="TABBED"
                    checked={projectPreferences.navigationMode === "TABBED"}
                    onChange={() => updateNavigationMode("TABBED")}
                    className="mt-1 size-4 text-accent-primary focus:ring-accent-strong"
                  />
                  <div className="flex-1">
                    <div className="text-13 text-primary">{t("horizontal_navigation_bar")}</div>
                    <div className="text-11 text-secondary">
                      Feature tabs will appear as horizontal tabs inside a project.
                    </div>
                  </div>
                </label>
              </div>

              {/* Limited Projects Checkbox */}
              <div className="space-y-1">
                <div className="rounded-md px-2 py-1.5 hover:bg-surface-2">
                  <Checkbox
                    label={t("show_limited_projects_on_sidebar")}
                    stretch="full"
                    checked={projectPreferences.showLimitedProjects}
                    onCheckedChange={updateShowLimitedProjects}
                  />
                </div>

                {projectPreferences.showLimitedProjects && (
                  <div className="pl-8">
                    <div className="flex w-full flex-col gap-1">
                      <div className="flex w-full flex-col gap-2 pb-1.5">
                        <label className="w-full text-11 text-secondary">{t("enter_number_of_projects")}</label>
                        <input
                          type="number"
                          min="1"
                          step="1"
                          value={projectCountInput}
                          onKeyDown={handleKeyDown}
                          onChange={(e) => handleProjectCountChange(e.target.value)}
                          className={cn(
                            "w-full rounded-md px-2 py-1 text-13",
                            "border bg-surface-2",
                            "text-secondary",
                            parseInt(projectCountInput) >= 1
                              ? "border-strong focus:border-accent-strong focus:ring-1 focus:ring-accent-strong"
                              : "border-danger-strong focus:border-danger-strong focus:ring-1 focus:ring-danger-strong"
                          )}
                        />
                      </div>
                      {parseInt(projectCountInput) < 1 && projectCountInput !== "" && (
                        <span className="pl-0.5 text-11 text-danger-primary">Minimum value is 1</span>
                      )}
                    </div>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      </div>
    </ModalCore>
  );
});
