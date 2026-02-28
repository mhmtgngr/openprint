/**
 * DiscoveredPrinters Feature Module
 * Exports components for discovered printer management
 */

// Components
export { DiscoveredPrinterList } from './DiscoveredPrinterList';

// Types are re-exported from agents feature
export type {
  DiscoveredPrinter,
  DiscoveredPrinterCapabilities,
} from '@/types/agents';
