# Upgrading

---

## Upgrading FOG Next

For any patch or minor release:

```bash
git pull
mise run server
sudo install -m 0755 build/fog /usr/local/bin/fog
sudo systemctl restart fog
```

Database migrations run automatically on startup. To apply them manually before restarting:

```bash
sudo fog migrate up
sudo fog migrate status   # confirm the new version
```

To roll back one migration step:

```bash
sudo fog migrate down
```

> **Never** roll back a migration that has already been used in production without first backing up the database.
