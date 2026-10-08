/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useForm, Controller, useFieldArray } from "react-hook-form";
// nerve imports
import type { ProjectMembersAdd, ProjectRole } from "@nerve/api-client";
import { Avatar } from "@makeplane/propel/components/avatar";
import { ROLE, EUserPermissions } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { AddOutline, ChevronDownOutline, CloseOutline } from "@makeplane/propel/icons";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { CustomSelect, CustomSearchSelect, EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
// helpers
import { getFileURL } from "@nerve/utils";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useUserPermissions } from "@/hooks/store/user";
// local imports
import { PROJECT_ROLES } from "./project-roles";

type Props = {
  isOpen: boolean;
  onClose: () => void;
  projectId: string;
  workspaceSlug: string;
};

type FormValues = ProjectMembersAdd;

const defaultValues: FormValues = {
  members: [
    {
      role: 5,
      member_id: "",
    },
  ],
};

export const AddProjectMembersModal = observer(function AddProjectMembersModal(props: Props) {
  const { isOpen, onClose, projectId, workspaceSlug } = props;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const {
    project: { getProjectMemberDetails, bulkAddMembersToProject },
    workspace: { workspaceMemberIds, getWorkspaceMemberDetails },
  } = useMember();
  // form info
  const {
    formState: { errors, isSubmitting },
    watch,
    setValue,
    reset,
    handleSubmit,
    control,
  } = useForm<FormValues>({ defaultValues });
  const { fields, append, remove } = useFieldArray({
    control,
    name: "members",
  });
  // derived values
  const currentProjectRole = getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId);
  const nonProjectMemberIds = workspaceMemberIds?.filter(
    (userId) => getProjectMemberDetails(userId, projectId) === null
  );

  const onSubmit = async (formData: FormValues) => {
    if (!workspaceSlug || !projectId || isSubmitting) return;

    try {
      await bulkAddMembersToProject(projectId, formData);
      onClose();
      setToast({
        title: "Success!",
        type: TOAST_TYPE.SUCCESS,
        message: "Members added successfully.",
      });
    } catch (error) {
      console.error(error);
    } finally {
      reset(defaultValues);
    }
  };

  const handleClose = () => {
    onClose();

    const timeout = setTimeout(() => {
      reset(defaultValues);
      clearTimeout(timeout);
    }, 500);
  };

  const appendField = () => {
    append({
      role: 5,
      member_id: "",
    });
  };

  const options = nonProjectMemberIds?.flatMap((userId) => {
    const memberDetails = getWorkspaceMemberDetails(userId);

    if (!memberDetails?.member) return [];
    return [
      {
        value: `${memberDetails?.member.id}`,
        query: `${memberDetails?.member.first_name} ${
          memberDetails?.member.last_name
        } ${memberDetails?.member.display_name.toLowerCase()}`,
        content: (
          <div className="flex w-full items-center gap-2">
            <div className="shrink-0 pt-0.5">
              <Avatar
                alt={memberDetails?.member.display_name}
                fallback={memberDetails?.member.display_name?.[0]?.toUpperCase()}
                src={getFileURL(memberDetails?.member.avatar_url)}
                size="xs"
              />
            </div>
            <div className="truncate">
              {memberDetails?.member.display_name} (
              {memberDetails?.member.first_name + " " + memberDetails?.member.last_name})
            </div>
          </div>
        ),
      },
    ];
  });

  const checkCurrentOptionWorkspaceRole = (value: string): ProjectRole[] => {
    const currentMemberWorkspaceRole = getWorkspaceMemberDetails(value)?.role;
    if (!value || !currentMemberWorkspaceRole) return PROJECT_ROLES;

    const isGuestOROwner = [EUserPermissions.ADMIN, EUserPermissions.GUEST].includes(
      currentMemberWorkspaceRole as EUserPermissions
    );

    return PROJECT_ROLES.filter((role) => !isGuestOROwner || role === currentMemberWorkspaceRole);
  };

  return (
    <ModalCore isOpen={isOpen} handleClose={handleClose} position={EModalPosition.CENTER} width={EModalWidth.XXL}>
      <form onSubmit={handleSubmit(onSubmit)} className="p-5">
        <div className="space-y-5">
          <h3 className="text-16 leading-6 font-medium text-primary">
            {t("project_settings.members.add_members.title")}
          </h3>
          <div className="mt-2">
            <p className="text-13 text-secondary">{t("project_settings.members.add_members.sub_heading")}</p>
          </div>

          <div className="mb-3 space-y-4">
            {fields.map((field, index) => (
              <div key={field.id} className="group mb-1 flex w-full items-start justify-between gap-x-4 text-13">
                <div className="flex w-full grow flex-col gap-1">
                  <Controller
                    control={control}
                    name={`members.${index}.member_id`}
                    rules={{ required: "Please select a member" }}
                    render={({ field: { value, onChange } }) => {
                      const selectedMember = getWorkspaceMemberDetails(value);
                      return (
                        <CustomSearchSelect
                          value={value}
                          customButton={
                            <button className="shadow-sm flex w-full items-center justify-between gap-1 rounded-md border border-subtle px-3 py-2 text-left text-13 text-secondary duration-300 hover:bg-layer-1 hover:text-primary focus:outline-none">
                              {value && value !== "" ? (
                                <div className="flex items-center gap-2">
                                  <Avatar
                                    alt={selectedMember?.member.display_name}
                                    fallback={selectedMember?.member.display_name?.[0]?.toUpperCase()}
                                    src={getFileURL(selectedMember?.member.avatar_url ?? "")}
                                    size="xs"
                                  />
                                  {selectedMember?.member.display_name}
                                </div>
                              ) : (
                                <div className="flex items-center gap-2 py-0.5">Select co-worker</div>
                              )}
                              <ChevronDownOutline className="h-3 w-3" aria-hidden="true" />
                            </button>
                          }
                          onChange={(val: string) => {
                            onChange(val);
                            // Update the role to the workspace role when member ID changes
                            setValue(`members.${index}.role`, getWorkspaceMemberDetails(val)?.role ?? 5);
                          }}
                          options={options}
                          optionsClassName="w-48"
                        />
                      );
                    }}
                  />
                  {errors.members && errors.members[index]?.member_id && (
                    <span className="px-1 text-13 text-danger-primary">
                      {errors.members[index]?.member_id?.message}
                    </span>
                  )}
                </div>

                <div className="flex shrink-0 items-center justify-between gap-2">
                  <div className="flex flex-col gap-1">
                    <Controller
                      name={`members.${index}.role`}
                      control={control}
                      rules={{ required: "Select Role" }}
                      render={({ field: roleField }) => (
                        <CustomSelect
                          {...roleField}
                          customButton={
                            <div className="shadow-sm flex w-24 items-center justify-between gap-1 rounded-md border border-subtle px-3 py-2.5 text-left text-13 text-secondary duration-300 hover:bg-layer-1 hover:text-primary focus:outline-none">
                              <span className="capitalize">
                                {roleField.value ? ROLE[roleField.value] : "Select role"}
                              </span>
                              <ChevronDownOutline className="h-3 w-3" aria-hidden="true" />
                            </div>
                          }
                          input
                        >
                          {checkCurrentOptionWorkspaceRole(watch(`members.${index}.member_id`)).map((role) => {
                            if (role > (currentProjectRole ?? EUserPermissions.GUEST)) return null;

                            return (
                              <CustomSelect.Option key={role} value={role}>
                                {ROLE[role]}
                              </CustomSelect.Option>
                            );
                          })}
                        </CustomSelect>
                      )}
                    />
                    {errors.members && errors.members[index]?.role && (
                      <span className="px-1 text-13 text-danger-primary">{errors.members[index]?.role?.message}</span>
                    )}
                  </div>

                  {fields.length > 1 && (
                    <div className="flex-item flex w-6">
                      <button
                        type="button"
                        className="place-items-center self-center rounded-sm"
                        onClick={() => remove(index)}
                      >
                        <CloseOutline className="h-4 w-4 text-secondary" />
                      </button>
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
        <div className="mt-5 flex items-center justify-between gap-2">
          <button
            type="button"
            className="flex items-center gap-2 bg-transparent py-2 pr-3 text-13 font-medium text-accent-primary outline-accent-strong"
            onClick={appendField}
          >
            <AddOutline className="h-4 w-4" />
            {t("common.add_more")}
          </button>
          <div className="flex items-center gap-2">
            <Button variant="secondary" size="lg" onClick={handleClose}>
              {t("cancel")}
            </Button>
            <Button variant="primary" size="lg" type="submit" loading={isSubmitting}>
              {isSubmitting
                ? `${fields && fields.length > 1 ? `${t("add_members")}...` : `${t("add_member")}...`}`
                : `${fields && fields.length > 1 ? t("add_members") : t("add_member")}`}
            </Button>
          </div>
        </div>
      </form>
    </ModalCore>
  );
});
