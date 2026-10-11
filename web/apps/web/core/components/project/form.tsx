/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { InfoOutline, LockOutline } from "@makeplane/propel/icons";
import { Field } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
import { TextArea, TextAreaGroup } from "@makeplane/propel/components/text-area";
import { NETWORK_CHOICES } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
// nerve imports
import { Button } from "@nerve/propel/button";
import { EmojiPicker, EmojiIconPickerTypes, Logo } from "@nerve/propel/emoji-icon-picker";
import { Tooltip } from "@makeplane/propel/components/tooltip";
import type { Project } from "@nerve/api-client";
import { CustomSelect } from "@nerve/ui";
import { projectIdentifierSanitizer, renderFormattedDate } from "@nerve/utils";
import { CoverImage } from "@/components/common/cover-image";
import { TimezoneSelect } from "@/components/global";
// hooks
import { usePlatformOS } from "@/hooks/use-platform-os";
// local imports
import { logoPropsOf } from "./logo-props";
import { ProjectNetworkIcon } from "./project-network-icon";
import { projectDetailsOf, useUpdateProjectDetails, type ProjectDetails } from "./use-update-project-details";

export interface IProjectDetailsForm {
  project: Project;
  workspaceSlug: string;
  isAdmin: boolean;
}

/**
 * A project's general settings (M3 design 7.6), as nerve last answered them when the page opened them: its page mounts
 * one per project. Its admin changes them; anyone else sees them, in a form he cannot change.
 */
