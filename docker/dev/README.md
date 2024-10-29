# HARMONY Development Environment Guide

## Getting Started

The development environment runs in Docker containers and features hot-reload capabilities for Go code, and the project's dependencies (e.g. templates, translations, ...) using Air.

### Prerequisites

- Docker and Docker Compose installed (should come with Docker Desktop for Windows pre-installed!)
- Git (for cloning the repository)

### Starting the Development Environment

1. Navigate to the development docker directory:

```bash
cd docker/dev
```

2. Start the environment:

```bash
docker compose up --build
```

The application will be available at `http://localhost:8080`. However, you'll still need DB migrations.

You'll also have to set up your authorization provider of choice, consult the `config/auth.toml` for this. Consider putting your overwrite in `config/local/auth.toml` so that it's not commited to the VCS by accident.

### Database Migrations

Migrations can be controlled via the `RUN_MIGRATIONS` environment variable in `docker-compose.yml`:

```yaml
environment:
  RUN_MIGRATIONS: true  # or false
```

**Important Notes:**

- This setting can only be changed in the `docker-compose.yml` file
- Setting it to `true` will run migrations on container startup
- While migrations are versioned, keeping this enabled could lead to unexpected database changes when pulling new
  migrations from remote
- For controlled database updates, it's recommended to keep this `false` and run migrations manually when needed

### Useful Docker Compose Commands

```bash
# Start in detached mode (run in background)
docker compose up -d

# Force recreation of containers
docker compose up --force-recreate

# Rebuild containers and start
docker compose up --build

# Stop and remove containers
docker compose down

# View logs when running in detached mode
docker compose logs -f

# Common combinations
docker compose up -d --build  # Rebuild and start in background
docker compose up --force-recreate --build  # Full rebuild and restart
```

### Development Workflow

1. The entire project directory is mounted into the container
2. Changes to Go files trigger automatic rebuilds via Air
3. Application logs are visible in the Docker Compose output
4. Database data persists across container restarts via Docker volumes

Remember to rebuild the containers (`--build`) when making changes to the Docker configuration files or when new
dependencies are added to the project.