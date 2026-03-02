/**
 * TenantContext - React context for managing multi-tenant state
 *
 * Provides tenant/organization context throughout the application for:
 * - Current organization access
 * - Platform admin organization switching
 * - Tenant-scoped resource management
 */

import { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { authApi } from '@/services/api';
import type { Organization, ResourceQuota, User, OrganizationStatus } from '@/types';

// Extend User type to include platform admin flag
interface PlatformUser extends User {
  isPlatformAdmin?: boolean;
}

// ============================================================================
// Context Types
// ============================================================================

interface TenantContextValue {
  // Current organization (for regular users/org admins)
  currentOrganization: Organization | null;
  currentQuota: ResourceQuota | null;

  // Platform admin state
  isPlatformAdmin: boolean;
  selectedOrganizationId: string | null;
  allOrganizations: Organization[];

  // Actions
  switchOrganization: (orgId: string) => Promise<void>;
  refreshOrganization: () => Promise<void>;
  setCurrentOrganization: (org: Organization) => void;

  // Loading states
  isLoadingOrg: boolean;
  isLoadingQuota: boolean;
}

interface TenantContextProviderProps {
  children: ReactNode;
}

// ============================================================================
// Create Context
// ============================================================================

const TenantContext = createContext<TenantContextValue | undefined>(undefined);

// ============================================================================
// Provider Component
// ============================================================================

export const TenantProvider = ({ children }: TenantContextProviderProps) => {
  const queryClient = useQueryClient();
  const [isPlatformAdmin, setIsPlatformAdmin] = useState(false);
  const [selectedOrganizationId, setSelectedOrganizationId] = useState<string | null>(null);

  // Fetch current user to determine platform admin status
  const { data: user } = useQuery({
    queryKey: ['currentUser'],
    queryFn: async () => {
      const userData = await authApi.getCurrentUser() as PlatformUser;
      // Check if user is a platform admin (has 'platform_admin' role or special flag)
      setIsPlatformAdmin(userData.isPlatformAdmin || (userData.role as string) === 'platform_admin');
      return userData;
    },
    staleTime: 300000, // 5 minutes
  });

  // Fetch current organization for regular users
  const { data: currentOrganization, isLoading: isLoadingOrg, refetch: refetchOrg } = useQuery({
    queryKey: ['currentOrganization', selectedOrganizationId],
    queryFn: async () => {
      // For platform admin with selected org, fetch that org
      if (isPlatformAdmin && selectedOrganizationId) {
        const response = await fetch(`/api/v1/platform/organizations/${selectedOrganizationId}`, {
          headers: {
            Authorization: `Bearer ${localStorage.getItem('access_token')}`,
          },
        });
        if (!response.ok) throw new Error('Failed to fetch organization');
        return response.json() as Promise<Organization>;
      }
      // For regular users, fetch their organization
      const response = await fetch('/api/v1/organizations', {
        headers: {
          Authorization: `Bearer ${localStorage.getItem('access_token')}`,
        },
      });
      if (!response.ok) throw new Error('Failed to fetch organization');
      return response.json() as Promise<Organization>;
    },
    enabled: !!user && (!isPlatformAdmin || !!selectedOrganizationId),
    staleTime: 60000, // 1 minute
  });

  // Fetch quota information
  const { data: currentQuota, isLoading: isLoadingQuota } = useQuery({
    queryKey: ['organizationQuota', currentOrganization?.id],
    queryFn: async () => {
      const response = await fetch(`/api/v1/organizations/${currentOrganization?.id}/quota`, {
        headers: {
          Authorization: `Bearer ${localStorage.getItem('access_token')}`,
        },
      });
      if (!response.ok) throw new Error('Failed to fetch quota');
      return response.json() as Promise<ResourceQuota>;
    },
    enabled: !!currentOrganization,
    staleTime: 60000,
  });

  // Fetch all organizations for platform admin
  const { data: allOrganizations = [] } = useQuery({
    queryKey: ['allOrganizations'],
    queryFn: async () => {
      const response = await fetch('/api/v1/platform/organizations', {
        headers: {
          Authorization: `Bearer ${localStorage.getItem('access_token')}`,
        },
      });
      if (!response.ok) throw new Error('Failed to fetch organizations');
      const result = await response.json();
      return result.data || [];
    },
    enabled: isPlatformAdmin,
    staleTime: 120000, // 2 minutes
  });

  // Switch to a different organization (platform admin only)
  const switchOrganization = useCallback(async (orgId: string) => {
    setSelectedOrganizationId(orgId);
    // Clear related queries to force refetch
    queryClient.invalidateQueries({ queryKey: ['currentOrganization'] });
    queryClient.invalidateQueries({ queryKey: ['organizationQuota'] });
  }, [queryClient]);

  // Refresh current organization data
  const refreshOrganization = useCallback(async () => {
    await refetchOrg();
  }, [refetchOrg]);

  // Manually set current organization (for platform admin)
  const setCurrentOrganization = useCallback((org: Organization) => {
    queryClient.setQueryData(['currentOrganization', org.id], org);
  }, [queryClient]);

  // Auto-select first organization for platform admin
  useEffect(() => {
    if (isPlatformAdmin && !selectedOrganizationId && allOrganizations.length > 0) {
      setSelectedOrganizationId(allOrganizations[0].id);
    }
  }, [isPlatformAdmin, selectedOrganizationId, allOrganizations]);

  const value: TenantContextValue = {
    currentOrganization: currentOrganization || null,
    currentQuota: currentQuota || null,
    isPlatformAdmin,
    selectedOrganizationId,
    allOrganizations,
    switchOrganization,
    refreshOrganization,
    setCurrentOrganization,
    isLoadingOrg,
    isLoadingQuota,
  };

  return (
    <TenantContext.Provider value={value}>
      {children}
    </TenantContext.Provider>
  );
};

// ============================================================================
// Hook for using Tenant Context
// ============================================================================

export const useTenant = (): TenantContextValue => {
  const context = useContext(TenantContext);
  if (context === undefined) {
    throw new Error('useTenant must be used within a TenantProvider');
  }
  return context;
};

// ============================================================================
// Helper Hooks
// ============================================================================

/**
 * Hook to get the current organization ID
 */
export const useCurrentOrgId = (): string | null => {
  const { currentOrganization } = useTenant();
  return currentOrganization?.id || null;
};

/**
 * Hook to check if current user can manage organization settings
 */
export const useCanManageOrg = (): boolean => {
  const { currentOrganization } = useTenant();
  const { data: user } = useQuery<PlatformUser>({ queryKey: ['currentUser'] });

  if (!user || !currentOrganization) return false;

  const userRole = user.role;
  return userRole === 'owner' || userRole === 'admin';
};

/**
 * Hook to check quota utilization percentage
 */
export const useQuotaUtilization = (): {
  usersPercentage: number;
  printersPercentage: number;
  storagePercentage: number;
  jobsPercentage: number;
  isNearLimit: boolean;
} => {
  const { currentQuota } = useTenant();

  const calculatePercentage = (current: number, max: number): number => {
    if (!max || max <= 0) return 0;
    return Math.min(100, Math.round((current / max) * 100));
  };

  const usersPercentage = currentQuota
    ? calculatePercentage(currentQuota.currentUserCount, currentQuota.maxUsers)
    : 0;

  const printersPercentage = currentQuota
    ? calculatePercentage(currentQuota.currentPrinterCount, currentQuota.maxPrinters)
    : 0;

  const storagePercentage = currentQuota
    ? calculatePercentage(currentQuota.currentStorageGB, currentQuota.maxStorageGB)
    : 0;

  const jobsPercentage = currentQuota
    ? calculatePercentage(currentQuota.currentJobsThisMonth, currentQuota.maxJobsPerMonth)
    : 0;

  const isNearLimit =
    usersPercentage >= 80 ||
    printersPercentage >= 80 ||
    storagePercentage >= 80 ||
    jobsPercentage >= 80;

  return {
    usersPercentage,
    printersPercentage,
    storagePercentage,
    jobsPercentage,
    isNearLimit,
  };
};

/**
 * Hook to get organization display info
 */
export const useOrgDisplay = (): {
  name: string;
  slug: string;
  plan: string;
  status?: OrganizationStatus;
  logo?: string;
} => {
  const { currentOrganization } = useTenant();

  return {
    name: currentOrganization?.displayName || currentOrganization?.name || 'Organization',
    slug: currentOrganization?.slug || '',
    plan: currentOrganization?.plan || 'free',
    status: currentOrganization?.status,
    logo: currentOrganization?.settings?.branding?.logoUrl as string | undefined,
  };
};

export default TenantContext;
