package pg

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/deicod/erm/observability/tracing"
	"github.com/deicod/erm/orm/runtime"
)

// Pool exposes the subset of pgxpool behaviour required by generated clients.
type Pool interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Close()
}

// ReplicaConfig describes a read replica participating in the cluster.
type ReplicaConfig struct {
	Name           string
	URL            string
	ReadOnly       bool
	MaxFollowerLag time.Duration
}

// ReplicaHealthReport captures the result of a health probe executed against a replica.
type ReplicaHealthReport struct {
	ReadOnly bool
	Lag      time.Duration
}

// ReplicaHealthCheck probes a replica connection to determine whether it can satisfy reads.
type ReplicaHealthCheck func(ctx context.Context, pool Pool) (ReplicaHealthReport, error)

// ReplicaReadOptions influence how reads target replicas.
type ReplicaReadOptions struct {
	MaxLag          time.Duration
	RequireReadOnly bool
	DisableFallback bool
}

// AllowFallback reports whether the read should fall back to the primary when the replica errors.
func (opts ReplicaReadOptions) AllowFallback() bool { return !opts.DisableFallback }

func (opts ReplicaReadOptions) normalised() ReplicaReadOptions {
	if opts.MaxLag < 0 {
		opts.MaxLag = 0
	}
	return opts
}

type replicaOptionsContextKey struct{}
type replicaPolicyContextKey struct{}
type forcePrimaryContextKey struct{}

// WithReplicaRead requests that read operations attempt to use a replica according to the
// provided options.
func WithReplicaRead(ctx context.Context, opts ReplicaReadOptions) context.Context {
	return context.WithValue(ctx, replicaOptionsContextKey{}, opts.normalised())
}

// WithReplicaPolicy annotates the context with a routing policy name. The driver resolves the
// policy against the set configured via UseReplicaPolicies.
func WithReplicaPolicy(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, replicaPolicyContextKey{}, name)
}

// WithPrimary ensures the primary writer connection is used even when replica policies would
// normally route the query elsewhere.
func WithPrimary(ctx context.Context) context.Context {
	return context.WithValue(ctx, forcePrimaryContextKey{}, true)
}

// DB manages writer and replica pools plus observability hooks.
type DB struct {
	Pool     Pool
	writer   Pool
	replicas []*replicaPool

	Observer runtime.QueryObserver

	healthCheck    ReplicaHealthCheck
	healthProbeSQL string
	healthInterval time.Duration

	routing replicaRouting
}

type replicaRouting struct {
	mu          sync.RWMutex
	defaultName string
	policies    map[string]ReplicaReadOptions
}

type replicaPool struct {
	name   string
	pool   Pool
	config ReplicaConfig

	mu     sync.RWMutex
	status replicaStatus
}

type replicaStatus struct {
	lastChecked time.Time
	healthy     bool
	lag         time.Duration
	readOnly    bool
	err         error
}