export function ProjectDetailsForm(props: IProjectDetailsForm) {
  const { project, workspaceSlug, isAdmin } = props;
  const { t } = useTranslation();
  // states
  const [isOpen, setIsOpen] = useState(false);
  // store hooks
  const updateDetails = useUpdateProjectDetails();
  const { isMobile } = usePlatformOS();

  // form info
  const {
    handleSubmit,
    watch,
    control,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<ProjectDetails>({ defaultValues: projectDetailsOf(project) });
  // derived values
  const currentNetwork = NETWORK_CHOICES.find((n) => n.key === project.network);

  // the page is busy until nerve has answered and the page has followed the answer
  const onSubmit = (details: ProjectDetails) => updateDetails(project, workspaceSlug, details, setError);

  return (
    <form onSubmit={(event) => void handleSubmit(onSubmit)(event)}>
      <div className="relative h-44 w-full">
        <div className="absolute inset-0 bg-gradient-to-t from-black/50 to-transparent" />
        <CoverImage
          src={project.cover_image_url ?? undefined}
          showDefaultWhenEmpty
          alt="Project cover image"
          className="h-44 w-full rounded-md"
        />
        <div className="absolute bottom-4 z-5 flex w-full items-end justify-between gap-3 px-4">
          <div className="flex flex-grow gap-3 truncate">
            <Controller
              control={control}
              name="logo_props"
              render={({ field: { value, onChange } }) => (
                <EmojiPicker
                  ariaLabel={t("aria_labels.project_icon")}
                  iconType="material"
                  closeOnSelect={false}
                  isOpen={isOpen}
                  handleToggle={(val: boolean) => setIsOpen(val)}
                  className="flex items-center justify-center"
                  buttonClassName="flex h-[52px] w-[52px] flex-shrink-0 items-center justify-center rounded-lg bg-white/10"
                  label={<Logo logo={value} size={28} />}
                  onChange={(picked) => {
                    onChange(logoPropsOf(picked));
                    setIsOpen(false);
                  }}
                  defaultIconColor={value.in_use === "icon" ? value.icon?.color : undefined}
                  defaultOpen={value.in_use === "emoji" ? EmojiIconPickerTypes.EMOJI : EmojiIconPickerTypes.ICON}
                  disabled={!isAdmin}
                />
              )}
            />
            <div className="flex flex-col gap-1 truncate text-on-color">
              <span className="truncate text-16 font-semibold">{watch("name")}</span>
              <span className="flex items-center gap-2 text-13">
                <span>{watch("identifier")} .</span>
                <span className="flex items-center gap-1.5">
                  {project.network === 0 && <LockOutline className="h-2.5 w-2.5 text-on-color" />}
                  {currentNetwork && t(currentNetwork.i18n_label)}
                </span>
              </span>
            </div>
          </div>
        </div>
      </div>
      <div className="mt-8 flex flex-col gap-8">
        <div className="flex flex-col gap-1">
          <h4 className="text-13">{t("common.project_name")}</h4>
          <Controller
            control={control}
            name="name"
            rules={{
              required: t("name_is_required"),
              maxLength: {
                value: 255,
                message: "Project name should be less than 255 characters",
              },
            }}
            render={({ field: { value, onChange, ref } }) => (
              <Field name="name" invalid={Boolean(errors.name)}>
                <InputGroup size="2xl">
                  <Input
                    size="2xl"
                    id="name"
                    name="name"
                    type="text"
                    ref={ref}
                    value={value}
                    onChange={onChange}
                    placeholder={t("common.project_name")}
                    disabled={!isAdmin}
                  />
                </InputGroup>
              </Field>
            )}
          />
          <span className="text-11 text-danger-primary">{errors.name?.message}</span>
        </div>
        <div className="flex flex-col gap-1">
          <h4 className="text-13">{t("description")}</h4>
          <Controller
            name="description"
            control={control}
            render={({ field: { value, onChange } }) => (
              <Field name="description" invalid={Boolean(errors.description)} disabled={!isAdmin}>
                <TextAreaGroup resize="none">
                  <TextArea
                    size="lg"
                    surface="field"
                    autoResize
                    maxRows={8}
                    id="description"
                    name="description"
                    value={value}
                    placeholder={t("project_description_placeholder")}
                    onChange={onChange}
                  />
                </TextAreaGroup>
              </Field>
            )}
          />
        </div>
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
          <div className="flex flex-col gap-1">
            <h4 className="text-13">Project ID</h4>
            <div className="relative">
              <Controller
                control={control}
                name="identifier"
                rules={{
                  required: t("project_id_is_required"),
                  validate: (value) => /^[ÇŞĞIİÖÜA-Z0-9]+$/.test(value.toUpperCase()) || t("project_id_allowed_char"),
                  minLength: {
                    value: 1,
                    message: t("project_id_min_char"),
                  },
                  maxLength: {
                    value: 10,
                    message: t("project_id_max_char"),
                  },
                }}
                render={({ field: { value, onChange, ref } }) => (
                  <Field name="identifier" invalid={Boolean(errors.identifier)}>
                    <InputGroup size="2xl">
                      <Input
                        size="2xl"
                        id="identifier"
                        name="identifier"
                        type="text"
                        value={value}
                        // upper case as typed, of the characters an identifier may have, as the creation's form
                        onChange={(event) => onChange(projectIdentifierSanitizer(event.target.value))}
                        ref={ref}
                        placeholder={t("project_settings.general.enter_project_id")}
                        disabled={!isAdmin}
                      />
                    </InputGroup>
                  </Field>
                )}
              />
              <Tooltip
                label={t("project_id_tooltip_content")}
                layout="stacked"
                side="right"
                align="start"
                disabled={isMobile}
              >
                <InfoOutline className="absolute top-2.5 right-2 h-4 w-4 text-placeholder" />
              </Tooltip>
            </div>
            <span className="text-11 text-danger-primary">{errors.identifier?.message}</span>
          </div>
          <div className="flex flex-col gap-1">
            <h4 className="text-13">{t("workspace_projects.network.label")}</h4>
            <Controller
              name="network"
              control={control}
              render={({ field: { value, onChange } }) => {
                const selectedNetwork = NETWORK_CHOICES.find((n) => n.key === value);
                return (
                  <CustomSelect
                    value={value}
                    onChange={onChange}
                    label={
                      <div className="flex items-center gap-1">
                        {selectedNetwork ? (
                          <>
                            <ProjectNetworkIcon iconKey={selectedNetwork.iconKey} className="h-3.5 w-3.5" />
                            {t(selectedNetwork.i18n_label)}
                          </>
                        ) : (
                          <span className="text-placeholder">{t("select_network")}</span>
                        )}
                      </div>
                    }
                    buttonClassName="!border-subtle !shadow-none font-medium rounded-md"
                    input
                    disabled={!isAdmin}
                  >
                    {NETWORK_CHOICES.map((network) => (
                      <CustomSelect.Option key={network.key} value={network.key}>
                        <div className="flex items-start gap-2">
                          <ProjectNetworkIcon iconKey={network.iconKey} className="h-3.5 w-3.5" />
                          <div className="-mt-1">
                            <p>{t(network.i18n_label)}</p>
                            <p className="text-11 text-placeholder">{t(network.description)}</p>
                          </div>
                        </div>
                      </CustomSelect.Option>
                    ))}
                  </CustomSelect>
                );
              }}
            />
          </div>
          <div className="col-span-1 flex flex-col gap-1 sm:col-span-2 xl:col-span-1">
            <h4 className="text-13">{t("common.project_timezone")}</h4>
            <Controller
              name="timezone"
              control={control}
              rules={{ required: t("project_settings.general.please_select_a_timezone") }}
              render={({ field: { value, onChange } }) => (
                <TimezoneSelect
                  value={value}
                  onChange={onChange}
                  error={Boolean(errors.timezone)}
                  buttonClassName="!border-subtle !shadow-none font-medium rounded-md"
                  disabled={!isAdmin}
                />
              )}
            />
            {errors.timezone && <span className="text-11 text-danger-primary">{errors.timezone.message}</span>}
          </div>
        </div>
        <div className="flex items-center justify-between py-2">
          <Button variant="primary" size="lg" type="submit" loading={isSubmitting} disabled={!isAdmin}>
            {isSubmitting ? t("updating") : t("common.update_project")}
          </Button>
          <span className="text-13 text-placeholder italic">
            {t("common.created_on")} {renderFormattedDate(project.created_at)}
          </span>
        </div>
      </div>
    </form>
  );
}
