# OpenPrint Cloud - Web Dashboard

The web dashboard for OpenPrint Cloud, providing a modern interface for managing cloud-routed printing.

## Tech Stack

- **React 18** - UI library
- **TypeScript** - Type safety
- **Vite** - Build tool and dev server
- **React Router** - Client-side routing
- **TanStack Query** - Server state management
- **Tailwind CSS v4** - Styling
- **Recharts** - Data visualization
- **Zustand** - Client state management
- **Playwright** - E2E testing
- **Vitest** - Unit testing

## Project Structure

```
src/
├── components/          # Shared UI components
│   ├── Layout.tsx      # Main app layout with sidebar
│   ├── icons/          # Icon components
│   ├── JobStatusBadge.tsx
│   ├── PrinterCard.tsx
│   ├── JobList.tsx
│   └── EnvironmentReport.tsx
├── pages/              # Page components
│   ├── Login.tsx
│   ├── Dashboard.tsx
│   ├── Printers.tsx
│   ├── Jobs.tsx
│   ├── Analytics.tsx
│   ├── Settings.tsx
│   └── Organization.tsx
├── hooks/              # Custom React hooks
│   ├── useAuth.ts      # Authentication hook
│   ├── useWebSocket.ts # Real-time updates
│   └── useJobs.ts      # Print job management
├── services/           # API clients
│   ├── api.ts          # REST API
│   └── websocket.ts    # WebSocket client
├── types/              # TypeScript types
│   └── index.ts
├── styles/             # Global styles
├── test/               # Test setup
├── App.tsx             # Root component with routing
├── main.tsx            # Entry point
└── index.css           # Global styles
```

## Getting Started

### Installation

```bash
npm install
```

### Development

```bash
npm run dev
```

The app will be available at `http://localhost:3000`.

### Build

```bash
npm run build
```

### Preview Production Build

```bash
npm run preview
```

## Testing

### Unit Tests

```bash
npm run test
```

### E2E Tests

```bash
npm run test:e2e
```

Run E2E tests with UI:

```bash
npm run test:e2e:ui
```

## Features

### Pages

- **Login** - Authentication with SSO options
- **Dashboard** - Overview of print jobs, printers, and environmental impact
- **Printers** - Manage organization printers
- **Jobs** - View and manage print job history
- **Analytics** - Usage statistics and reports (Admin only)
- **Settings** - User profile and preferences
- **Organization** - Team and printer management (Admin only)

### Components

- `JobStatusBadge` - Status indicator for print jobs
- `PrinterCard` - Printer information card
- `JobList` - List of print jobs with real-time updates
- `EnvironmentReport` - Environmental impact dashboard
- `Layout` - App shell with navigation sidebar

### Hooks

- `useAuth` - Authentication state and actions
- `useWebSocket` - Real-time connection management
- `useJobs` - Print job queries and mutations
- `useJobUpdates` - Real-time job status updates
- `usePrinterUpdates` - Real-time printer status updates

## API Integration

The dashboard connects to the OpenPrint backend API:

- Base URL: `http://localhost:8080` (configurable via `VITE_API_URL`)
- WebSocket: `ws://localhost:8080/ws` (configurable via `VITE_WS_URL`)

## Authentication

The app uses JWT-based authentication:

- Access tokens stored in memory
- Refresh tokens stored in localStorage
- Automatic token refresh on 401 responses
- Redirects unauthenticated users to login

## Environment Variables

```bash
VITE_API_URL=http://localhost:8080/api/v1
VITE_WS_URL=ws://localhost:8080/ws
```

## Dark Mode

The app supports dark mode via system preference. Toggle controls are available in Settings.

## Browser Support

- Chrome/Edge 90+
- Firefox 88+
- Safari 14+