// PoolConfig describes connection pool tuning knobs exposed via configuration.
type PoolConfig struct {
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

// Option configures pgx connections.
type Option func(*pgxpool.Config)

const (
	defaultHealthInterval = 5 * time.Second
	replicaHealthProbeSQL = "SELECT pg_is_in_recovery() AS in_recovery, COALESCE(EXTRACT(EPOCH FROM (now() - pg_last_xact_replay_timestamp())), 0)::double precision AS replay_lag"
)

// Connect initialises a pgx pool with optional configuration overrides.
func Connect(ctx context.Context, url string, opts ...Option) (*DB, error) {
	return ConnectCluster(ctx, url, nil, opts...)
}

// ConnectCluster initialises a primary pool plus the provided replicas.
func ConnectCluster(ctx context.Context, url string, replicas []ReplicaConfig, opts ...Option) (*DB, error) {
	writer, err := newPool(ctx, url, opts...)
	if err != nil {
		return nil, err
	}

	db := &DB{
		Pool:           writer,
		writer:         writer,
		healthCheck:    defaultReplicaHealthCheck,
		healthProbeSQL: replicaHealthProbeSQL,
		healthInterval: defaultHealthInterval,
	}

	for i, cfg := range replicas {
		if cfg.URL == "" {
			continue
		}
		pool, err := newPool(ctx, cfg.URL, opts...)
		if err != nil {
			db.closeReplicas()
			writer.Close()
			return nil, err
		}
		name := cfg.Name
		if name == "" {
			name = fmt.Sprintf("replica-%d", i+1)
		}
		db.replicas = append(db.replicas, &replicaPool{
			name:   name,
			pool:   pool,
			config: cfg,
		})
	}

	return db, nil
}

// Close releases all underlying pools.
func (db *DB) Close() {
	if db == nil {
		return
	}
	switch {
	case db.writer != nil:
		db.writer.Close()
	case db.Pool != nil:
		db.Pool.Close()
	}
	db.closeReplicas()
}

func (db *DB) closeReplicas() {
	for _, replica := range db.replicas {
		if replica == nil || replica.pool == nil {
			continue
		}
		replica.pool.Close()
	}
}

// Writer returns the primary pool used for mutations and transactions.
func (db *DB) Writer() Pool { return db.writerPool() }

func (db *DB) writerPool() Pool {
	if db == nil {
		return nil
	}
	if db.writer != nil {
		return db.writer
	}
	return db.Pool
}

// Reader returns the pool selected for read operations after applying routing policies.
func (db *DB) Reader(ctx context.Context) Pool {
	pool, _, _, _ := db.readerPool(ctx)
	return pool
}

// Query routes ad-hoc read statements through the replica routing machinery.
func (db *DB) Query(ctx context.Context, table, sql string, args ...any) (pgx.Rows, error) {
	return db.queryWithOperation(ctx, runtime.OperationSelect, table, sql, args...)
}

// QueryRow routes read statements returning a single row through the replica routing machinery.
func (db *DB) QueryRow(ctx context.Context, table, sql string, args ...any) pgx.Row {
	return db.queryRowWithOperation(ctx, runtime.OperationSelect, table, sql, args...)
}

// Select issues a SELECT generated from runtime specs against the appropriate pool.
func (db *DB) Select(ctx context.Context, spec runtime.SelectSpec) (pgx.Rows, error) {
	sql, args := runtime.BuildSelectSQL(spec)
	return db.queryWithOperation(ctx, runtime.OperationSelect, spec.Table, sql, args...)
}

// Aggregate issues an aggregate query generated from runtime specs against the appropriate pool.
func (db *DB) Aggregate(ctx context.Context, spec runtime.AggregateSpec) pgx.Row {
	sql, args := runtime.BuildAggregateSQL(spec)
	return db.queryRowWithOperation(ctx, runtime.OperationAggregate, spec.Table, sql, args...)
}

type observationFlags struct {
	failover    bool
	reason      error
	healthCheck bool
}

func buildObservationAttrs(target string, replica bool, flags observationFlags) []tracing.Attribute {
	attrs := []tracing.Attribute{
		tracing.String("orm.target", target),
		tracing.Bool("orm.replica", replica),
	}
	if flags.failover {
		attrs = append(attrs, tracing.Bool("orm.failover", true))
		if flags.reason != nil {
			attrs = append(attrs, tracing.String("orm.failover_reason", flags.reason.Error()))
		}
	}
	if flags.healthCheck {
		attrs = append(attrs, tracing.Bool("orm.health_check", true))
	}
	return attrs
}

func (db *DB) queryWithOperation(ctx context.Context, op runtime.QueryOperation, table, sql string, args ...any) (pgx.Rows, error) {
	pool, usedReplica, policy, target := db.readerPool(ctx)
	attrs := buildObservationAttrs(target, usedReplica, observationFlags{})
	obs := db.Observer.Observe(ctx, op, table, sql, args, runtime.WithObservationAttributes(attrs...))
	rows, err := pool.Query(obs.Context(), sql, args...)
	obs.End(err)
	if err != nil && usedReplica && policy.AllowFallback() {
		return db.retrySelectOnPrimary(ctx, op, table, sql, args, err)
	}
	return rows, err
}

func (db *DB) queryRowWithOperation(ctx context.Context, op runtime.QueryOperation, table, sql string, args ...any) pgx.Row {
	pool, usedReplica, policy, target := db.readerPool(ctx)
	attrs := buildObservationAttrs(target, usedReplica, observationFlags{})
	obs := db.Observer.Observe(ctx, op, table, sql, args, runtime.WithObservationAttributes(attrs...))
	row := pool.QueryRow(obs.Context(), sql, args...)
	observed := &observedRow{Row: row, obs: obs}
	if usedReplica && policy.AllowFallback() {
		observed.fallback = func(prevErr error) (pgx.Row, runtime.QueryObservation, error) {
			return db.queryRowOnPrimary(ctx, op, table, sql, args, prevErr)
		}
	}
	return observed
}

func (db *DB) retrySelectOnPrimary(ctx context.Context, op runtime.QueryOperation, table, sql string, args []any, prevErr error) (pgx.Rows, error) {
	attrs := buildObservationAttrs("primary", false, observationFlags{failover: true, reason: prevErr})
	obs := db.Observer.Observe(ctx, op, table, sql, args, runtime.WithObservationAttributes(attrs...))
	writer := db.writerPool()
	if writer == nil {
		err := fmt.Errorf("pg: primary pool unavailable for failover")
		obs.End(err)
		return nil, err
	}
	rows, err := writer.Query(obs.Context(), sql, args...)
	obs.End(err)
	return rows, err
}

func (db *DB) queryRowOnPrimary(ctx context.Context, op runtime.QueryOperation, table, sql string, args []any, prevErr error) (pgx.Row, runtime.QueryObservation, error) {
	attrs := buildObservationAttrs("primary", false, observationFlags{failover: true, reason: prevErr})
	obs := db.Observer.Observe(ctx, op, table, sql, args, runtime.WithObservationAttributes(attrs...))
	writer := db.writerPool()
	if writer == nil {
		err := fmt.Errorf("pg: primary pool unavailable for failover")
		return nil, obs, err
	}
	row := writer.QueryRow(obs.Context(), sql, args...)
	return row, obs, nil
}

func (db *DB) readerPool(ctx context.Context) (Pool, bool, ReplicaReadOptions, string) {
	if ctx == nil {
		ctx = context.Background()
	}
	writer := db.writerPool()
	if writer == nil {
		return nil, false, ReplicaReadOptions{}, "primary"
	}
	if force, ok := ctx.Value(forcePrimaryContextKey{}).(bool); ok && force {
		return writer, false, ReplicaReadOptions{}, "primary"
	}

	opts, usePolicy := replicaOptionsFromContext(ctx)
	var policyName string
	if !usePolicy {
		if name, ok := ctx.Value(replicaPolicyContextKey{}).(string); ok {
			policyName = name
		}
	}

	if !usePolicy {
		if policyName != "" {
			if policy, ok := db.routing.lookup(policyName); ok {
				opts = policy
				usePolicy = true
			}
		}
	}

	if !usePolicy {
		if policy, ok := db.routing.defaultPolicy(); ok {
			opts = policy
			usePolicy = true
		}
	}

	if !usePolicy || len(db.replicas) == 0 {
		return writer, false, ReplicaReadOptions{}, "primary"
	}

	replica, status, ok := db.selectReplica(ctx, opts)
	if !ok {
		return writer, false, opts, "primary"
	}

	if opts.RequireReadOnly && !status.readOnly {
		return writer, false, opts, "primary"
	}
	if opts.MaxLag > 0 && status.lag > opts.MaxLag {
		return writer, false, opts, "primary"
	}
	return replica.pool, true, opts, replica.name
}

func replicaOptionsFromContext(ctx context.Context) (ReplicaReadOptions, bool) {
	opts, ok := ctx.Value(replicaOptionsContextKey{}).(ReplicaReadOptions)
	if !ok {
		return ReplicaReadOptions{}, false
	}
	return opts.normalised(), true
}

func (db *DB) selectReplica(ctx context.Context, opts ReplicaReadOptions) (*replicaPool, replicaStatus, bool) {
	for _, replica := range db.replicas {
		status := replica.snapshot()
		if db.healthInterval <= 0 || status.stale(db.healthInterval) {
			status = db.runReplicaHealthCheck(ctx, replica)
		}
		if !status.healthy {
			continue
		}
		if opts.RequireReadOnly && !status.readOnly {
			continue
		}
		if opts.MaxLag > 0 && status.lag > opts.MaxLag {
			continue
		}
		return replica, status, true
	}
	return nil, replicaStatus{}, false
}

func (db *DB) runReplicaHealthCheck(ctx context.Context, replica *replicaPool) replicaStatus {
	replica.mu.Lock()
	defer replica.mu.Unlock()

	status := replica.status
	if db.healthInterval > 0 && !status.lastChecked.IsZero() && time.Since(status.lastChecked) < db.healthInterval {
		return status
	}

	attrs := buildObservationAttrs(replica.name, true, observationFlags{healthCheck: true})
	obs := db.Observer.Observe(ctx, runtime.OperationSelect, "pg_replica_health", db.healthProbeSQL, nil, runtime.WithObservationAttributes(attrs...))
	report, err := db.healthChecker()(obs.Context(), replica.pool)

	status.lastChecked = time.Now()
	if err != nil {
		status.healthy = false
		status.err = err
		replica.status = status
		obs.End(err)
		return status
	}

	status.readOnly = report.ReadOnly
	status.lag = report.Lag
	status.err = nil
	status.healthy = true

	if replica.config.ReadOnly && !report.ReadOnly {
		status.healthy = false
		status.err = fmt.Errorf("replica %s is not read-only", replica.name)
	}
	if replica.config.MaxFollowerLag > 0 && report.Lag > replica.config.MaxFollowerLag {
		status.healthy = false
		status.err = fmt.Errorf("replica %s lag %s exceeds max %s", replica.name, report.Lag, replica.config.MaxFollowerLag)
	}

	replica.status = status
	obs.End(status.err)
	return status
}

func (db *DB) healthChecker() ReplicaHealthCheck {
	if db.healthCheck == nil {
		return defaultReplicaHealthCheck
	}
	return db.healthCheck
}

// UseObserver attaches a query observer to the database handle.
func (db *DB) UseObserver(observer runtime.QueryObserver) {
	if db == nil {
		return
	}
	db.Observer = observer
}

// UseReplicaHealthCheck overrides the health probe used for replicas.
func (db *DB) UseReplicaHealthCheck(check ReplicaHealthCheck) {
	if db == nil {
		return
	}
	db.healthCheck = check
}

// SetReplicaHealthInterval adjusts the interval between health checks. Zero or negative values
// force probes before every read attempt.
func (db *DB) SetReplicaHealthInterval(interval time.Duration) {
	if db == nil {
		return
	}
	db.healthInterval = interval
}

// UseReplicaPolicies registers named replica routing policies. The default policy name is applied
// when reads do not explicitly opt-in via context.
func (db *DB) UseReplicaPolicies(defaultPolicy string, policies map[string]ReplicaReadOptions) {
	if db == nil {
		return
	}
	db.routing.set(defaultPolicy, policies)
}

func (r *replicaRouting) set(defaultPolicy string, policies map[string]ReplicaReadOptions) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defaultName = defaultPolicy
	if len(policies) == 0 {
		r.policies = nil
		return
	}
	r.policies = make(map[string]ReplicaReadOptions, len(policies))
	for name, policy := range policies {
		r.policies[name] = policy.normalised()
	}
}

