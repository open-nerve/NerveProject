/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ReactNode } from "react";
import { observer } from "mobx-react";
import { MembersOutline, StartDateOutline, UserAltOutline, WorkItemsOutline } from "@makeplane/propel/icons";
// nerve imports
import type { Module, Progress } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Loader } from "@nerve/ui";
// components
import { MemberDropdown } from "@/components/dropdowns/member/dropdown";
// hooks
import { useModules } from "@/hooks/store/use-modules";
// local imports
import { ModuleDates } from "../module-dates";
import { useModuleChange } from "../use-module-change";

type Props = {
  module: Module;
  projectId: string;
  /** nerve's progress of the module; undefined until nerve has read it. */
  progress: Pick<Progress, "total_issues" | "completed_issues" | "cancelled_issues"> | undefined;
  /** Whether the caller changes the module here (the sidebar's root decides it). */
  editable: boolean;
};

/**
 * The details of a module's progress sidebar (P6a spec 2.6): its days, its lead and its members, which the caller
 * changes there while editable, each by itself (useModuleChange); and its work items, the completed of those not
 * cancelled (progressPercent's counts, M6 design 3.5), as nerve's progress counts them.
 */
export const ModuleSidebarDetails = observer(function ModuleSidebarDetails(props: Props) {
  const { module, projectId, progress, editable } = props;
  // hooks
  const { t } = useTranslation();

  return (
    <dl className="flex flex-col gap-5 pt-2.5 pb-6">
      <Detail icon={<StartDateOutline className="h-4 w-4" />} title={t("date_range")}>
        <ModuleDates
          module={module}
          editable={editable}
          buttonContainerClassName="w-full"
          buttonVariant="background-with-text"
        />
      </Detail>
      <Detail icon={<UserAltOutline className="h-4 w-4" />} title={t("lead")}>
        <ModuleLead module={module} projectId={projectId} editable={editable} />
      </Detail>
      <Detail icon={<MembersOutline className="h-4 w-4" />} title={t("members")}>
        <ModuleMembers module={module} projectId={projectId} editable={editable} />
      </Detail>
      <Detail icon={<WorkItemsOutline className="h-4 w-4" />} title={t("work_items")}>
        {progress === undefined ? (
          <Loader>
            <Loader.Item height="20px" width="40px" />
          </Loader>
        ) : (
          <span className="px-1.5 text-13 text-tertiary">
            {`${progress.completed_issues}/${progress.total_issues - progress.cancelled_issues}`}
          </span>
        )}
      </Detail>
    </dl>
  );
});

/** A row of the details: its title, by its icon, and its value. */
function Detail({ icon, title, children }: { icon: ReactNode; title: string; children: ReactNode }) {
  return (
    <div className="flex items-center justify-start gap-1">
      <dt className="flex w-2/5 items-center justify-start gap-2 text-tertiary">
        {icon}
        <span className="text-14">{title}</span>
      </dt>
      <dd className="flex h-7 w-3/5 items-center">{children}</dd>
    </div>
  );
}

type Control = { module: Module; editable: boolean };

/** The module's lead, one of the project's members who is not its guest, which the caller changes while editable. */
const ModuleLead = observer(function ModuleLead({ module, projectId, editable }: Control & { projectId: string }) {
  // hooks
  const { t } = useTranslation();
  const { update } = useModules();
  const { busy, send } = useModuleChange();

  return (
    <MemberDropdown
      value={module.lead_id}
      onChange={(lead_id: string | null) => {
        if (lead_id !== module.lead_id) void send(() => update(module.id, { lead_id }));
      }}
      projectId={projectId}
      multiple={false}
      buttonVariant="background-with-text"
      placeholder={t("lead")}
      disabled={!editable || busy}
      icon={UserAltOutline}
    />
  );
});

/**
 * The module's members, the project's who are not its guests, whom the caller adds or removes while editable (R6): a
 * click of the list adds one or removes one, and the store makes the whole set of nerve's in its turn (changeMembers),
 * so that a change sent before it is not lost.
 */
const ModuleMembers = observer(function ModuleMembers({
  module,
  projectId,
  editable,
}: Control & { projectId: string }) {
  // hooks
  const { changeMembers } = useModules();
  const { busy, send } = useModuleChange();
  const members = module.member_ids;

  return (
    <MemberDropdown
      value={members}
      onChange={(picked: string[]) => {
        const add = picked.filter((id) => !members.includes(id));
        const remove = members.filter((id) => !picked.includes(id));
        if (add.length > 0 || remove.length > 0) void send(() => changeMembers(module.id, { add, remove }));
      }}
      multiple
      projectId={projectId}
      buttonVariant={members.length > 0 ? "transparent-without-text" : "background-with-text"}
      buttonClassName={members.length > 0 ? "hover:bg-transparent px-0" : ""}
      disabled={!editable || busy}
    />
  );
});
