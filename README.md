# ProxS3

Native S3 storage plugin for Proxmox VE. Provides ISO images, container templates, snippets, and backups backed by any S3-compatible object store.

Unlike FUSE-based approaches (s3fs, rclone mount), ProxS3 integrates directly with Proxmox's storage subsystem. Proxmox understands the storage is remote, handles disconnects gracefully, and shows proper status in the UI.

## Architecture

- **proxs3d** — Go daemon that handles all S3 operations, local caching, and health monitoring. Listens on a Unix socket.
- **S3Plugin.pm** — Perl storage plugin that registers as a native Proxmox storage type (`s3`). Forwards all operations to the daemon.

```
Proxmox UI / pvesm
       |
       v
  S3Plugin.pm  (PVE::Storage::Custom)
       | Unix socket
       v
    proxs3d    (Go daemon)
       | HTTPS         | local disk
       v               v
   S3 bucket       file cache
```

The daemon auto-discovers S3 storages from `/etc/pve/storage.cfg` — no duplicate configuration needed. When you add, update, or remove an S3 storage via the Proxmox UI or `pvesm`, the plugin signals the daemon to reload.

## Supported Content Types

- `iso` — ISO images
- `vztmpl` — Container templates
- `snippets` — Snippets (cloud-init, hookscripts)
- `backup` — Backup files

## Caching

ProxS3 maintains a per-node local file cache for downloaded objects. This is critical for performance since ISOs and templates are large and read-heavy.

- **S3 is always the source of truth.** On every access, the daemon checks the S3 object's ETag/LastModified against the cached copy. If the remote object has changed, the cache entry is invalidated and re-downloaded.
- **Each node has its own cache.** Cache location and size are configurable per-node via `proxs3d.json` — put it on a fast local disk or a dedicated mount, not your rootfs.
- **LRU eviction.** When the cache exceeds `cache_max_mb`, the oldest files are evicted automatically.
- **Upload caching.** Files uploaded via the Proxmox UI are cached locally after upload, so they're immediately available without a round-trip to S3.

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
    "storage_cfg": "/etc/pve/storage.cfg",
    "proxy": {
        "https_proxy": "",
        "http_proxy": "",
        "no_proxy": ""
    }
}
```

**Important:** Set `cache_dir` to a path with enough space — don't leave it on your rootfs if it's small. A dedicated mount or a large local disk is ideal.

If your network requires an HTTP proxy for outbound access, set the `proxy` fields.

Per-storage configuration (endpoint, bucket, region, etc.) is read automatically from `/etc/pve/storage.cfg`. You do **not** need to duplicate storage definitions in this file.

### 2. Start the daemon

```bash
systemctl enable --now proxs3d
```

### 3. Add storage in Proxmox

Via the web UI: **Datacenter -> Storage -> Add -> S3**

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

Credentials are automatically stored in `/etc/pve/priv/proxs3/s3-isos.json` (cluster-shared, root-only) and are **not** written to `storage.cfg`. The daemon is automatically signalled to reload.

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

- **AWS S3** — set `use-ssl 1`, `path-style 0`
- **MinIO** — set `path-style 1`
- **Ceph RGW** — set `path-style 1`
- **Cloudflare R2** — set `path-style 0`
- **Wasabi** — set `path-style 0`

## Multi-Node Clusters

ProxS3 is designed for Proxmox clusters:

- `storage.cfg` is shared across all nodes via pmxcfs — add the storage once, it appears everywhere
- Credentials in `/etc/pve/priv/` are also cluster-shared
- The `proxs3d` daemon and `.deb` package must be installed on each node
- Each node maintains its own local cache (configurable location and size)
- The daemon config `/etc/proxs3/proxs3d.json` is per-node (not in `/etc/pve/`) so you can tune cache paths per node

## Daemon Management

```bash
# Reload config (re-reads storage.cfg, picks up new/changed storages)
systemctl reload proxs3d

# View logs
journalctl -u proxs3d -f

# Restart
systemctl restart proxs3d
```

## Building the .deb

```bash
sudo apt install debhelper golang-go
make deb
```

## License

MIT
