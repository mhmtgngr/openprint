import type { Printer, PrinterPermission, PermissionType, PaginatedResponse, PrinterSupply } from '@/types';
import { httpClient } from '@/services/http';

export const printersApi = {
  async list(): Promise<Printer[]> {
    const result = await httpClient.get<PaginatedResponse<Printer>>('/printers');
    return result.data || [];
  },

  async get(id: string): Promise<Printer> {
    return httpClient.get<Printer>(`/printers/${id}`);
  },

  async update(id: string, data: Partial<Printer>): Promise<Printer> {
    return httpClient.patch<Printer>(`/printers/${id}`, data);
  },

  async delete(id: string): Promise<void> {
    return httpClient.delete<void>(`/printers/${id}`);
  },

  async getPermissions(printerId: string): Promise<PrinterPermission[]> {
    return httpClient.get<PrinterPermission[]>(`/printers/${printerId}/permissions`);
  },

  async grantPermission(
    printerId: string,
    userId: string,
    permissionType: PermissionType
  ): Promise<PrinterPermission> {
    return httpClient.post<PrinterPermission>(`/printers/${printerId}/permissions`, {
      userId,
      permissionType,
    });
  },

  async revokePermission(printerId: string, userId: string): Promise<void> {
    return httpClient.delete<void>(`/printers/${printerId}/permissions/${userId}`);
  },

  async getSupplies(printerId: string): Promise<PrinterSupply[]> {
    return httpClient.get<PrinterSupply[]>(`/printers/${printerId}/supplies`);
  },
};
