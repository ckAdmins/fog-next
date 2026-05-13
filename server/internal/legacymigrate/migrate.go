// Package legacymigrate migrates data from a FOG 1.x MySQL database into the
// new PostgreSQL schema using direct Ent ORM calls.
package legacymigrate

import (
"context"
"database/sql"
"fmt"
"log/slog"
"time"

_ "github.com/go-sql-driver/mysql" // MySQL driver
"github.com/google/uuid"
"github.com/nemvince/fog-next/ent"
entuser "github.com/nemvince/fog-next/ent/user"
)

// Config holds the MySQL DSN for the legacy FOG database.
type Config struct {
DSN string // e.g. "fog:secret@tcp(localhost:3306)/fog?parseTime=true"
}

// Runner executes the legacy migration against the provided Ent client.
type Runner struct {
legacy *sql.DB
db     *ent.Client
}

// New opens the legacy MySQL connection and returns a Runner.
func New(cfg Config, db *ent.Client) (*Runner, error) {
legacy, err := sql.Open("mysql", cfg.DSN)
if err != nil {
return nil, fmt.Errorf("open legacy db: %w", err)
}
if err := legacy.Ping(); err != nil {
return nil, fmt.Errorf("ping legacy db: %w", err)
}
return &Runner{legacy: legacy, db: db}, nil
}

// Close releases the MySQL connection.
func (r *Runner) Close() error { return r.legacy.Close() }

// Run migrates all entities in dependency order.
func (r *Runner) Run(ctx context.Context) (*Report, error) {
rep := &Report{}

if err := r.migrateHosts(ctx, rep); err != nil {
return rep, fmt.Errorf("hosts: %w", err)
}
if err := r.migrateImages(ctx, rep); err != nil {
return rep, fmt.Errorf("images: %w", err)
}
if err := r.migrateGroups(ctx, rep); err != nil {
return rep, fmt.Errorf("groups: %w", err)
}
if err := r.migrateSnapins(ctx, rep); err != nil {
return rep, fmt.Errorf("snapins: %w", err)
}
if err := r.migrateUsers(ctx, rep); err != nil {
return rep, fmt.Errorf("users: %w", err)
}

return rep, nil
}

// ── Hosts ─────────────────────────────────────────────────────────────────

func (r *Runner) migrateHosts(ctx context.Context, rep *Report) error {
rows, err := r.legacy.QueryContext(ctx, `
SELECT h.hostName, h.hostDesc, h.hostIP,
       h.hostKernel, h.hostKernelArgs,
       h.hostBuilding, h.hostCreateDate
  FROM hosts h
 ORDER BY h.hostID`)
if err != nil {
return err
}
defer rows.Close()

for rows.Next() {
var name, desc, ip, kernel, kernelArgs, building string
var createdAt time.Time
_ = building
if err := rows.Scan(&name, &desc, &ip, &kernel, &kernelArgs, &building, &createdAt); err != nil {
rep.Errors = append(rep.Errors, fmt.Sprintf("scan host: %v", err))
continue
}

h, err := r.db.Host.Create().
SetName(name).
SetDescription(desc).
SetIP(ip).
SetKernel(kernel).
SetKernelArgs(kernelArgs).
SetIsEnabled(true).
Save(ctx)
if err != nil {
slog.Warn("skip host", "name", name, "error", err)
rep.Skipped++
continue
}
rep.Hosts++

if err := r.migrateHostMACs(ctx, name, h.ID); err != nil {
rep.Errors = append(rep.Errors, fmt.Sprintf("macs for host %s: %v", name, err))
}
}
return rows.Err()
}

func (r *Runner) migrateHostMACs(ctx context.Context, hostName string, hostID uuid.UUID) error {
rows, err := r.legacy.QueryContext(ctx, `
SELECT m.mac, m.hostID = h.hostID AS isPrimary
  FROM nics m
  JOIN hosts h ON h.hostName = ?
 WHERE m.hostID = h.hostID`, hostName)
if err != nil {
return err
}
defer rows.Close()

for rows.Next() {
var mac string
var isPrimary bool
if err := rows.Scan(&mac, &isPrimary); err != nil {
continue
}
if err := r.db.HostMAC.Create().
SetHostID(hostID).
SetMAC(mac).
SetIsPrimary(isPrimary).
Exec(ctx); err != nil {
slog.Warn("skip mac", "mac", mac, "error", err)
}
}
return rows.Err()
}

// ── Images ────────────────────────────────────────────────────────────────

