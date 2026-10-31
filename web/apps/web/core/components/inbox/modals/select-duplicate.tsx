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
import { useTranslation } from "@nerve/i18n";
import { EModalPosition, EModalWidth, ModalCore, ModalTitle } from "@nerve/ui";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useIssueSearch } from "@/hooks/use-issue-search";
import { useRefusalMessage } from "@/hooks/use-refusal-toast";
import { IssueSearchModalEmptyState } from "@/components/core/modals/issue-search-modal-empty-state";
import { IssueIdentifier } from "@/components/issues/issue-detail/issue-identifier";
import { useDetailScope } from "@/components/issues/issue-detail-widgets/use-detail-scope";

type Props = {
  isOpen: boolean;
  value?: string | null;
  onClose: () => void;
  onSubmit: (issueId: string) => void;
};

export const SelectDuplicateInboxIssueModal = observer(function SelectDuplicateInboxIssueModal(props: Props) {
  const { workspaceSlug, projectId, issueId } = useParams();
  const { rootIssueStore: root } = useIssueDetail();
  const workspace = workspaceSlug ? root.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug) : undefined;
  const identity = root.permissionReads.identity(workspace?.id, projectId ? [projectId] : [], true);
  return props.isOpen ? (
    <DuplicateSelection key={JSON.stringify([workspaceSlug, projectId, issueId, identity])} {...props} />
  ) : null;
});

const DuplicateSelection = observer(function DuplicateSelection({ isOpen, value, onClose, onSubmit }: Props) {
  const { t } = useTranslation();
  const { workspaceSlug, projectId, issueId } = useParams();
  const { rootIssueStore: root } = useIssueDetail();
  const [term, setTerm] = useState("");
  // An intake item is not an Issue source. Its duplicate picker reads the project's candidates only.
  const search = useIssueSearch(
    root,
    workspaceSlug,
    { project_id: projectId, ...(term ? { q: term } : {}) },
    isOpen && !!projectId
  );
  const capture = useDetailScope(workspaceSlug ?? "", projectId, issueId ?? "inbox", true);
  const message = useRefusalMessage();
  const submitted = useRef(false);
  const issues = search.ids.flatMap((id) => {
    const issue = root.issues.getIssueById(id);
    return issue && issue.id !== issueId ? [issue] : [];
  });
  const choose = (id: string | null) => {
    if (!id || id === issueId || submitted.current || !projectId || !search.proof || !search.ids.includes(id)) return;
    const current = capture();
    const workspace = workspaceSlug ? root.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug) : undefined;
    const issue = root.issues.getIssueById(id);
    if (
      !current() ||
      !workspace ||
      !issue ||
      issue.project_id !== projectId ||
      issue.workspace_id !== workspace.id ||
      !root.isReadCurrent(search.proof, [projectId], workspace.id)
    )
      return;
    // This remains an ID notification; the intake consumer owns its separate business command.
    onSubmit(id);
    submitted.current = true;
    if (current()) onClose();
  };
  return (
    <ModalCore isOpen={isOpen} handleClose={onClose} position={EModalPosition.CENTER} width={EModalWidth.XXL}>
      <ModalTitle className="px-4 pt-4 text-16 font-medium">{t("inbox_issue.actions.mark_as_duplicate")}</ModalTitle>
      <Combobox as="div" value={value ?? null} onChange={choose}>
        <div className="relative m-1">
          <SearchOutline
            className="pointer-events-none absolute top-3.5 left-4 h-5 w-5 text-secondary"
            aria-hidden="true"
          />
          <Combobox.Input
            aria-label={t("common.search.placeholder")}
            placeholder={t("common.search.placeholder")}
            className="h-12 w-full bg-transparent pr-4 pl-11 text-13 outline-none"
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
        {search.loading && (
          <p role="status" className="p-3">
            {t("common.loading")}
          </p>
        )}
        {!search.loading && search.error === undefined && issues.length === 0 && (
          <IssueSearchModalEmptyState
            issues={issues}
            searchTerm={term}
            debouncedSearchTerm={term}
            isSearching={false}
          />
        )}
        <Combobox.Options
          as="ul"
          static
          modal={false}
          className="vertical-scrollbar max-h-80 overflow-y-auto p-2 text-13"
        >
          {issues.map((issue) => (
            <Combobox.Option
              as="li"
              key={issue.id}
              value={issue.id}
              className="my-0.5 flex cursor-pointer items-center gap-2 rounded-md px-3 py-2 text-secondary data-focus:bg-layer-1"
            >
              <span
                className="h-1.5 w-1.5 shrink-0 rounded-full"
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
