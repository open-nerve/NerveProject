/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useId, useState } from "react";
import { observer } from "mobx-react";
import { CloseOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Checkbox } from "@makeplane/propel/components/checkbox";
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
import { cn } from "@nerve/utils";
// hooks
import { countOf } from "@/hooks/navigation-preferences";
import { useProjectNavigationPreferences } from "@/hooks/use-navigation-preferences";

type TProjectNavigationDialogProps = {
  isOpen: boolean;
  onClose: () => void;
};

/** Keeps out of the count field the keys a number's other characters are typed with: e, E, +, - and the point. */
function blockNonDigits(e: React.KeyboardEvent<HTMLInputElement>) {
  if (["e", "E", "+", "-", "."].includes(e.key)) e.preventDefault();
}

export const ProjectNavigationDialog = observer(function ProjectNavigationDialog(props: TProjectNavigationDialogProps) {
  const { isOpen, onClose } = props;
  const { t } = useTranslation();

  const countId = useId();

  // store hooks: the caller's settings as nerve gave them, and their change (use-navigation-preferences.ts)
  const { preferences: projectPreferences, changeNavigation } = useProjectNavigationPreferences();

  // the count field's text while the caller edits it, sent once he leaves the field; otherwise the field shows the
  // settings' count, as they arrive
  const [countDraft, setCountDraft] = useState<string | null>(null);
  const countText = countDraft ?? projectPreferences.limitedProjectsCount.toString();
  const countValid = countOf(countText) !== undefined;

  const sendCount = () => {
    if (countDraft === null) return;
    const count = countOf(countDraft);
    setCountDraft(null);
    if (count !== undefined) void changeNavigation({ limitedProjectsCount: count });
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
                    onChange={() => void changeNavigation({ navigationMode: "ACCORDION" })}
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
                    onChange={() => void changeNavigation({ navigationMode: "TABBED" })}
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
                    onCheckedChange={() => void changeNavigation({ limitToggled: true })}
                  />
                </div>

                {projectPreferences.showLimitedProjects && (
                  <div className="pl-8">
                    <div className="flex w-full flex-col gap-1">
                      <div className="flex w-full flex-col gap-2 pb-1.5">
                        <label htmlFor={countId} className="w-full text-11 text-secondary">
                          {t("enter_number_of_projects")}
                        </label>
                        <input
                          id={countId}
                          type="number"
                          min="1"
                          step="1"
                          value={countText}
                          onKeyDown={blockNonDigits}
                          onChange={(e) => setCountDraft(e.target.value.replace(/\D/g, ""))}
                          onBlur={sendCount}
                          className={cn(
                            "w-full rounded-md px-2 py-1 text-13",
                            "border bg-surface-2",
                            "text-secondary",
                            countValid
                              ? "border-strong focus:border-accent-strong focus:ring-1 focus:ring-accent-strong"
                              : "border-danger-strong focus:border-danger-strong focus:ring-1 focus:ring-danger-strong"
                          )}
                        />
                      </div>
                      {!countValid && countText !== "" && (
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
