# Volcengine SSH IP Updater

## Project Overview
This project is a Python-based utility designed to automatically update the SSH ingress rule (port 22) of a specific **Volcengine (火山引擎) Security Group**. It monitors the public IP address of the machine where it is running and, upon detecting a change, updates the security group to allow SSH access only from the new IP.

This project now includes a **Web Interface** for easy configuration and monitoring.

## Key Files

*   **`app.py`**: The Flask web application entry point.
*   **`update_ssh_ip.py`**: The core logic for IP detection and Volcengine API interaction.
*   **`models.py`**: SQLite database models for storing settings and logs.
*   **`templates/`**: HTML templates for the web UI.
*   **`run.sh`**: Helper script to set up the environment and start the web server.

## Quick Start (Web Interface)

The easiest way to use this tool is via the web interface.

1.  **Run the start script:**
    ```bash
    ./run.sh
    ```
    This will automatically create a python virtual environment, install dependencies, and start the server.

2.  **Open your browser:**
    Navigate to `http://localhost:5000`.

3.  **Configure:**
    Go to the **Settings** page and enter your:
    *   Volcengine Access Key & Secret Key
    *   Region (e.g., `cn-beijing`)
    *   Security Group ID
    *   SSH Port (default 22)

4.  **Monitor:**
    The **Dashboard** shows the current status, next scheduled run, and recent logs. The system will automatically check for IP changes every 15 minutes (configurable).

## Command Line Usage (Legacy)

You can still run the core script independently if you prefer not to use the web UI, but you must modify the `main()` function in `update_ssh_ip.py` or set up the environment variables (though the refactor prioritizes the class-based approach now).

## Installation Details

If you want to install manually without `run.sh`:

```bash
# Create virtual environment
python3 -m venv venv
source venv/bin/activate

# Install dependencies
pip install -r requirements.txt

# Run the app
python app.py
```

## Running as a Service (Linux)

**Option 1: Web Interface Service (Recommended)**
To run the web application (and the background updater) as a systemd service:

1.  Edit `volcengine-web.service`:
    *   Update `User` to your username.
    *   Update `WorkingDirectory` and `ExecStart` paths to match your installation.
2.  Install and start:
    ```bash
    sudo cp volcengine-web.service /etc/systemd/system/
    sudo systemctl daemon-reload
    sudo systemctl enable volcengine-web
    sudo systemctl start volcengine-web
    ```
    Access the UI at `http://your-server-ip:5000`.

**Option 2: Headless Script Service**
(See `volcengine-ssh-updater.service` for the legacy headless-only mode).

## Security Note

*   **Credentials:** Your Access Key and Secret Key are stored in a local SQLite database (`config.db`). Ensure this file is not shared or committed to public repositories.
*   **Web Access:** The development server (`app.run`) is set to `0.0.0.0` to allow access from other machines on your local network. In a production environment exposed to the internet, use a production WSGI server (like Gunicorn) and a reverse proxy (Nginx) with authentication.