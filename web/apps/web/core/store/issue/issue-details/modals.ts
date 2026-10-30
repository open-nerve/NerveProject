/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { makeAutoObservable } from "mobx";
import type { IssueLink } from "@nerve/api-client";
import type { TIssueRelationTypes, TWorkItemWidgets } from "@nerve/types";
export type TIssueRelationModal = {
  issueId: string | null;
  relationType: TIssueRelationTypes | null;
};

type TIssueCrudState = { toggle: boolean; parentIssueId: string | undefined };

export type TIssueCrudOperationState = {
  create: TIssueCrudState;
  existing: TIssueCrudState;
};

const initialCrud = (): TIssueCrudOperationState => ({
  create: { toggle: false, parentIssueId: undefined },
  existing: { toggle: false, parentIssueId: undefined },
});
/** Presentation state belongs to one mounted detail/peek, never to the entity cache. */
export class IssueDetailModals {
  active = false;
  relationKey: TIssueRelationTypes | null = null;
  issueLinkData: IssueLink | null = null;
  issueCrudOperationState = initialCrud();
  isCreateIssueModalOpen = false;
  isIssueLinkModalOpen = false;
  isParentIssueModalOpen: string | null = null;
  isDeleteIssueModalOpen: string | null = null;
  isRelationModalOpen: TIssueRelationModal | null = null;
  isSubIssuesModalOpen: string | null = null;
  constructor(private readonly widgetAction: (widget: TWorkItemWidgets) => void) {
    makeAutoObservable(this);
  }
  activate = () => {
    this.active = true;
  };
  retire = () => {
    this.active = false;
    this.relationKey = null;
    this.issueLinkData = null;
    this.issueCrudOperationState = initialCrud();
    this.isCreateIssueModalOpen = false;
    this.isIssueLinkModalOpen = false;
    this.isParentIssueModalOpen = null;
    this.isDeleteIssueModalOpen = null;
    this.isRelationModalOpen = null;
    this.isSubIssuesModalOpen = null;
  };
  get isAnyModalOpen() {
    return (
      this.isCreateIssueModalOpen ||
      this.isIssueLinkModalOpen ||
      !!this.isParentIssueModalOpen ||
      !!this.isDeleteIssueModalOpen ||
      !!this.isRelationModalOpen?.issueId ||
      !!this.isSubIssuesModalOpen
    );
  }
  setRelationKey = (value: TIssueRelationTypes | null) => {
    if (this.active) this.relationKey = value;
  };
  setIssueLinkData = (value: IssueLink | null) => {
    if (this.active) this.issueLinkData = value;
  };
  setIssueCrudOperationState = (value: TIssueCrudOperationState) => {
    if (this.active) this.issueCrudOperationState = value;
  };
  toggleCreateIssueModal = (value: boolean) => {
    if (this.active) this.isCreateIssueModalOpen = value;
  };
  toggleIssueLinkModal = (value: boolean) => {
    if (this.active) this.isIssueLinkModalOpen = value;
  };
  toggleParentIssueModal = (value: string | null) => {
    if (this.active) this.isParentIssueModalOpen = value;
  };
  toggleDeleteIssueModal = (value: string | null) => {
    if (this.active) this.isDeleteIssueModalOpen = value;
  };
  toggleSubIssuesModal = (value: string | null) => {
    if (this.active) this.isSubIssuesModalOpen = value;
  };
  toggleRelationModal = (issueId: string | null, relationType: TIssueRelationTypes | null) => {
    if (this.active) this.isRelationModalOpen = issueId && relationType ? { issueId, relationType } : null;
  };
  setLastWidgetAction = (widget: TWorkItemWidgets) => {
    if (this.active) this.widgetAction(widget);
  };
}