func (r *replicaRouting) lookup(name string) (ReplicaReadOptions, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.policies == nil {
		return ReplicaReadOptions{}, false
	}
	policy, ok := r.policies[name]
	return policy, ok
}

func (r *replicaRouting) defaultPolicy() (ReplicaReadOptions, bool) {
	r.mu.RLock()
	name := r.defaultName
	r.mu.RUnlock()
	if name == "" {
		return ReplicaReadOptions{}, false
	}
	return r.lookup(name)
}

func (p *replicaPool) snapshot() replicaStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}

func (s replicaStatus) stale(interval time.Duration) bool {
	if interval <= 0 {
		return true
	}
	if s.lastChecked.IsZero() {
		return true
	}
	return time.Since(s.lastChecked) > interval
}

func defaultReplicaHealthCheck(ctx context.Context, pool Pool) (ReplicaHealthReport, error) {
	row := pool.QueryRow(ctx, replicaHealthProbeSQL)
	var readOnly bool
	var lagSeconds float64
	if err := row.Scan(&readOnly, &lagSeconds); err != nil {
		return ReplicaHealthReport{}, err
	}
	return ReplicaHealthReport{
		ReadOnly: readOnly,
		Lag:      time.Duration(lagSeconds * float64(time.Second)),
	}, nil
}

