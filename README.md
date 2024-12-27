<div align="center">
  <p>qBitTorrent mock api for *Arr's to forward requests to real debrid</p>
</div>

---

## Table of Contents

- [What is qDebrid](#what-is-qdebrid)
- [Getting Started](#getting-started)
  - [Docker Setup](#docker-setup)
  - [Native Setup](#native-setup)
- [Development](#development)
  - [Prerequisites](#prerequisites)
  - [Setup Instructions](#setup-instructions)
  - [Running the Project](#running-the-project)
- [Contributing](#contributing)
- [License](#license)

---

## What is qDebrid

qDebrid creates a 'fake' QBitTorrent server with the endpoints necessary for *Arrs services to add/remove/scan for media, qDebrid will forward these requests to your real debrid account.

## Getting Started

### Docker Setup

The image for this project is available on Docker at `ghcr.io/sushydev/qdebrid:main`. Below is an example of a `docker-compose.yml` file to set up the project:

```yaml
qdebrid:
  container_name: qdebrid
  image: ghcr.io/sushydev/qdebrid:main
  restart: unless-stopped
  network_mode: host  # Preferable if using a specific network
  volumes:
    - ./qdebrid.yml:/app/config.yml  # Bind configuration
    - ./your-mount-directory:/mnt/your-mount-directory  # Make mount available for path checking
    - ./logs/qdebrid:/app/logs  # Store logs
  healthcheck:
    test: ["CMD-SHELL", "curl --fail http://localhost:8080 || exit 1"]
    interval: 1m00s
    timeout: 15s
    retries: 3
    start_period: 1m00s
```

### Native Setup

To build the project manually, you can use the following Go commands:

1. **Download Dependencies:**
    ```sh
    go mod download
    ```

2. **Build the Project:**
    ```sh
    CGO_ENABLED=0 GOOS=linux go build -o main main.go
    ```

### Configuration

qDebrid uses a `config.yml` file (Very important its `yml` and not `yaml`) with the following properties

Example `config.yml` with all its default values. In your own `config.yml` you can leave out all the settings that you leave on default.
```yaml
settings:
  qdebrid:
    host: ""
    port: 8080

    # Category name for *Arrs (Sonarr, Radarr, etc.)
    category_name: "qdebrid"

    # Parent directory where media is saved
    save_path: "/mnt/fvs/debrid_drive/media_manager"

    # Whether to validate paths when updating *Arrs
    validate_paths: true

    # Deprecated: Reject if unavailable
    allow_uncached: false

    # Allowed file types to be considered when adding to RD
    allowed_file_types:
      - "mkv"
      - "mp4"

    # Minimum file size in bytes. (500MB)
    min_file_size: 500000000

    # Log level for the application (e.g., info, debug, warn)
    log_level: "info" 
  
  real_debrid:
    # Your Real Debrid API token
    token: ""  
```

### Setup in Arrs

qDebrid is tested in Sonarr and Radarr but might work for other services too.

In your Arrs of choice:
1. Go to `settings/general` and get your API Key
2. Go to `settings/downloadclients` and select qBitTorrent
3. Set the host and port to what you configured (or default localhost & 8080)
4. Set the "Username" field to the host of your Arrs service (For example http://localhost:8989 for sonarr)
5. Set the "Password" field to the api key of your Arrs service
6. Set the "Category" field to what you configured (or default qdebrid)
7. Click test to check if it works and save!

#### Why does it need my Arrs API key?
Since qDebrid has no clue what entries in your real debrid are related to your Arrs it makes a request to the `/history` endpoint of the arrs and only returns entries that have been requested by that specific arrs service.
This way you only see the correct entries in your "Activity" panel in the arrs rather than every entry in your real debrid account with a bunch of warning messages.

#### Done
Now you're ready to use it
    
---

## Development

### Prerequisites

Ensure you have the following installed on your system:

- **Go** (version 1.23.2 or later)

### Setup Instructions

1. **Install Dependencies:**
    ```sh
    go mod download
    ```

### Running the Project

- **Start:**
    ```sh
    go run main.go
    ```

---

## Contributing

Contributions are welcome! Please follow the guidelines in the [CONTRIBUTING.md](CONTRIBUTING.md) for submitting changes.

---

## License

This project is licensed under the GNU GPLv3 License. See the [LICENSE](LICENSE) file for details.
