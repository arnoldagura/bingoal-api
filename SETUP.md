# Bingoals API Setup

## Prerequisites
- Go 1.21+ installed
- (Optional) Docker for containerized deployment

## Quick Start

### 1. Install Dependencies
```bash
cd /Users/arnold/Documents/work/bingoals-api
go mod tidy
```

### 2. Create Environment File
```bash
cp .env.example .env
# Edit .env and set your JWT_SECRET
```

### 3. Run the Server
```bash
make run
# or
go run cmd/api/main.go
```

Server will start at `http://localhost:8080`

## API Endpoints

### Auth (Public)
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/auth/register` | Register new user |
| POST | `/api/auth/login` | Login user |

### User (Protected)
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/me` | Get current user |

### Boards (Protected)
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/boards` | List all boards |
| POST | `/api/boards` | Create board |
| GET | `/api/boards/:id` | Get board |
| PUT | `/api/boards/:id` | Update board |
| DELETE | `/api/boards/:id` | Delete board |

### Goals (Protected)
| Method | Endpoint | Description |
|--------|----------|-------------|
| PUT | `/api/boards/:boardId/goals/:position` | Update goal |
| POST | `/api/boards/:boardId/goals/:position/toggle` | Toggle completion |

## Test the API

### Register
```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123","name":"Test User"}'
```

### Login
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'
```

### Create Board (use token from login)
```bash
curl -X POST http://localhost:8080/api/boards \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{"title":"2026 Goals"}'
```

## Project Structure
```
bingoals-api/
├── cmd/api/main.go          # Entry point
├── internal/
│   ├── config/              # Configuration
│   ├── database/            # Database connection
│   ├── handlers/            # HTTP handlers
│   ├── middleware/          # JWT auth middleware
│   ├── models/              # Data models
│   └── routes/              # Route definitions
├── .env.example
├── Dockerfile
├── Makefile
└── go.mod
```

## Next Steps After Setup
1. [ ] Run `go mod tidy` to install dependencies
2. [ ] Copy `.env.example` to `.env`
3. [ ] Run `make run` to start server
4. [ ] Test with curl or Postman
5. [ ] Connect Flutter app to this API
