import { useQuery } from '@tanstack/react-query';
import { policyApi } from './api';
import type { PolicyTemplate } from './types';

/**
 * Hook for fetching policy templates
 */
export function usePolicyTemplates() {
  return useQuery({
    queryKey: ['policy-templates'],
    queryFn: () => policyApi.getTemplates(),
  });
}

/**
 * Hook for fetching a single policy template
 */
export function usePolicyTemplate(id: string) {
  return useQuery({
    queryKey: ['policy-templates', id],
    queryFn: () => policyApi.getTemplate(id),
    enabled: !!id,
  });
}

/**
 * Get templates by category
 */
export function useTemplatesByCategory() {
  const { data: templates, ...rest } = usePolicyTemplates();

  const categorized = templates?.reduce((acc, template) => {
    if (!acc[template.category]) {
      acc[template.category] = [];
    }
    acc[template.category].push(template);
    return acc;
  }, {} as Record<string, PolicyTemplate[]>);

  return {
    ...rest,
    categorized,
  };
}
