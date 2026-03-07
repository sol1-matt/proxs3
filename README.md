# ProxS3

Native S3 storage plugin for Proxmox VE. Provides ISO images, container templates, snippets, and backups backed by any S3-compatible object store.

Unlike FUSE-based approaches (s3fs, rclone mount), ProxS3 integrates directly with Proxmox's storage subsystem. Proxmox understands the storage is remote, handles disconnects gracefully, and shows proper status in the UI.

## Architecture

- **proxs3d** — Go daemon that handles all S3 operations, local caching, and health monitoring. Listens on a Unix socket.
- **S3Plugin.pm** — Perl storage plugin that registers as a native Proxmox storage type (`s3`). Forwards all operations to the daemon.

```
Proxmox UI / pvesm
       │
       ▼
  S3Plugin.pm  (PVE::Storage::Custom)
       │ Unix socket
       ▼
    proxs3d    (Go daemon)
       │ HTTPS
       ▼
   S3 bucket   (AWS, MinIO, Ceph RGW, etc.)
```

## Supported Content Types

- `iso` — ISO images
- `vztmpl` — Container templates
- `snippets` — Snippets (cloud-init, hookscripts)
- `backup` — Backup files

## Installation

### From .deb (recommended)

Download the latest `.deb` from [Releases](https://github.com/sol1/proxs3/releases) and install on each Proxmox node:

```bash
dpkg -i proxs3_*.deb
```

### From source

```bash
make build
sudo make install
```

## Configuration

### 1. Configure the daemon

Edit `/etc/proxs3/proxs3d.json`:

```json
{
    "socket_path": "/run/proxs3d.sock",
    "cache_dir": "/var/cache/proxs3",
    "cache_max_mb": 4096,
    "credential_dir": "/etc/pve/priv/proxs3",
    "proxy": {
        "https_proxy": "",
        "http_proxy": "",
        "no_proxy": ""
    },
    "storages": [
        {
            "storage_id": "s3-isos",
            "bucket": "my-proxmox-bucket",
            "endpoint": "s3.amazonaws.com",
            "region": "us-east-1",
            "use_ssl": true,
            "path_style": false
        }
    ]
}
```

If your network requires an HTTP proxy for outbound access, set the `proxy` fields.

### 2. Start the daemon

```bash
systemctl enable --now proxs3d
```

### 3. Add storage in Proxmox

Via the web UI: **Datacenter → Storage → Add → S3**

Or via CLI:

```bash
pvesm add s3 s3-isos \
    --endpoint s3.amazonaws.com \
    --bucket my-proxmox-bucket \
    --region us-east-1 \
    --access-key AKIAIOSFODNN7EXAMPLE \
    --secret-key wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY \
    --content iso,vztmpl,snippets \
    --use-ssl 1
```

Credentials are automatically stored in `/etc/pve/priv/proxs3/s3-isos.json` (cluster-shared, root-only) and are **not** written to `storage.cfg`.

### 4. Bucket layout

ProxS3 expects the following key prefixes in your S3 bucket:

| Content Type | S3 Prefix |
|---|---|
| ISO images | `template/iso/` |
| Container templates | `template/cache/` |
| Snippets | `snippets/` |
| Backups | `dump/` |

## S3-Compatible Stores

ProxS3 works with any S3-compatible object store:

- **AWS S3** — set `use_ssl: true`, `path_style: false`
- **MinIO** — set `path_style: true`
- **Ceph RGW** — set `path_style: true`
- **Cloudflare R2** — set `path_style: false`
- **Wasabi** — set `path_style: false`

## Multi-Node Clusters

ProxS3 is designed for Proxmox clusters:

- `storage.cfg` is shared across all nodes via pmxcfs
- Credentials in `/etc/pve/priv/` are also cluster-shared
- The `proxs3d` daemon and `.deb` package must be installed on each node
- Each node maintains its own local cache in `/var/cache/proxs3/`
- The daemon config `/etc/proxs3/proxs3d.json` is per-node (not in `/etc/pve/`)

## Building the .deb

```bash
sudo apt install debhelper golang-go
make deb
```

## License

MIT
