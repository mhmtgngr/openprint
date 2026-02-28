/**
 * Devices feature - Device/printer management components
 */

// Components
export { Devices } from './Devices';
export { DeviceCard } from './DeviceCard';
export { DeviceRegister } from './DeviceRegister';
export { DeviceDetail } from './DeviceDetail';

// API
export { devicesApi, isAgent, isPrinter } from './api';

// Types
export type {
  Device,
  DeviceStatus,
  DeviceAgent,
  DevicePrinter,
  DeviceListParams,
  RegisterPrinterFormData,
  RegisterAgentFormData,
  DeviceStats,
  DeviceAction,
  DeviceStatusConfig,
  RegisterPrinterFormErrors,
  RegisterAgentFormErrors,
} from './types';

// Constants
export { DEVICE_STATUS_CONFIG, HEARTBEAT_INTERVALS } from './types';
