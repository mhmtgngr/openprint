/**
 * Test Data Factory
 * Generates realistic test data for E2E tests
 */

// Random data generators
const random = {
  integer: (min: number, max: number): number => {
    return Math.floor(Math.random() * (max - min + 1)) + min;
  },

  float: (min: number, max: number, decimals: number = 2): number => {
    return parseFloat((Math.random() * (max - min) + min).toFixed(decimals));
  },

  string: (length: number): string => {
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    let result = '';
    for (let i = 0; i < length; i++) {
      result += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    return result;
  },

  email: (): string => {
    const domains = ['example.com', 'test.com', 'demo.org', 'sample.net'];
    const users = ['john.doe', 'jane.smith', 'bob.wilson', 'alice.johnson', 'test.user'];
    return `${users[Math.floor(Math.random() * users.length)]}${random.integer(1, 999)}@${domains[Math.floor(Math.random() * domains.length)]}`;
  },

  date: (daysAgo: number = 0): string => {
    const date = new Date();
    date.setDate(date.getDate() - daysAgo);
    return date.toISOString();
  },

  boolean: (): boolean => {
    return Math.random() < 0.5;
  },

  // Pick random item from array
  pick: <T>(array: T[]): T => {
    return array[Math.floor(Math.random() * array.length)];
  },

  // Pick multiple unique items from array
  pickMultiple: <T>(array: T[], count: number): T[] => {
    const shuffled = [...array].sort(() => Math.random() - 0.5);
    return shuffled.slice(0, Math.min(count, array.length));
  },
};

// User data factory
export class UserFactory {
  static readonly firstNames = ['John', 'Jane', 'Bob', 'Alice', 'Charlie', 'Diana', 'Edward', 'Fiona', 'George', 'Helen'];
  static readonly lastNames = ['Smith', 'Johnson', 'Williams', 'Brown', 'Jones', 'Garcia', 'Miller', 'Davis', 'Rodriguez', 'Martinez'];
  static readonly roles = ['user', 'admin', 'manager', 'operator'] as const;
  static readonly departments = ['Sales', 'Marketing', 'Engineering', 'HR', 'Finance', 'Operations'];

  static create(overrides: Partial<User> = {}): User {
    const firstName = random.pick(this.firstNames);
    const lastName = random.pick(this.lastNames);

    return {
      id: `user-${random.string(8)}`,
      email: `${firstName.toLowerCase()}.${lastName.toLowerCase()}@example.com`,
      name: `${firstName} ${lastName}`,
      role: random.pick(this.roles),
      department: random.pick(this.departments),
      isActive: true,
      emailVerified: true,
      createdAt: random.date(random.integer(0, 365)),
      ...overrides,
    };
  }

  static createMany(count: number, overrides: Partial<User> = {}): User[] {
    return Array.from({ length: count }, () => this.create(overrides));
  }

  static createAdmin(): User {
    return this.create({
      role: 'admin',
      email: 'admin@example.com',
      name: 'Admin User',
    });
  }
}

// Printer data factory
export class PrinterFactory {
  static readonly manufacturers = ['HP', 'Canon', 'Epson', 'Brother', 'Xerox', 'Kyocera', 'Ricoh'];
  static readonly models = ['LaserJet', 'PIXMA', 'WorkForce', 'HL', 'WorkCentre', 'ECOSYS', 'PriPort'];
  static readonly types = ['network', 'usb', 'wireless', 'shared'] as const;
  static readonly statuses = ['online', 'offline', 'error'] as const;

  static create(overrides: Partial<Printer> = {}): Printer {
    const manufacturer = random.pick(this.manufacturers);
    const model = random.pick(this.models);
    const name = `${manufacturer} ${model} ${random.integer(100, 9999)}`;

    return {
      id: `printer-${random.string(8)}`,
      name,
      manufacturer,
      model,
      type: random.pick(this.types),
      status: random.pick(this.statuses),
      isActive: random.boolean(),
      isOnline: random.boolean() && random.boolean(), // More likely to be offline if not active
      ipAddress: `192.168.1.${random.integer(1, 254)}`,
      location: `Office ${random.integer(1, 5)} - Floor ${random.integer(1, 3)}`,
      capabilities: {
        supportsColor: random.boolean(),
        supportsDuplex: random.boolean(),
        supportedPaperSizes: random.pickMultiple(['A4', 'Letter', 'A3', 'Legal', '11x17'], random.integer(2, 4)),
        resolution: `${random.integer(600, 4800)}x${random.integer(600, 1200)} dpi`,
        maxSheetCount: random.integer(100, 500),
      },
      agentId: `agent-${random.string(8)}`,
      orgId: 'org-1',
      createdAt: random.date(random.integer(0, 180)),
      lastSeen: random.date(random.integer(0, 1)),
      ...overrides,
    };
  }

  static createMany(count: number, overrides: Partial<Printer> = {}): Printer[] {
    return Array.from({ length: count }, () => this.create(overrides));
  }

  static createOnline(overrides: Partial<Printer> = {}): Printer {
    return this.create({ status: 'online', isOnline: true, isActive: true, ...overrides });
  }

  static createOffline(overrides: Partial<Printer> = {}): Printer {
    return this.create({ status: 'offline', isOnline: false, ...overrides });
  }
}

// Job data factory
export class JobFactory {
  static readonly documentTypes = [
    'application/pdf',
    'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    'application/vnd.openxmlformats-officedocument.presentationml.presentation',
    'image/jpeg',
    'image/png',
  ] as const;

  static readonly documentNames = [
    'Quarterly Report.pdf',
    'Sales Presentation.pptx',
    'Invoice Template.docx',
    'Budget Spreadsheet.xlsx',
    'Marketing Materials.pdf',
    'Contract Draft.pdf',
    'Meeting Notes.docx',
    'Project Plan.xlsx',
  ];

  static readonly statuses = ['queued', 'processing', 'completed', 'failed', 'cancelled'] as const;

  static create(overrides: Partial<Job> = {}): Job {
    const pageCount = random.integer(1, 100);
    const colorPages = random.integer(0, pageCount);
    const status = random.pick(this.statuses);

    return {
      id: `job-${random.string(8)}`,
      userId: `user-${random.string(8)}`,
      printerId: `printer-${random.string(8)}`,
      orgId: 'org-1',
      status,
      documentName: random.pick(this.documentNames),
      documentType: random.pick(this.documentTypes),
      pageCount,
      colorPages,
      fileSize: random.integer(102400, 10485760), // 100KB to 10MB
      settings: {
        color: colorPages > 0,
        duplex: random.boolean(),
        paperSize: random.pick(['A4', 'Letter', 'A3']),
        copies: random.integer(1, 10),
      },
      createdAt: random.date(random.integer(0, 30)),
      completedAt: ['completed', 'failed', 'cancelled'].includes(status) ? random.date(random.integer(0, 1)) : undefined,
      errorMessage: status === 'failed' ? 'Printer offline' : undefined,
      estimatedCost: random.float(0.01, 5.00),
      printer: PrinterFactory.create(),
      ...overrides,
    };
  }

  static createMany(count: number, overrides: Partial<Job> = {}): Job[] {
    return Array.from({ length: count }, () => this.create(overrides));
  }

  static createCompleted(overrides: Partial<Job> = {}): Job {
    return this.create({ status: 'completed', completedAt: random.date(0), ...overrides });
  }

  static createFailed(overrides: Partial<Job> = {}): Job {
    return this.create({ status: 'failed', completedAt: random.date(0), errorMessage: 'Printer offline', ...overrides });
  }

  static createProcessing(overrides: Partial<Job> = {}): Job {
    return this.create({ status: 'processing', ...overrides });
  }
}

// Agent data factory
export class AgentFactory {
  static readonly platforms = ['windows', 'macos', 'linux'] as const;
  static readonly statuses = ['online', 'offline', 'error'] as const;

  static create(overrides: Partial<Agent> = {}): Agent {
    const platform = random.pick(this.platforms);

    return {
      id: `agent-${random.string(8)}`,
      name: `${platform.toUpperCase()}-WORKSTATION-${random.integer(100, 999)}`,
      platform,
      platformVersion: this.getPlatformVersion(platform),
      status: random.pick(this.statuses),
      ipAddress: `192.168.1.${random.integer(1, 254)}`,
      agentVersion: `1.${random.integer(0, 9)}.${random.integer(0, 99)}`,
      lastHeartbeat: random.date(random.integer(0, 1)),
      printerCount: random.integer(0, 5),
      jobQueueDepth: random.integer(0, 10),
      sessionState: random.pick(['active', 'idle', 'disconnected']),
      capabilities: {
        supportedFormats: random.pickMultiple(['PDF', 'DOCX', 'XLSX', 'PPTX', 'JPG', 'PNG'], random.integer(3, 6)),
        maxJobSize: random.integer(52428800, 104857600),
        supportsColor: random.boolean(),
        supportsDuplex: random.boolean(),
      },
      orgId: 'org-1',
      createdAt: random.date(random.integer(0, 365)),
      associatedUser: UserFactory.create(),
      ...overrides,
    };
  }

  static createMany(count: number, overrides: Partial<Agent> = {}): Agent[] {
    return Array.from({ length: count }, () => this.create(overrides));
  }

  private static getPlatformVersion(platform: string): string {
    const versions: Record<string, string[]> = {
      windows: ['Windows 11 Pro', 'Windows 10 Pro', 'Windows 11 Enterprise', 'Windows 10 Enterprise'],
      macos: ['macOS Sonoma 14.2', 'macOS Ventura 13.6', 'macOS Monterey 12.7'],
      linux: ['Ubuntu 22.04 LTS', 'Ubuntu 23.10', 'Fedora 39', 'Debian 12'],
    };
    return random.pick(versions[platform] || versions.linux);
  }
}

// Organization data factory
export class OrganizationFactory {
  static readonly plans = ['free', 'pro', 'enterprise'] as const;

  static create(overrides: Partial<Organization> = {}): Organization {
    const plan = random.pick(this.plans);
    const limits = this.getPlanLimits(plan);

    return {
      id: `org-${random.string(8)}`,
      name: `${random.pick(['Acme', 'Tech', 'Global', 'Premier', 'Advanced'])} ${random.pick(['Corp', 'Inc', 'LLC', 'Ltd', 'Solutions'])}`,
      slug: `${random.string(8).toLowerCase()}`,
      plan,
      maxUsers: limits.maxUsers,
      maxPrinters: limits.maxPrinters,
      settings: {
        allowUserRegistration: random.boolean(),
        requireApproval: plan === 'enterprise',
        defaultQuota: limits.defaultQuota,
      },
      createdAt: random.date(random.integer(30, 365)),
      updatedAt: random.date(random.integer(0, 30)),
      ...overrides,
    };
  }

  private static getPlanLimits(plan: string) {
    const limits = {
      free: { maxUsers: 5, maxPrinters: 3, defaultQuota: 100 },
      pro: { maxUsers: 50, maxPrinters: 20, defaultQuota: 500 },
      enterprise: { maxUsers: -1, maxPrinters: -1, defaultQuota: 1000 },
    };
    return limits[plan] || limits.free;
  }
}

// Policy data factory
export class PolicyFactory {
  static create(overrides: Partial<Policy> = {}): Policy {
    const conditions = random.integer(1, 3);
    const actions = random.integer(1, 2);

    return {
      id: `policy-${random.string(8)}`,
      name: `${random.pick(['Limit', 'Require', 'Restrict', 'Allow'])} ${random.pick(['Color', 'Duplex', 'Large Jobs', 'After Hours'])} ${random.pick(['Printing', 'Access', 'Usage'])}`,
      description: `Policy ${random.string(20)} for compliance`,
      enabled: random.boolean(),
      priority: random.integer(1, 100),
      conditions: Array.from({ length: conditions }, () => ({
        type: random.pick(['user', 'group', 'printer', 'document', 'time'] as const),
        operator: random.pick(['equals', 'contains', 'greater_than', 'less_than'] as const),
        value: random.string(10),
      })),
      actions: Array.from({ length: actions }, () => ({
        type: random.pick(['allow', 'deny', 'modify', 'require_approval'] as const),
        parameter: random.boolean() ? { key: random.string(5) } : undefined,
      })),
      scope: {
        users: random.boolean() ? random.pickMultiple([`user-${random.string(8)}` for (let i = 0; i < 10)], random.integer(1, 5)) : undefined,
        groups: random.boolean() ? [`group-${random.string(8)}`] : undefined,
        printers: random.boolean() ? [`printer-${random.string(8)}`] : undefined,
      },
      createdAt: random.date(random.integer(0, 180)),
      updatedAt: random.date(random.integer(0, 30)),
      ...overrides,
    };
  }

  static createMany(count: number, overrides: Partial<Policy> = {}): Policy[] {
    return Array.from({ length: count }, () => this.create(overrides));
  }
}

// Audit log data factory
export class AuditLogFactory {
  static readonly actions = [
    'user.created',
    'user.updated',
    'user.deleted',
    'user.role_changed',
    'job.created',
    'job.completed',
    'job.failed',
    'job.cancelled',
    'printer.created',
    'printer.updated',
    'printer.deleted',
    'policy.created',
    'policy.updated',
    'policy.deleted',
    'login.success',
    'login.failed',
    'settings.updated',
  ] as const;

  static create(overrides: Partial<AuditLog> = {}): AuditLog {
    const action = random.pick(this.actions);

    return {
      id: `audit-${random.string(8)}`,
      userId: `user-${random.string(8)}`,
      orgId: 'org-1',
      action,
      resourceType: this.getResourceType(action),
      resourceId: `${this.getResourceType(action)}-${random.string(8)}`,
      details: random.boolean() ? { key: random.string(5), value: random.string(10) } : undefined,
      ipAddress: `192.168.1.${random.integer(1, 254)}`,
      userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
      timestamp: random.date(random.integer(0, 30)),
      ...overrides,
    };
  }

  static createMany(count: number, overrides: Partial<AuditLog> = {}): AuditLog[] {
    return Array.from({ length: count }, () => this.create(overrides));
  }

  private static getResourceType(action: string): string {
    const resourceMap: Record<string, string> = {
      'user.created': 'user',
      'user.updated': 'user',
      'user.deleted': 'user',
      'user.role_changed': 'user',
      'job.created': 'job',
      'job.completed': 'job',
      'job.failed': 'job',
      'job.cancelled': 'job',
      'printer.created': 'printer',
      'printer.updated': 'printer',
      'printer.deleted': 'printer',
      'policy.created': 'policy',
      'policy.updated': 'policy',
      'policy.deleted': 'policy',
      'login.success': 'session',
      'login.failed': 'session',
      'settings.updated': 'settings',
    };
    return resourceMap[action] || 'unknown';
  }
}

// Environment report data factory
export class EnvironmentReportFactory {
  static create(overrides: Partial<EnvironmentReport> = {}): EnvironmentReport {
    const pagesPrinted = random.integer(1000, 50000);
    const co2Perpage = 0.2; // kg CO2 per page
    const treesPerpage = 0.0001; // trees per page

    return {
      pagesPrinted,
      co2Grams: Math.round(pagesPrinted * co2Perpage * 1000),
      treesSaved: parseFloat((pagesPrinted * treesPerpage).toFixed(2)),
      period: random.pick(['7d', '30d', '90d', '365d']),
      comparisonWithPrevious: random.float(-20, 20),
      ...overrides,
    };
  }
}

// Type definitions
export interface User {
  id: string;
  email: string;
  name: string;
  role: 'user' | 'admin' | 'manager' | 'operator';
  department?: string;
  isActive: boolean;
  emailVerified: boolean;
  createdAt: string;
  avatar?: string;
  phone?: string;
}

export interface Printer {
  id: string;
  name: string;
  manufacturer: string;
  model: string;
  type: 'network' | 'usb' | 'wireless' | 'shared';
  status: 'online' | 'offline' | 'error';
  isActive: boolean;
  isOnline: boolean;
  ipAddress: string;
  location: string;
  capabilities: {
    supportsColor: boolean;
    supportsDuplex: boolean;
    supportedPaperSizes: string[];
    resolution: string;
    maxSheetCount: number;
  };
  agentId: string;
  orgId: string;
  createdAt: string;
  lastSeen: string;
}

export interface Job {
  id: string;
  userId: string;
  printerId: string;
  orgId: string;
  status: 'queued' | 'processing' | 'completed' | 'failed' | 'cancelled';
  documentName: string;
  documentType: string;
  pageCount: number;
  colorPages: number;
  fileSize: number;
  settings: {
    color: boolean;
    duplex: boolean;
    paperSize: string;
    copies: number;
  };
  createdAt: string;
  completedAt?: string;
  errorMessage?: string;
  estimatedCost?: number;
  printer?: Printer;
}

export interface Agent {
  id: string;
  name: string;
  platform: 'windows' | 'macos' | 'linux';
  platformVersion: string;
  status: 'online' | 'offline' | 'error';
  ipAddress: string;
  agentVersion: string;
  lastHeartbeat: string;
  printerCount: number;
  jobQueueDepth: number;
  sessionState: 'active' | 'idle' | 'disconnected';
  capabilities: {
    supportedFormats: string[];
    maxJobSize: number;
    supportsColor: boolean;
    supportsDuplex: boolean;
  };
  orgId: string;
  createdAt: string;
  associatedUser?: User;
}

export interface Organization {
  id: string;
  name: string;
  slug: string;
  plan: 'free' | 'pro' | 'enterprise';
  maxUsers: number;
  maxPrinters: number;
  settings: {
    allowUserRegistration: boolean;
    requireApproval: boolean;
    defaultQuota: number;
  };
  createdAt: string;
  updatedAt: string;
}

export interface Policy {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
  priority: number;
  conditions: Array<{
    type: 'user' | 'group' | 'printer' | 'document' | 'time';
    operator: 'equals' | 'contains' | 'greater_than' | 'less_than';
    value: string | number;
  }>;
  actions: Array<{
    type: 'allow' | 'deny' | 'modify' | 'require_approval';
    parameter?: Record<string, unknown>;
  }>;
  scope: {
    users?: string[];
    groups?: string[];
    printers?: string[];
  };
  createdAt: string;
  updatedAt: string;
}

export interface AuditLog {
  id: string;
  userId: string;
  orgId: string;
  action: string;
  resourceType: string;
  resourceId: string;
  details?: Record<string, unknown>;
  ipAddress: string;
  userAgent: string;
  timestamp: string;
}

export interface EnvironmentReport {
  pagesPrinted: number;
  co2Grams: number;
  treesSaved: number;
  period: string;
  comparisonWithPrevious?: number;
}

// Export all factories
export const TestDataFactory = {
  User: UserFactory,
  Printer: PrinterFactory,
  Job: JobFactory,
  Agent: AgentFactory,
  Organization: OrganizationFactory,
  Policy: PolicyFactory,
  AuditLog: AuditLogFactory,
  EnvironmentReport: EnvironmentReportFactory,
  random,
};

export default TestDataFactory;
