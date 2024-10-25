package web

import (
	"bytes"
	"fmt"
	"log"
	"net/http"

	"github.com/eldelto/core/internal/boltutil"
	"go.etcd.io/bbolt"
)

type StatisticsRepository interface {
	AddView(page string) error
	GetViews() (map[string]uint, error)
}

type StatisticsModule struct {
	statsRepo StatisticsRepository
}

func NewStatisticsModule(repo StatisticsRepository) *StatisticsModule {
	return &StatisticsModule{
		statsRepo: repo,
	}
}

func (m *StatisticsModule) Controller() *Controller {
	return &Controller{
		BasePath: "/",
		Handlers: map[Endpoint]Handler{
			{Method: http.MethodGet, Path: "/stats"}: m.getStatistics(),
		},
	}
}

func (m *StatisticsModule) getStatistics() Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		views, err := m.statsRepo.GetViews()
		if err != nil {
			return err
		}

		viewsPage := fmt.Sprintf("Stats: %v", views)
		buff := bytes.NewBufferString(viewsPage)
		buff.WriteTo(w)
		return nil
	}
}

func (m *StatisticsModule) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := m.statsRepo.AddView(r.URL.Path); err != nil {
			log.Println(err)
		}

		next.ServeHTTP(w, r)
	})
}

const viewsBucket = "statistics.views"

type BoltStatisticsRepository struct {
	db *bbolt.DB
}

func NewBoltStatisticsRepository(db *bbolt.DB) (*BoltStatisticsRepository, error) {
	if err := boltutil.EnsureBucketExists(db, viewsBucket); err != nil {
		return nil, fmt.Errorf("new bolt statistics repository: %w", err)
	}

	return &BoltStatisticsRepository{
		db: db,
	}, nil
}

func (r *BoltStatisticsRepository) AddView(page string) error {
	if err := boltutil.Update(r.db, viewsBucket, page,
		func(oldViews uint) uint { return oldViews + 1 }); err != nil {
		return fmt.Errorf("adding view for page %q: %w", page, err)
	}

	return nil
}

func (r *BoltStatisticsRepository) GetViews() (map[string]uint, error) {
	views, err := boltutil.List[uint](r.db, viewsBucket)
	if err != nil {
		return nil, fmt.Errorf("getting views: %w", err)
	}

	return views, nil
}
