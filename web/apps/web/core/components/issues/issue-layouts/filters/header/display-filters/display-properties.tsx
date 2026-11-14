/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
// nerve constants
import { ISSUE_DISPLAY_PROPERTIES } from "@nerve/constants";
// nerve i18n
import { useTranslation } from "@nerve/i18n";
// types
import type { IIssueDisplayProperties } from "@nerve/types";
// hooks
import { useProject } from "@/hooks/store/use-project";
// lib
import { displayPropertyOn } from "@/lib/planning-switch";
// components
import { FilterHeader } from "../helpers/filter-header";

type Props = {
  displayProperties: IIssueDisplayProperties;
  displayPropertiesToRender: (keyof IIssueDisplayProperties)[];
  handleUpdate: (updatedDisplayProperties: Partial<IIssueDisplayProperties>) => void;
};

/**
 * The display properties a list page offers (P7a spec 2.5, R11): those of its options; on a page of a project (its
 * list, a cycle's, a module's: the route names the project) the cycle and the modules only while the route's project
 * has their domain turned on, the page's own domain too; on a page of no project (the member view, whose rows are of
 * many projects) both.
 */
export const FilterDisplayProperties = observer(function FilterDisplayProperties(props: Props) {
  const { displayProperties, displayPropertiesToRender, handleUpdate } = props;
  // router
  const { projectId } = useParams();
  // hooks
  const { t } = useTranslation();
  const { getProjectById } = useProject();
  // states
  const [previewEnabled, setPreviewEnabled] = React.useState(true);

  const project = projectId ? getProjectById(projectId) : undefined;
  const offered = (key: keyof IIssueDisplayProperties) => !projectId || displayPropertyOn(project, key);
  const filteredDisplayProperties = ISSUE_DISPLAY_PROPERTIES.filter(
    (property) => displayPropertiesToRender.includes(property.key) && offered(property.key)
  );

  return (
    <>
      <FilterHeader
        title={t("issue.display.properties.label")}
        isPreviewEnabled={previewEnabled}
        handleIsPreviewEnabled={() => setPreviewEnabled(!previewEnabled)}
      />
      {previewEnabled && (
        <div className="mt-1 flex flex-wrap items-center gap-2">
          {filteredDisplayProperties.map((displayProperty) => (
            <button
              key={displayProperty.key}
              type="button"
              aria-pressed={!!displayProperties?.[displayProperty.key]}
              className={`rounded-sm border px-2 py-0.5 text-11 transition-all ${
                displayProperties?.[displayProperty.key]
                  ? "border-accent-strong bg-accent-primary text-on-color"
                  : "border-subtle hover:bg-layer-1"
              }`}
              onClick={() =>
                handleUpdate({
                  [displayProperty.key]: !displayProperties?.[displayProperty.key],
                })
              }
            >
              {t(displayProperty.titleTranslationKey)}
            </button>
          ))}
        </div>
      )}
    </>
  );
});
