/**
 * Test data fixtures for E2E tests
 */

export const testUsers = {
  admin: {
    email: 'admin@openprint.test',
    password: 'TestAdmin123!',
    name: 'Admin User',
    role: 'admin',
  },
  user: {
    email: 'user@openprint.test',
    password: 'TestUser123!',
    name: 'Regular User',
    role: 'user',
  },
  owner: {
    email: 'owner@openprint.test',
    password: 'TestOwner123!',
    name: 'Owner User',
    role: 'owner',
  },
} as const;

export const testPrinters = {
  hpLaserjet: {
    name: 'HP LaserJet Pro M404n',
    type: 'laser',
    ip: '192.168.1.100',
    port: 9100,
    isOnline: true,
    isActive: true,
  },
  canonImageRunner: {
    name: 'Canon imageRUNNER 1435iF',
    type: 'mfp',
    ip: '192.168.1.101',
    port: 9100,
    isOnline: true,
    isActive: true,
  },
  epsonEcoTank: {
    name: 'Epson EcoTank ET-4760',
    type: 'inkjet',
    ip: '192.168.1.102',
    port: 9100,
    isOnline: false,
    isActive: true,
  },
} as const;

export const testDocuments = {
  pdf: {
    name: 'Test Document.pdf',
    pages: 5,
    size: 102400, // 100KB
    type: 'application/pdf',
  },
  docx: {
    name: 'Meeting Notes.docx',
    pages: 2,
    size: 51200, // 50KB
    type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
  },
} as const;

export const testPolicies = {
  colorRestriction: {
    name: 'Color Printing Restriction',
    description: 'Limit color printing to admin users only',
    priority: 1,
    isEnabled: true,
    conditions: {
      userRole: ['user'],
    },
    actions: {
      forceGrayscale: true,
    },
  },
  duplexDefault: {
    name: 'Duplex Default',
    description: 'Enable double-sided printing by default',
    priority: 2,
    isEnabled: true,
    conditions: {
      always: true,
    },
    actions: {
      forceDuplex: true,
    },
  },
} as const;

export const testQuotas = {
  monthly: {
    name: 'Monthly Page Limit',
    type: 'monthly',
    limit: 1000,
    resetDay: 1,
  },
  weekly: {
    name: 'Weekly Page Limit',
    type: 'weekly',
    limit: 250,
    resetDay: 1, // Monday
  },
} as const;
