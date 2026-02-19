package postgres

//
//import (
//	"context"
//	"database/sql"
//	"fmt"
//	"log"
//	"os"
//	"path/filepath"
//	"reflect"
//	"testing"
//
//	_ "github.com/golang-migrate/migrate/v4/database/postgres"
//	_ "github.com/golang-migrate/migrate/v4/source/file"
//	_ "github.com/jackc/pgx/v5/stdlib"
//	"github.com/lbrezgin/telemetry/internal/config"
//	"github.com/lbrezgin/telemetry/internal/model"
//	"github.com/ory/dockertest/v3"
//	"github.com/ory/dockertest/v3/docker"
//)
//
//var (
//	db  *sql.DB
//	cfg *config.Repo
//)
//
//var (
//	// DB
//	testUser         = "test"
//	testPassword     = "testpassword"
//	testDB           = "test-telemetry-db"
//	migrationPathAbs = "../../../migrations"
//
//	// Docker
//	repository = "postgres"
//	tag        = "18"
//	name       = "metrics-repository-tests"
//	env        = []string{
//		"POSTGRES_DB=" + testDB,
//		"POSTGRES_USER=" + testUser,
//		"POSTGRES_PASSWORD=" + testPassword,
//	}
//	exposedPorts = []string{"5432/tcp"}
//)
//
//func TestMain(m *testing.M) {
//	i, err := runMain(m)
//	if err != nil {
//		log.Fatal(err)
//	}
//	os.Exit(i)
//}
//
//func runMain(m *testing.M) (int, error) {
//	pool, err := dockertest.NewPool("")
//	if err != nil {
//		return 1, fmt.Errorf("could not construct pool: %w", err)
//	}
//
//	err = pool.Client.Ping()
//	if err != nil {
//		return 1, fmt.Errorf("could not connect to Docker: %w", err)
//	}
//
//	pg, err := pool.RunWithOptions(
//		&dockertest.RunOptions{
//			Repository:   repository,
//			Tag:          tag,
//			Name:         name,
//			Env:          env,
//			ExposedPorts: exposedPorts,
//		}, func(config *docker.HostConfig) {
//			config.AutoRemove = true
//			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
//		},
//	)
//	if err != nil {
//		return 1, fmt.Errorf("failed to run the postgres container: %w", err)
//	}
//
//	defer func() {
//		if err = pool.Purge(pg); err != nil {
//			log.Printf("failed to purge the postgres container: %s", err)
//		}
//	}()
//
//	// Open database connection
//	dsn := fmt.Sprintf("postgres://%s:%s@localhost:%s/%s?sslmode=disable", testUser, testPassword, pg.GetPort(exposedPorts[0]), testDB)
//	if err = pool.Retry(func() error {
//		db, err = sql.Open("pgx", dsn)
//		if err != nil {
//			return fmt.Errorf("failed to open database connection: %w", err)
//		}
//		return db.Ping()
//	}); err != nil {
//		return 1, err
//	}
//
//	defer func() {
//		if err = db.Close(); err != nil {
//			log.Printf("failed to close the database connection: %s", err)
//		}
//	}()
//
//	abs, _ := filepath.Abs(migrationPathAbs)
//	source := "file://" + abs
//
//	cfg := &config.Repo{
//		DSN:            dsn,
//		Driver:         "pgx",
//		MigrationsPath: source,
//	}
//
//	rePo, err := New(db, cfg)
//	if err != nil {
//		return 1, fmt.Errorf("failed to create repository: %w", err)
//	}
//	if err = rePo.RunMigrations(); err != nil {
//		return 1, fmt.Errorf("failed to run migrations: %w", err)
//	}
//
//	return m.Run(), nil
//}
//
//func Test_repo_Upsert(t *testing.T) {
//	type fields struct {
//		db  *sql.DB
//		cfg *config.Repo
//	}
//	type args struct {
//		ctx    context.Context
//		metric *model.Metrics
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantErr bool
//	}{
//		{
//			name: "return ErrNilMetric if metric is nil",
//			fields: fields{
//				db:  db,
//				cfg: cfg,
//			},
//			args: args{
//				ctx:    context.Background(),
//				metric: nil,
//			},
//			wantErr: true,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			r := &repo{
//				db:  tt.fields.db,
//				cfg: tt.fields.cfg,
//			}
//			if err := r.Upsert(tt.args.ctx, tt.args.metric); (err != nil) != tt.wantErr {
//				t.Errorf("Upsert() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}
//
//func Test_repo_Find(t *testing.T) {
//	type fields struct {
//		db  *sql.DB
//		cfg *config.Repo
//	}
//	type args struct {
//		ctx   context.Context
//		id    string
//		mtype string
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		want    *model.Metrics
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			r := &repo{
//				db:  tt.fields.db,
//				cfg: tt.fields.cfg,
//			}
//			got, err := r.Find(tt.args.ctx, tt.args.id, tt.args.mtype)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("Find() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			if !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("Find() got = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func Test_repo_List(t *testing.T) {
//	type fields struct {
//		db  *sql.DB
//		cfg *config.Repo
//	}
//	type args struct {
//		ctx context.Context
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		want    []model.Metrics
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			r := &repo{
//				db:  tt.fields.db,
//				cfg: tt.fields.cfg,
//			}
//			got, err := r.List(tt.args.ctx)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			if !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("List() got = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func Test_repo_Restore(t *testing.T) {
//	type fields struct {
//		db  *sql.DB
//		cfg *config.Repo
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			r := &repo{
//				db:  tt.fields.db,
//				cfg: tt.fields.cfg,
//			}
//			if err := r.Restore(); (err != nil) != tt.wantErr {
//				t.Errorf("Restore() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}
