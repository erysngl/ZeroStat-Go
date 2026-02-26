<div align="right">
  <strong>🇺🇸 English</strong> | <a href="README-TR.md">🇹🇷 Türkçe</a>
</div>

<div align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go" alt="Go Version" />
  <img src="https://img.shields.io/badge/HTMX-1.9.9-336699?style=for-the-badge&logo=htmx" alt="HTMX" />
  <img src="https://img.shields.io/badge/Tailwind_CSS-3.x-38B2AC?style=for-the-badge&logo=tailwind-css" alt="Tailwind CSS" />
  <img src="https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker" alt="Docker Ready" />
</div>

<h1 align="center">ZeroStat-Go</h1>

<p align="center">
  <strong>Ultra-Lightweight System & Network Dashboard</strong>
</p>

## Overview

**ZeroStat-Go** is a high-performance, minimalist server resource dashboard engineered for maximum efficiency. It completely eschews heavy JavaScript frameworks in favor of Go, HTMX, and Tailwind CSS, providing real-time infrastructure visibility with an incredibly tiny footprint.

Monitor your **CPU, Memory, Disk capacity, and active Network I/O (in precise KB/s)** without taxing the hardware you are trying to observe.

## Key Features

- **Blazing Fast Backend:** Powered by a statically compiled Go binary utilizing `gopsutil`.
- **Zero-JS-Framework Frontend:** Binds Go templating directly to **HTMX** for seamless, partial-page updates.
- **Dynamic Theming:** Built-in Light and Dark mode toggles leveraging Tailwind CSS.
- **Secure Access:** Robust session-based authentication guarding your metrics layer.
- **KB/s Network Tracking:** Live Rx/Tx network speed tracking scaled dynamically.
- **Dynamic Configuration:** Adjust listening ports (default **9124**), passwords, and themes post-deployment via an integrated settings panel.
- **i18n Support:** First-class support for English and Turkish locales.
- **Cloud Native:** Arrives with an optimized, multi-stage Alpine Dockerfile clocking in at under `20MB`.

## Smart Automation Engine & Alerting

ZeroStat-Go features a powerful, built-in automation engine that evaluates your system metrics against user-defined thresholds safely in the background. 

- **Advanced Alerting Logic:** Setup CPU, RAM, and Network (KB/s) threshold monitoring with second-based sustain durations (**Duration**) to prevent false positives, alongside spam-prevention timeout windows (**Cooldown**).
- **Multi-Channel Notifications:** Automatically dispatch alerts via the integrated **Telegram Bot**, customizable Webhooks, or SMTP Email.
- **Dynamic Messaging:** Craft smart notification templates using context-aware placeholders like `{hostname}`, `{metric}`, `{value}`, and `{duration}` to provide deep context when a guardrail is breached.
- **Safe Execution:** Automate shell commands (e.g., `docker stop $(docker ps -q)`) safely using a vetted sandbox environment without freezing the main process thread.

## Architecture

* **Language:** Go (Golang)
* **Frontend:** HTMX + HTML/Templates
* **Styling:** Tailwind CSS (via CDN to eliminate `node_modules` overhead)
* **OS Interfacing:** `shirou/gopsutil`
* **State/Routing:** Native `net/http` + `gorilla/sessions`

## Configuration

ZeroStat-Go handles configuration initially via environment variables (falling back to secure defaults), which can later be mutated via the web dashboard.

1. Copy the example configuration:
   ```bash
   cp .env.example .env
   ```
2. Adjust your variables:
   ```ini
   ZEROSTAT_PORT=9124
   ZEROSTAT_PASSWORD=your_secure_password
   SESSION_SECRET=your_32_byte_session_secret
   
   # Notification Settings
   TG_BOT_TOKEN=your_telegram_bot_token
   TG_CHAT_ID=your_telegram_chat_id
   WEBHOOK_URL=https://your-webhook.com/endpoint
   ```

## Installation & Deployment

### Method 1: Docker Deployment (Recommended)

The most robust way to run ZeroStat-Go, mapping host metrics accurately into the container. Save the following content as `docker-compose.yml`:

```yaml
services:
  zerostat:
    image: ghcr.io/erysngl/zerostat-go:latest
    container_name: zerostat-dashboard
    restart: unless-stopped
    ports:
      - "9124:9124"
    environment:
      - ZEROSTAT_PASSWORD=admin
    pid: host
    volumes:
      - /proc:/host/proc:ro
      - /sys:/host/sys:ro
      - /:/host/root:ro
      - ./.env:/app/.env
      - ./data:/app/data
```

Before starting, ensure you create an empty `.env` file and a `data` directory first to prevent Docker from misinterpreting the mounts:
```bash
touch .env
mkdir data
```

Then, launch the system and access your dashboard:

1. Bring up the container in the background:
   ```bash
   docker-compose up -d
   ```
2. Navigate to **http://localhost:9124** in your browser.

### Data Persistence (Kalıcı Veri)

ZeroStat-Go allows you to configure settings like your Port, Admin Password, and Telegram/Webhook credentials dynamically through the web UI's Settings panel, while your active rules are controlled through the Automation panel. 

By mapping the `.env` file (`- ./.env:/app/.env`) and the `data/` directory (`- ./data:/app/data`) as shown in the docker-compose snippet, you enforce **Full Data Persistence**:
1. **Application Settings:** Written instantly to `.env` upon save.
2. **Automation Rules:** Instantly serialized to `data/rules.json` upon adding, deleting, or toggling conditions.

Consequently, if your Docker container is updated, rebuilt, or deleted, **your settings and threshold configurations will not be lost**. They will be safely reloaded on boot.

### Method 2: Native Build

Assuming you have Go `1.21+` installed:

```bash
# Clone the repository
git clone https://github.com/yourusername/zerostat.git
cd zerostat

# Fetch dependencies
go mod tidy

# Build the executable
go build -ldflags="-s -w" -o zerostat ./cmd/zerostat

# Run the daemon (Accessible via http://localhost:9124)
./zerostat
```

## Security

ZeroStat-Go protects the dashboard endpoint behind a highly secure, HTTP-only cookie session mechanism (`SameSite=Lax`). The default password is `admin` (or defined in your `.env`). It is **highly recommended** to change this immediately upon first login via the Settings panel or your `.env` file before exposing port **9124** to the public internet. Furthermore, malformed cookies are handled gracefully by safely clearing sessions rather than throwing errors.

## License

This project is licensed under the MIT License.

---
<p align="center">
  <a href="https://erysngl.github.io">ERYSNGL | Github</a>
</p>
