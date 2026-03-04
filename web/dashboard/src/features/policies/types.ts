// Policy types extended from shared types
import type { PrintPolicy, CreatePolicyRequest, PolicyConditions, PolicyActions } from '@/types';

export type { PrintPolicy, CreatePolicyRequest, PolicyConditions, PolicyActions };

export type PolicyStatus = 'enabled' | 'disabled' | 'all';
export type PolicyType = 'restriction' | 'enforcement' | 'approval' | 'routing';
export type PolicySort = 'priority' | 'name' | 'created' | 'updated';

export interface PolicyCondition {
  id: string;
  type: ConditionType;
  operator: ConditionOperator;
  value: string;
}

export type ConditionType =
  | 'userRole'
  | 'group'
  | 'printer'
  | 'time'
  | 'documentType'
  | 'pageCount'
  | 'fileSize'
  | 'colorMode'
  | 'always';

export type ConditionOperator =
  | 'equals'
  | 'notEquals'
  | 'contains'
  | 'notContains'
  | 'greaterThan'
  | 'lessThan'
  | 'between'
  | 'in'
  | 'notIn';

export interface PolicyAction {
  id: string;
  type: ActionType;
  parameter?: string;
  value?: boolean | number | string;
}

export type ActionType =
  | 'allow'
  | 'deny'
  | 'modify'
  | 'redirect'
  | 'notify'
  | 'blockJob'
  | 'forceDuplex'
  | 'forceGrayscale'
  | 'requireApproval'
  | 'setCopiesLimit'
  | 'routeToPrinter';

export interface PolicyTemplate {
  id: string;
  name: string;
  description: string;
  category: PolicyTemplateCategory;
  conditions: PolicyCondition[];
  actions: PolicyAction[];
  icon?: string;
}

export type PolicyTemplateCategory = 'security' | 'cost_control' | 'access_control' | 'quality';

export interface PolicyEvaluationContext {
  userId?: string;
  userRole?: string;
  printerId?: string;
  documentType?: string;
  pageCount?: number;
  colorMode?: boolean;
  fileSize?: number;
  timestamp?: string;
}

export interface PolicyEvaluationResult {
  matched: boolean;
  policyId?: string;
  policyName?: string;
  actions: PolicyAction[];
  message: string;
  modifiedSettings?: {
    duplex?: boolean;
    color?: boolean;
    copies?: number;
    approvalRequired?: boolean;
  };
}

export interface PolicyFilterOptions {
  status?: PolicyStatus;
  type?: PolicyType;
  search?: string;
  sortBy?: PolicySort;
}

export interface PolicyHistoryEntry {
  id: string;
  version: number;
  policyId: string;
  changedBy: string;
  changedAt: string;
  changes: string;
  data: Partial<PrintPolicy>;
}

export interface PolicyTestJob {
  documentName: string;
  pageCount: number;
  colorMode: boolean;
  copies: number;
  userRole?: string;
  printerId?: string;
}