func (r *Runner) migrateImages(ctx context.Context, rep *Report) error {
rows, err := r.legacy.QueryContext(ctx, `
SELECT imageName, imageDesc, imagePath, imageEnabled, imageReplicate
  FROM images ORDER BY imageID`)
if err != nil {
return err
}
defer rows.Close()

for rows.Next() {
var name, desc, path string
var enabled, replicate bool
if err := rows.Scan(&name, &desc, &path, &enabled, &replicate); err != nil {
rep.Errors = append(rep.Errors, fmt.Sprintf("scan image: %v", err))
continue
}
if err := r.db.Image.Create().
SetName(name).
SetDescription(desc).
SetPath(path).
SetIsEnabled(enabled).
SetToReplicate(replicate).
Exec(ctx); err != nil {
slog.Warn("skip image", "name", name, "error", err)
rep.Skipped++
continue
}
rep.Images++
}
return rows.Err()
}

// ── Groups ────────────────────────────────────────────────────────────────

func (r *Runner) migrateGroups(ctx context.Context, rep *Report) error {
rows, err := r.legacy.QueryContext(ctx, `
SELECT groupName, groupDesc FROM groups ORDER BY groupID`)
if err != nil {
return err
}
defer rows.Close()

for rows.Next() {
var name, desc string
if err := rows.Scan(&name, &desc); err != nil {
rep.Errors = append(rep.Errors, fmt.Sprintf("scan group: %v", err))
continue
}
if err := r.db.Group.Create().
SetName(name).
SetDescription(desc).
Exec(ctx); err != nil {
slog.Warn("skip group", "name", name, "error", err)
rep.Skipped++
continue
}
rep.Groups++
}
return rows.Err()
}

// ── Snapins ───────────────────────────────────────────────────────────────

func (r *Runner) migrateSnapins(ctx context.Context, rep *Report) error {
rows, err := r.legacy.QueryContext(ctx, `
SELECT snapinName, snapinDesc, snapinFilename, snapinEnabled, snapinReplicate
  FROM snapins ORDER BY snapinID`)
if err != nil {
return err
}
defer rows.Close()

for rows.Next() {
var name, desc, file string
var enabled, replicate bool
if err := rows.Scan(&name, &desc, &file, &enabled, &replicate); err != nil {
rep.Errors = append(rep.Errors, fmt.Sprintf("scan snapin: %v", err))
continue
}
if err := r.db.Snapin.Create().
SetName(name).
SetDescription(desc).
SetFileName(file).
SetIsEnabled(enabled).
SetToReplicate(replicate).
Exec(ctx); err != nil {
slog.Warn("skip snapin", "name", name, "error", err)
rep.Skipped++
continue
}
rep.Snapins++
}
return rows.Err()
}

// ── Users ─────────────────────────────────────────────────────────────────

func (r *Runner) migrateUsers(ctx context.Context, rep *Report) error {
rows, err := r.legacy.QueryContext(ctx, `
SELECT userLogin, userFirstName, userLastName, userType
  FROM users ORDER BY userID`)
if err != nil {
return err
}
defer rows.Close()

for rows.Next() {
var username, first, last string
var userType int
_ = first
_ = last
if err := rows.Scan(&username, &first, &last, &userType); err != nil {
rep.Errors = append(rep.Errors, fmt.Sprintf("scan user: %v", err))
continue
}

role := entuser.RoleReadonly
if userType == 0 { // 0 = admin in legacy FOG
role = entuser.RoleAdmin
}

if err := r.db.User.Create().
SetUsername(username).
SetPasswordHash("").
SetRole(role).
SetIsActive(true).
Exec(ctx); err != nil {
slog.Warn("skip user", "username", username, "error", err)
rep.Skipped++
continue
}
rep.Users++
}
return rows.Err()
}

// ── Report ────────────────────────────────────────────────────────────────

// Report summarises the outcome of a migration run.
type Report struct {
Hosts   int
Images  int
Groups  int
Snapins int
Users   int
Skipped int
Errors  []string
}

func (r *Report) String() string {
return fmt.Sprintf(
"Migration complete:\n"+
"  Hosts:   %d\n"+
"  Images:  %d\n"+
"  Groups:  %d\n"+
"  Snapins: %d\n"+
"  Users:   %d\n"+
"  Skipped: %d\n"+
"  Errors:  %d",
r.Hosts, r.Images, r.Groups, r.Snapins, r.Users, r.Skipped, len(r.Errors),
)
}
