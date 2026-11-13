/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useId, useState } from "react";
import { Controller, useForm, useWatch } from "react-hook-form";
// nerve imports
import type { Module } from "@nerve/api-client";
import { Field, FieldError } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
import { TextArea, TextAreaGroup } from "@makeplane/propel/components/text-area";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { ModalTitle } from "@nerve/ui";
import { getDate, renderFormattedPayloadDate } from "@nerve/utils";
// components
import { DateRangeDropdown } from "@/components/dropdowns/date-range";
import { MemberDropdown } from "@/components/dropdowns/member/dropdown";
import { ProjectDropdown } from "@/components/dropdowns/project/dropdown";
// hooks
import { useUserPermissions } from "@/hooks/store/user/user-permissions";
// lib
import { useRefusedFocus } from "@/lib/refused-focus";
// local imports
import { ModuleStatusSelect } from "./select/status";

/**
 * What the form holds of a module: the project a new one goes to, its name, its description (plain text), its two
 * days, its status, its lead and its members.
 */
export type ModuleFormValues = Pick<
  Module,
  "project_id" | "name" | "description" | "start_date" | "target_date" | "status" | "lead_id" | "member_ids"
>;
/** The fields under which nerve's refusals of what the form sent show; it says the others in a toast. */
export type ModuleFormField = "name" | "start_date" | "target_date" | "lead_id" | "member_ids";
/** Where the form says a message of nerve's refusal: under the field (its setError). */
export type ModuleUnderField = (field: ModuleFormField, message: string) => void;

type Props = {
  /** The workspace of the projects a new module may go to, the route's. */
  workspaceSlug: string;
  /** The project a new module goes to until the caller picks another he may write; a change keeps its module's. */
  projectId: string;
  /** The module a change starts from, as the store holds it as the form opens. */
  module?: Module;
  /** Whether the form changes a module (else it makes one). */
  editing: boolean;
  /** Whether what the form sent is out: it is not sent again, nor the form cancelled, until nerve has answered. */
  busy: boolean;
  /** Sends what the form holds, given what it opened with; settles once the page has followed nerve's answer. */
  onSubmit: (values: ModuleFormValues, initial: ModuleFormValues, underField: ModuleUnderField) => Promise<void>;
  onCancel: () => void;
};

/**
 * A module's form (P6a spec 2.7): for a new one, the project among those whose modules the caller may write
 * (canWriteModules, as nerve decides: an admin's or a member's role, not one nerve shows only his own work items, and
 * its modules turned on; R7), then its name (required, 255 characters at most; it has the focus as the dialog opens),
 * its description, its two days (either may be left out, R13), its status (the backlog for a new one), its lead and its
 * members, among the members of its project who are not its guests (MemberDropdown, by the project; R6), which another
 * project chosen clears. Every control is in the natural Tab order. A field its rules refuse, or one nerve refuses,
 * shows its message under it and takes the focus (useRefusedFocus); nothing of what the form holds is reset by a
 * refusal.
 */
export function ModuleForm(props: Props) {
  const { workspaceSlug, projectId, module, editing, busy, onSubmit, onCancel } = props;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { canWriteModules } = useUserPermissions();
  // what the form opens with, which a change compares what it sends with
  const [initial] = useState<ModuleFormValues>(() => ({
    project_id: module?.project_id ?? projectId,
    name: module?.name ?? "",
    description: module?.description ?? "",
    start_date: module?.start_date ?? null,
    target_date: module?.target_date ?? null,
    status: module?.status ?? "backlog",
    lead_id: module?.lead_id ?? null,
    member_ids: module?.member_ids ?? [],
  }));
  // form data
  const {
    formState: { errors, submitCount },
    handleSubmit,
    control,
    setError,
    setValue,
  } = useForm<ModuleFormValues>({ defaultValues: initial, shouldFocusError: false });
  const formRef = useRefusedFocus<HTMLFormElement>(submitCount);
  // the project whose members the lead and the members are picked among
  const chosenProjectId = useWatch({ control, name: "project_id" });
  const datesMessageId = useId();
  const leadMessageId = useId();
  const membersMessageId = useId();
  // nerve refuses one day at a time (a target before the start): the range shows that one
  const datesMessage = errors.start_date?.message ?? errors.target_date?.message;
  const leadMessage = errors.lead_id?.message;
  const membersMessage = errors.member_ids?.message;
  const underField: ModuleUnderField = (field, message) => setError(field, { type: "server", message });

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
                    onChange={(picked) => {
                      if (picked === value) return;
                      onChange(picked);
                      // the lead and the members picked are the other project's members, whom nerve would refuse
                      setValue("lead_id", null);
                      setValue("member_ids", []);
                    }}
                    multiple={false}
                    buttonVariant="border-with-text"
                    renderCondition={(id) => canWriteModules(workspaceSlug, id)}
                  />
                </div>
              )}
            />
          )}
          <ModalTitle className="text-18 font-medium text-secondary">
            {editing ? t("project_module.update_module") : t("project_module.create_module")}
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
          <div className="flex flex-wrap items-start gap-2">
            <Controller
              control={control}
              name="start_date"
              render={({ field: { value: start, onChange: onStart } }) => (
                <Controller
                  control={control}
                  name="target_date"
                  render={({ field: { value: target, onChange: onTarget } }) => (
                    <Field name="dates" invalid={datesMessage !== undefined}>
                      <DateRangeDropdown
                        buttonVariant="border-with-text"
                        className="h-7"
                        value={{ from: getDate(start), to: getDate(target) }}
                        onSelect={(range) => {
                          onStart(renderFormattedPayloadDate(range?.from) ?? null);
                          onTarget(renderFormattedPayloadDate(range?.to) ?? null);
                        }}
                        placeholder={{ from: t("start_date"), to: t("end_date") }}
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
            <ModuleStatusSelect control={control} />
            <Controller
              control={control}
              name="lead_id"
              render={({ field: { value, onChange } }) => (
                <Field name="lead_id" invalid={leadMessage !== undefined}>
                  <div className="h-7">
                    <MemberDropdown
                      value={value}
                      onChange={onChange}
                      projectId={chosenProjectId}
                      multiple={false}
                      buttonVariant="border-with-text"
                      placeholder={t("lead")}
                      invalid={leadMessage !== undefined}
                      describedBy={leadMessage === undefined ? undefined : leadMessageId}
                    />
                  </div>
                  {leadMessage !== undefined && (
                    <FieldError id={leadMessageId} size="lg" match>
                      {leadMessage}
                    </FieldError>
                  )}
                </Field>
              )}
            />
            <Controller
              control={control}
              name="member_ids"
              render={({ field: { value, onChange } }) => (
                <Field name="member_ids" invalid={membersMessage !== undefined}>
                  <div className="h-7">
                    <MemberDropdown
                      value={value}
                      onChange={onChange}
                      projectId={chosenProjectId}
                      multiple
                      buttonVariant={value.length > 0 ? "transparent-without-text" : "border-with-text"}
                      buttonClassName={value.length > 0 ? "hover:bg-transparent px-0" : ""}
                      placeholder={t("members")}
                      invalid={membersMessage !== undefined}
                      describedBy={membersMessage === undefined ? undefined : membersMessageId}
                    />
                  </div>
                  {membersMessage !== undefined && (
                    <FieldError id={membersMessageId} size="lg" match>
                      {membersMessage}
                    </FieldError>
                  )}
                </Field>
              )}
            />
          </div>
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
              : t("project_module.update_module")
            : busy
              ? t("common.creating")
              : t("project_module.create_module")}
        </Button>
      </div>
    </form>
  );
}
