/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { IssueFormValues } from "../types";
import { useId } from "react";
import { observer } from "mobx-react";
import type { Control } from "react-hook-form";
import { Controller, useWatch } from "react-hook-form";
import { Field, FieldError } from "@makeplane/propel/components/field";
import { useTranslation } from "@nerve/i18n";
import { CycleDropdown } from "@/components/dropdowns/cycle/dropdown";
import { ModuleDropdown } from "@/components/dropdowns/module/dropdown";
import { useProject } from "@/hooks/store/use-project";
import { usePlanningNames } from "@/hooks/use-planning-names";
import { modulesAfter } from "@/lib/planning-directory";
import { planningOn } from "@/lib/planning-switch";

type Props = {
  control: Control<IssueFormValues>;
  workspaceSlug: string;
  /** The form's project: its switches turn the fields on, its cycles and modules are their options. */
  projectId: string | null;
  /** Whether the form waits for its project's lists: the fields show loading, holding their values. */
  loading: boolean;
  handleFormChange: () => void;
};

/**
 * A work item's cycle and modules in its form (P7a spec 2.6, R11): each while the form's project has its domain turned
 * on, by their names, an ended or an archived one too (captioned); the project's lists are read for a name the store
 * does not have (usePlanningNames, R14). A pick changes the form's own value, which is what the form sends: the cycle
 * picked, none for No cycle; the module added to the form's set or taken out of it (modulesAfter). The values are the
 * form's start (the modal's, before the form mounts), never filled in here; loading while the form waits for its
 * project's lists. nerve's refusal of a field's value shows under it and describes it (R17).
 */
export const IssuePlanningFields = observer(function IssuePlanningFields(props: Props) {
  const { control, workspaceSlug, projectId, loading, handleFormChange } = props;
  const { t } = useTranslation();
  const id = useId();
  const { getProjectById } = useProject();
  const project = getProjectById(projectId ?? undefined);
  const cycleId = useWatch({ control, name: "cycle_id" });
  const moduleIds = useWatch({ control, name: "module_ids" });
  usePlanningNames(workspaceSlug, project?.id, "cycles", cycleId ? [cycleId] : []);
  usePlanningNames(workspaceSlug, project?.id, "modules", moduleIds);
  if (!project) return null;
  return (
    <>
      {planningOn(project, "cycles") && (
        <Controller
          control={control}
          name="cycle_id"
          render={({ field: { value, onChange }, fieldState: { error } }) => (
            <Field name="cycle_id" invalid={!!error}>
              <div className="h-7">
                <CycleDropdown
                  value={value}
                  onChange={(picked) => {
                    onChange(picked);
                    handleFormChange();
                  }}
                  projectId={project.id}
                  isInitializing={loading}
                  placeholder={t("cycle.label", { count: 1 })}
                  buttonVariant="border-with-text"
                  invalid={!!error}
                  describedBy={error ? `${id}-cycle_id` : undefined}
                />
              </div>
              {error?.message && (
                <FieldError id={`${id}-cycle_id`} size="lg" match>
                  {error.message}
                </FieldError>
              )}
            </Field>
          )}
        />
      )}
      {planningOn(project, "modules") && (
        <Controller
          control={control}
          name="module_ids"
          render={({ field: { value, onChange }, fieldState: { error } }) => (
            <Field name="module_ids" invalid={!!error}>
              <div className="h-7">
                <ModuleDropdown
                  value={value}
                  onChange={(change) => {
                    onChange(modulesAfter(value, change));
                    handleFormChange();
                  }}
                  projectId={project.id}
                  isInitializing={loading}
                  buttonVariant="border-with-text"
                  invalid={!!error}
                  describedBy={error ? `${id}-module_ids` : undefined}
                />
              </div>
              {error?.message && (
                <FieldError id={`${id}-module_ids`} size="lg" match>
                  {error.message}
                </FieldError>
              )}
            </Field>
          )}
        />
      )}
    </>
  );
});
