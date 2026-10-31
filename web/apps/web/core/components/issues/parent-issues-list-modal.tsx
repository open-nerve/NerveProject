/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useRef, useState } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
import { Combobox } from "@headlessui/react";
import { SearchOutline } from "@makeplane/propel/icons";
import type { Issue } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Loader, EModalPosition, EModalWidth, ModalCore, ModalTitle } from "@nerve/ui";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useIssueSearch } from "@/hooks/use-issue-search";
import { useRefusalMessage } from "@/hooks/use-refusal-toast";
import { IssueSearchModalEmptyState } from "@/components/core/modals/issue-search-modal-empty-state";
import { IssueIdentifier } from "./issue-detail/issue-identifier";
import { useDetailScope } from "./issue-detail-widgets/use-detail-scope";

type Props = {
  isOpen: boolean;
  handleClose: () => void;
  value?: Issue | null;
  onChange: (issue: Issue) => void | Promise<void>;
  projectId: string | undefined;
  issueId?: string;
};

export const ParentIssuesListModal = observer(function ParentIssuesListModal(props: Props) {
  const { workspaceSlug } = useParams();
  const { rootIssueStore: root } = useIssueDetail();
  const workspace = workspaceSlug ? root.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug) : undefined;
  const permission = root.permissionReads.identity(workspace?.id, props.projectId ? [props.projectId] : [], true);
  return props.isOpen ? (
    <ParentIssuesListContent
      key={JSON.stringify([workspaceSlug, props.projectId, props.issueId, permission])}
      {...props}
    />
  ) : null;
});

const ParentIssuesListContent = observer(function ParentIssuesListContent({
  isOpen,
  handleClose: onClose,
  value,
  onChange,
  projectId,
  issueId,
}: Props) {
  const { t } = useTranslation();
  const { workspaceSlug } = useParams();
  const { rootIssueStore: root } = useIssueDetail();
  const [term, setTerm] = useState("");
  const search = useIssueSearch(
    root,
    workspaceSlug,
    {
      project_id: projectId,
      ...(issueId ? { for: "parent" as const, issue_id: issueId } : {}),
      ...(term ? { q: term } : {}),
    },
    isOpen && !!projectId
  );
  const capture = useDetailScope(workspaceSlug ?? "", projectId, issueId ?? "new", true);
  const message = useRefusalMessage();
  const busy = useRef(false);
  const [pending, setPending] = useState(false);
  const issues = search.ids.flatMap((id) => {
    const issue = root.issues.getIssueById(id);
    return issue ? [issue] : [];
  });
  const close = () => {
    if (!busy.current) onClose();
  };
  const choose = async (selected: Issue | null) => {
    if (!selected || busy.current || !projectId || !search.proof || !search.ids.includes(selected.id)) return;
    const current = capture();
    const workspace = workspaceSlug ? root.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug) : undefined;
    const issue = root.issues.getIssueById(selected.id);
    if (
      !current() ||
      !workspace ||
      !issue ||
      issue.project_id !== projectId ||
      issue.workspace_id !== workspace.id ||
      !root.isReadCurrent(search.proof, [projectId], workspace.id)
    )
      return;
    busy.current = true;
    setPending(true);
    try {
      await onChange(issue);
      if (current()) onClose();
    } catch {
      // The command owns its refusal. Keep the input and candidates for an explicit retry.
    } finally {
      busy.current = false;
      if (current()) setPending(false);
    }
  };
  return (
    <ModalCore isOpen={isOpen} handleClose={close} position={EModalPosition.CENTER} width={EModalWidth.XXL}>
      <ModalTitle className="px-4 pt-4 text-16 font-medium">{t("issue.add.parent")}</ModalTitle>
      <Combobox as="div" value={value ?? null} by="id" disabled={pending} onChange={choose}>
        <div className="relative m-1">
          <SearchOutline
            className="pointer-events-none absolute top-3.5 left-4 h-5 w-5 text-secondary"
            aria-hidden="true"
          />
          <Combobox.Input
            aria-label={t("common.search.placeholder")}
            className="h-12 w-full bg-transparent pr-4 pl-11 text-13 outline-none"
            placeholder={t("common.search.placeholder")}
            value={term}
            onChange={(event) => setTerm(event.target.value)}
          />
        </div>
        {search.error !== undefined && (
          <div role="alert" className="p-3">
            {message(search.error)}{" "}
            <button type="button" onClick={search.retry}>
              {t("common.retry")}
            </button>
          </div>
        )}
        {search.loading ? (
          <Loader className="space-y-3 p-3">
            <Loader.Item height="40px" />
            <Loader.Item height="40px" />
          </Loader>
        ) : search.error === undefined && issues.length === 0 ? (
          <IssueSearchModalEmptyState
            issues={issues}
            searchTerm={term}
            debouncedSearchTerm={term}
            isSearching={false}
          />
        ) : null}
        <Combobox.Options
          as="ul"
          static
          className="vertical-scrollbar scrollbar-md max-h-80 scroll-py-2 overflow-y-auto p-2"
        >
          {issues.map((issue) => (
            <Combobox.Option
              as="li"
              key={issue.id}
              value={issue}
              className="my-0.5 flex cursor-pointer items-center gap-2 rounded-md px-3 py-2 text-13 text-secondary select-none data-focus:bg-layer-1 data-focus:text-primary"
            >
              <span
                className="h-1.5 w-1.5 shrink-0 rounded-full bg-icon-placeholder"
                style={{ backgroundColor: root.rootStore.state.getStateById(issue.state_id)?.color }}
              />
              <IssueIdentifier projectId={issue.project_id} issueId={issue.id} size="xs" variant="secondary" />
              <span className="truncate">{issue.name}</span>
            </Combobox.Option>
          ))}
        </Combobox.Options>
      </Combobox>
    </ModalCore>
  );
});