func newPool(ctx context.Context, url string, opts ...Option) (Pool, error) {
	cfg, err := newPoolConfig(url, opts...)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

type observedRow struct {
	pgx.Row
	obs          runtime.QueryObservation
	once         sync.Once
	fallback     func(error) (pgx.Row, runtime.QueryObservation, error)
	fallbackOnce sync.Once
}

func (r *observedRow) Scan(dest ...any) error {
	err := r.Row.Scan(dest...)
	if err != nil && r.fallback != nil {
		var (
			fbRow pgx.Row
			fbObs runtime.QueryObservation
			fbErr error
		)
		r.fallbackOnce.Do(func() {
			fbRow, fbObs, fbErr = r.fallback(err)
			if fbErr != nil {
				if fbObs.Context() != nil {
					fbObs.End(fbErr)
				}
				// If fallback setup failed, we must still close the original observation
				// with the original error.
				r.obs.End(err)
				r.obs = runtime.QueryObservation{}
				return
			}
			r.obs.End(err)
			r.obs = fbObs
			r.Row = fbRow
			err = r.Row.Scan(dest...)
		})
		if fbErr != nil {
			err = fbErr
		}
	}
	r.once.Do(func() { r.obs.End(err) })
	return err
}

func newPoolConfig(url string, opts ...Option) (*pgxpool.Config, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	applyDefaults(cfg)
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(cfg)
	}
	return cfg, nil
}

