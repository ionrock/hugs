# Hugs - Hugo Blog Editor

A simple web-based editor for Hugo blog posts with Git integration.

## Installation

```bash
go install github.com/ionrock/hugs@latest
```

## Usage

```bash
hugs --content-dir=/path/to/your/hugo/blog
```

### Options

- `--content-dir`: Path to your Hugo blog directory (default: current directory)
- `--port`: Port to run the server on (default: 8080)
- `--debug`: Enable debug logging
- `--hugo-server`: Start the Hugo server alongside the editor

## Systemd Service

A systemd service file is included to run Hugs as a user service on Linux.

1. Copy the service file to your user systemd directory:
   ```bash
   mkdir -p ~/.config/systemd/user/
   cp hugs.service ~/.config/systemd/user/
   ```

2. Edit the service file to set the correct paths:
   ```bash
   nano ~/.config/systemd/user/hugs.service
   ```

3. Enable and start the service:
   ```bash
   systemctl --user enable hugs.service
   systemctl --user start hugs.service
   ```

4. Check the status:
   ```bash
   systemctl --user status hugs.service
   ```

5. View logs:
   ```bash
   journalctl --user -u hugs.service
   ```
