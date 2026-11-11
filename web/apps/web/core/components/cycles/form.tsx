/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useId, useState } from "react";
import { Controller, useForm } from "react-hook-form";
// nerve imports
import type { Cycle } from "@nerve/api-client";
import { Field, FieldError } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
import { TextArea, TextAreaGroup } from "@makeplane/propel/components/text-area";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { ModalTitle } from "@nerve/ui";
import { getDate, renderFormattedPayloadDate } from "@nerve/utils";
// components
import { DateRangeDropdown } from "@/components/dropdowns/date-range";
import { ProjectDropdown } from "@/components/dropdowns/project/dropdown";
// hooks
import { useUserPermissions } from "@/hooks/store/user/user-permissions";
// lib
import { useRefusedFocus } from "@/lib/refused-focus";

/** What the form holds of a cycle: the project a new one goes to, its name, its description (plain text), its days. */
export type CycleFormValues = Pick<Cycle, "project_id" | "name" | "description" | "start_date" | "end_date">;
/** The fields under which nerve's refusals of what the form sent show; it says the others in a toast. */
export type CycleFormField = "name" | "start_date" | "end_date";
/** Where the form says a message of nerve's refusal: under the field (its setError). */
export type CycleUnderField = (field: CycleFormField, message: string) => void;

type Props = {
  /** The workspace of the projects a new cycle may go to, the route's. */
  workspaceSlug: string;
  /** The project a new cycle goes to until the caller picks another he may write; a change keeps its cycle's. */
  projectId: string;
  /** The cycle a change starts from, as the store holds it as the form opens. */
  cycle?: Cycle;
  /** Whether the form changes a cycle (else it makes one). */
  editing: boolean;
  /** Whether what the form sent is out: it is not sent again, nor the form cancelled, until nerve has answered. */
  busy: boolean;
  /** Sends what the form holds, given what it opened with; settles once the page has followed nerve's answer. */
  onSubmit: (values: CycleFormValues, initial: CycleFormValues, underField: CycleUnderField) => Promise<void>;
  onCancel: () => void;
};

/**
 * A cycle's form (P5a spec 2.6): for a new one, the project among those whose cycles the caller may write
 * (canWriteCycles, as nerve decides: an admin's or a member's role, not one nerve shows only his own work items, and
 * its cycles turned on), then its name (required, 255 characters at most; it has the focus as the dialog opens), its
 * description and its two days, which the calendar offers from the browser's today on. Every control is in the natural
 * Tab order. A name the rules refuse, or a field nerve refuses, shows its message under it and takes the focus
 * (useRefusedFocus).
 */
export function CycleForm(props: Props) {
  const { workspaceSlug, projectId, cycle, editing, busy, onSubmit, onCancel } = props;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { canWriteCycles } = useUserPermissions();
  // what the form opens with, which a change compares what it sends with
  const [initial] = useState<CycleFormValues>(() => ({
    project_id: cycle?.project_id ?? projectId,
    name: cycle?.name ?? "",
    description: cycle?.description ?? "",
    start_date: cycle?.start_date ?? null,
    end_date: cycle?.end_date ?? null,
  }));
  // form data
  const {
    formState: { errors, submitCount },
    handleSubmit,
    control,
    setError,
  } = useForm<CycleFormValues>({ defaultValues: initial, shouldFocusError: false });
  const formRef = useRefusedFocus<HTMLFormElement>(submitCount);
  const datesMessageId = useId();
  // nerve refuses one day at a time (one given alone, or an end before the start): the range shows that one
  const datesMessage = errors.start_date?.message ?? errors.end_date?.message;
  const underField: CycleUnderField = (field, message) => setError(field, { type: "server", message });

  return (
    <form
      ref={formRef}
      onSubmit={(event) => void handleSubmit((values) => onSubmit(values, initial, underField))(event)}
    >
      <div className="space-y-5 p-5">
        <div className="flex items-center gap-x-3">
          {!editing && (
            <Controller
              control={control}
              name="project_id"
              render={({ field: { value, onChange } }) => (
                <div className="h-7">
                  <ProjectDropdown
                    value={value}
                    onChange={onChange}
                    multiple={false}
                    buttonVariant="border-with-text"
                    renderCondition={(id) => canWriteCycles(workspaceSlug, id)}
                  />
                </div>
              )}
            />
          )}
          <ModalTitle className="text-18 font-medium text-secondary">
            {editing ? t("project_cycles.update_cycle") : t("project_cycles.create_cycle")}
          </ModalTitle>
        </div>
        <div className="space-y-3">
          <Controller
            name="name"
            control={control}
            rules={{
              required: t("title_is_required"),
              maxLength: {
                value: 255,
                message: t("title_should_be_less_than_255_characters"),
              },
            }}
            render={({ field: { value, onChange } }) => (
              <Field name="name" invalid={Boolean(errors.name)}>
                <InputGroup size="2xl">
                  <Input
                    size="2xl"
                    name="name"
                    type="text"
                    placeholder={t("title")}
                    value={value}
                    onChange={onChange}
                    data-autofocus
                    aria-label={t("title")}
                  />
                </InputGroup>
                {errors.name?.message && (
                  <FieldError size="2xl" match>
                    {errors.name.message}
                  </FieldError>
                )}
              </Field>
            )}
          />
          <Controller
            name="description"
            control={control}
            render={({ field: { value, onChange } }) => (
              <Field name="description">
                <TextAreaGroup resize="none">
                  <TextArea
                    size="lg"
                    surface="field"
                    autoResize
                    maxRows={8}
                    name="description"
                    placeholder={t("description")}
                    value={value}
                    onChange={onChange}
                  />
                </TextAreaGroup>
              </Field>
            )}
          />
          <Controller
            control={control}
            name="start_date"
            render={({ field: { value: start, onChange: onStart } }) => (
              <Controller
                control={control}
                name="end_date"
                render={({ field: { value: end, onChange: onEnd } }) => (
                  <Field name="dates" invalid={datesMessage !== undefined}>
                    <DateRangeDropdown
                      buttonVariant="border-with-text"
                      className="h-7"
                      minDate={new Date()}
                      value={{ from: getDate(start), to: getDate(end) }}
                      onSelect={(range) => {
                        onStart(renderFormattedPayloadDate(range?.from) ?? null);
                        onEnd(renderFormattedPayloadDate(range?.to) ?? null);
                      }}
                      placeholder={{ from: t("project_cycles.start_date"), to: t("project_cycles.end_date") }}
                      hideIcon={{ to: true }}
                      invalid={datesMessage !== undefined}
                      describedBy={datesMessage === undefined ? undefined : datesMessageId}
                    />
                    {datesMessage !== undefined && (
                      <FieldError id={datesMessageId} size="lg" match>
                        {datesMessage}
                      </FieldError>
                    )}
                  </Field>
                )}
              />
            )}
          />
        </div>
      </div>
      <div className="flex items-center justify-end gap-2 border-t-[0.5px] border-subtle px-5 py-4">
        <Button type="button" variant="secondary" size="lg" onClick={onCancel} disabled={busy}>
          {t("common.cancel")}
        </Button>
        <Button variant="primary" size="lg" type="submit" loading={busy}>
          {editing
            ? busy
              ? t("common.updating")
              : t("project_cycles.update_cycle")
            : busy
              ? t("common.creating")
              : t("project_cycles.create_cycle")}
        </Button>
      </div>
    </form>
  );
}