func applyDefaults(cfg *pgxpool.Config) {
	cfg.MaxConns = 10
	cfg.MinConns = 2
	cfg.MaxConnLifetime = time.Hour
}

// WithMaxConns sets the maximum pool size.
func WithMaxConns(n int32) Option {
	return func(cfg *pgxpool.Config) {
		cfg.MaxConns = n
	}
}

// WithMinConns sets the minimum pool size.
func WithMinConns(n int32) Option {
	return func(cfg *pgxpool.Config) {
		cfg.MinConns = n
	}
}

// WithMaxConnLifetime configures the maximum connection lifetime.
func WithMaxConnLifetime(d time.Duration) Option {
	return func(cfg *pgxpool.Config) { cfg.MaxConnLifetime = d }
}

// WithMaxConnIdleTime configures how long an idle connection may remain in the pool.
func WithMaxConnIdleTime(d time.Duration) Option {
	return func(cfg *pgxpool.Config) { cfg.MaxConnIdleTime = d }
}

// WithHealthCheckPeriod configures the background health check period.
func WithHealthCheckPeriod(d time.Duration) Option {
	return func(cfg *pgxpool.Config) { cfg.HealthCheckPeriod = d }
}

// WithPoolConfig applies a group of pool settings derived from configuration.
func WithPoolConfig(pc PoolConfig) Option {
	return func(cfg *pgxpool.Config) {
		if pc.MaxConns > 0 {
			cfg.MaxConns = pc.MaxConns
		}
		if pc.MinConns > 0 {
			cfg.MinConns = pc.MinConns
		}
		if pc.MaxConnLifetime > 0 {
			cfg.MaxConnLifetime = pc.MaxConnLifetime
		}
		if pc.MaxConnIdleTime > 0 {
			cfg.MaxConnIdleTime = pc.MaxConnIdleTime
		}
		if pc.HealthCheckPeriod > 0 {
			cfg.HealthCheckPeriod = pc.HealthCheckPeriod
		}
	}
}

// WithTracer enables pgx tracing using the provided tracer abstraction.
func WithTracer(tracer tracing.Tracer) Option {
	return func(cfg *pgxpool.Config) {
		if tracer == nil {
			cfg.ConnConfig.Tracer = nil
			return
		}
		cfg.ConnConfig.Tracer = newPGXTracer(tracer)
	}
}
