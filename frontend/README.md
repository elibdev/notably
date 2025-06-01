# Notably Frontend

A modern, responsive web application for managing dynamic data tables with real-time collaboration and history tracking.

## Overview

Notably is a data management platform that allows users to create custom tables with flexible field types, manage entities, and track changes over time. The frontend is built with React, TypeScript, Vite, and Mantine UI components.

## Features

- **User Authentication**: Secure login and registration system
- **Dynamic Tables**: Create tables with custom fields (string, number, boolean, date, etc.)
- **Entity Management**: Full CRUD operations for table entities
- **History Tracking**: View changes and modifications over time
- **Responsive Design**: Works seamlessly on desktop and mobile devices
- **Real-time Updates**: Live data synchronization with the backend
- **Soft Deletes**: Restore accidentally deleted entities

## Tech Stack

- **Framework**: React 19 with TypeScript
- **Build Tool**: Vite 6
- **UI Library**: Mantine v8
- **Routing**: React Router v7
- **Form Handling**: Mantine Form with validation
- **Icons**: Tabler Icons
- **Date Handling**: Mantine Dates
- **Testing**: Vitest with React Testing Library
- **State Management**: React Context API

## Prerequisites

- Node.js 18+ 
- npm or yarn
- Backend API server running on `http://localhost:8080`

## Installation

1. **Clone the repository** (if not already done):
   ```bash
   git clone <repository-url>
   cd notably/frontend
   ```

2. **Install dependencies**:
   ```bash
   npm install
   ```

3. **Set up environment variables**:
   ```bash
   cp .env.example .env
   ```
   
   Edit `.env` to configure your API endpoint:
   ```env
   VITE_API_BASE_URL=http://localhost:8080
   ```

## Development

### Running the Development Server

```bash
npm run dev
```

The application will be available at `http://localhost:5173`

### Building for Production

```bash
npm run build
```

Built files will be in the `dist/` directory.

### Preview Production Build

```bash
npm run preview
```

## Testing

### Run All Tests

```bash
npm run test
```

### Run Tests in Watch Mode

```bash
npm run test:watch
```

### Run Tests Once (CI Mode)

```bash
npm run test:run
```

### Test Coverage

```bash
npm run test -- --coverage
```

## Project Structure

```
src/
├── components/          # Reusable UI components
├── contexts/           # React Context providers
│   └── AuthContext.tsx # Authentication state management
├── pages/              # Main application pages
│   ├── LoginPage.tsx   # User login
│   ├── RegisterPage.tsx # User registration
│   ├── DashboardPage.tsx # Tables overview
│   └── TableDetailPage.tsx # Table entity management
├── services/           # API service layer
│   └── api.ts          # HTTP client and API methods
├── types/              # TypeScript type definitions
│   └── api.ts          # API response/request types
├── test/               # Test utilities and setup
│   └── setup.ts        # Vitest configuration
├── App.tsx             # Main app component with routing
└── main.tsx            # Application entry point
```

## API Integration

The frontend communicates with a REST API backend. Key endpoints include:

- `POST /auth/login` - User authentication
- `POST /auth/register` - User registration
- `GET /tables` - List user's tables
- `POST /tables` - Create new table
- `GET /tables/{id}/entities` - List table entities
- `POST /tables/{id}/entities` - Create new entity

Full API documentation is available in the backend swagger specification.

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `VITE_API_BASE_URL` | Backend API base URL | `http://localhost:8080` |
| `VITE_APP_NAME` | Application name | `Notably` |
| `VITE_APP_VERSION` | Application version | `1.0.0` |
| `VITE_DEV_TOOLS` | Enable dev tools | `true` |

## Key Components

### Authentication Context
Manages user authentication state and provides login/logout functionality throughout the app.

### Dashboard Page
Displays all user tables with the ability to create new tables and manage existing ones.

### Table Detail Page
Shows table entities with full CRUD operations, filtering, and pagination.

### API Service
Centralized HTTP client with error handling, token management, and request/response transformation.

## Development Workflow

1. **Feature Development**:
   - Create feature branch from main
   - Write tests for new functionality
   - Implement the feature
   - Ensure all tests pass
   - Submit pull request

2. **Code Quality**:
   - TypeScript strict mode enabled
   - ESLint for code linting
   - Prettier for code formatting (recommended)
   - Comprehensive test coverage

3. **Testing Strategy**:
   - Unit tests for utilities and services
   - Component tests for UI interactions
   - Integration tests for complex workflows
   - Mock API responses for reliable testing

## Browser Support

- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

## Deployment

The application can be deployed to any static hosting service:

1. **Build the application**:
   ```bash
   npm run build
   ```

2. **Deploy the `dist/` folder** to your hosting provider

Popular options include:
- Vercel
- Netlify
- AWS S3 + CloudFront
- GitHub Pages

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

## Troubleshooting

### Common Issues

**404 errors when calling API**:
- Ensure the backend server is running
- Check the `VITE_API_BASE_URL` environment variable
- Verify API endpoints in the swagger documentation

**Build failures**:
- Clear node_modules and reinstall: `rm -rf node_modules package-lock.json && npm install`
- Check TypeScript errors: `npm run build`

**Test failures**:
- Update test snapshots if UI changes: `npm run test -- --update-snapshots`
- Check mock implementations for API changes

## License

[Add your license information here]