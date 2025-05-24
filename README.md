I want to build a database on Dynamo DB that will be a flexible time versioned database kind of similar to datomic or XTDB in its architecture.
The basic building block is a tuple that represents the value of a single field at a given time, something like: (id, timestamp, namespace/fieldName, dataType, value).

I want to build different indexes for common access patterns for looking things up based on field or time, or maybe even on value for certain data types.

The idea is to build a versioned flexible database platform to be at the core of a personal information management system to keep track of your life, or any other thing.
Kind of inspired by notion or by spreadsheets.

This database platform will be at the core of this broader tool and will enable it to scale.

Everything should be partitioned by user, and every user will get their own namespace so their data is all in their own partition and they can make their own field types.

The main UI will be an app / web app that will essentially look like some tables like airtable or something where each data type is rendered in a useful way.

The table abstraction is something that will have to be built after, but it will be based on this fundamental database platform.

## Environment Setup

Notably supports both AWS DynamoDB and a local DynamoDB emulator for development.

### AWS DynamoDB

Ensure you have valid AWS credentials and region configured. For example:

```bash
export AWS_ACCESS_KEY_ID=YOUR_ACCESS_KEY_ID
export AWS_SECRET_ACCESS_KEY=YOUR_SECRET_ACCESS_KEY
export AWS_REGION=us-west-2
```

### Local DynamoDB Emulator

You can run DynamoDB Local via Docker:

```bash
docker run --name dynamodb-local -p 8000:8000 amazon/dynamodb-local
```

Point the SDK to your local emulator by setting:

```bash
export DYNAMODB_ENDPOINT_URL=http://localhost:8000
export AWS_REGION=us-west-2      # region is still required
export AWS_ACCESS_KEY_ID=foo     # use dummy credentials
export AWS_SECRET_ACCESS_KEY=bar

## Testing

Notably uses a DynamoDB emulator for testing instead of mocks.

### Running Tests

To run tests that interact with DynamoDB, start the emulator first:

```bash
docker run --name dynamodb-local -p 8000:8000 amazon/dynamodb-local
```

Then run tests:

```bash
go test ./...
```

Use the `-short` flag to skip tests requiring DynamoDB:

```bash
go test -short ./...
```

See the [Testing Guidelines](backend/TESTING.md) for more details on using the `testutil/dynamotest` package.

## Frontend

The frontend is a React + TypeScript application built with Vite and styled with Mantine UI components.

To run the web frontend, navigate into the `frontend` directory and install dependencies:

```bash
cd frontend
npm install
npm run dev
```

The frontend will be available at http://localhost:5173 and proxies API requests to http://localhost:8080.

### Development Scripts

- `npm run dev` - Start the development server
- `npm run build` - Build for production 
- `npm run preview` - Preview production build
- `npm run lint` - Run ESLint
- `npm run dev:all` - Start both frontend and backend concurrently

### Testing

The frontend uses Playwright for end-to-end testing with comprehensive test coverage:

```bash
# Install Playwright browsers
npm run test:install

# Run all tests
npm run test

# Run tests in headed mode (with browser UI)
npm run test:headed

# Run tests with debug mode
npm run test:debug

# Open Playwright UI for interactive testing
npm run test:ui

# Run specific test suites
npm run test:auth          # Authentication tests
npm run test:tables        # Table operations tests
npm run test:history       # History snapshot tests
npm run test:workflows     # Data workflow tests
npm run test:accessibility # Accessibility tests
```

To set up the backend for local dev:

```bash
DYNAMODB_TABLE_NAME=NotablyTest DYNAMODB_ENDPOINT_URL="http://localhost:8000" go run cmd/create-table/main.go
DYNAMODB_TABLE_NAME=NotablyTest DYNAMODB_ENDPOINT_URL="http://localhost:8000" go run cmd/server/main.go
```
