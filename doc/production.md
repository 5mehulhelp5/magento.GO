# Production & Daemon

## Background Process

```bash
nohup go run magento.go > output.log 2>&1 &
```

Or with binary:
```bash
go build -o magento
nohup ./magento > output.log 2>&1 &
```

## systemd Service

Create `/etc/systemd/system/magento.service`:

```ini
[Unit]
Description=Magento Go API

[Service]
ExecStart=/path/to/magento
WorkingDirectory=/path/to/gogento-catalog
Restart=always
User=youruser
Environment=PORT=8080

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl start magento
sudo systemctl enable magento
```
