package cinema

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/opencode/movie-watcher/configs"
	"github.com/opencode/movie-watcher/database"
	"github.com/opencode/movie-watcher/internal/api"
	"github.com/opencode/movie-watcher/internal/models"
)

func newTestService(t *testing.T, handler http.HandlerFunc) *Service {
	t.Helper()

	configs.AppConfig = &configs.Config{
		Database:    configs.DatabaseConfig{Path: ":memory:"},
		CityID:      36,
		TheaterID:   1446,
		Partnership: "cinepolis",
	}

	if err := database.Init(":memory:"); err != nil {
		t.Fatalf("failed to init database: %v", err)
	}

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	origBaseURL := api.BaseURL
	api.BaseURL = server.URL
	t.Cleanup(func() { api.BaseURL = origBaseURL })

	return &Service{client: api.NewClient("cinepolis", 36, 1446)}
}

func sessionsResponse(theaterID string, sessions []api.SessionItem) []api.TheaterSession {
	return []api.TheaterSession{
		{
			Date: "2026-06-08",
			Movies: []api.MovieSession{
				{
					ID: "123",
					Rooms: []api.SessionRoom{
						{Name: "Sala 1", Sessions: sessions},
					},
				},
			},
		},
	}
}

func TestLoadSessionsOnlyTouchesOwnTheater(t *testing.T) {
	s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		resp := sessionsResponse("1446", []api.SessionItem{
			{ID: "s1", Price: 30, Enabled: true, Date: api.SessionDate{LocalDate: "2026-06-08T19:00:00-03:00"}},
		})
		_ = json.NewEncoder(w).Encode(resp)
	})

	database.GetDB().Create(&models.Movie{ExternalID: "123", Title: "Movie A"})

	otherTheaterSession := models.Session{ExternalID: "other-1", TheaterID: 9999, MovieTitle: "Other cinema movie"}
	database.GetDB().Create(&otherTheaterSession)

	if _, err := s.LoadSessions(1446); err != nil {
		t.Fatalf("LoadSessions failed: %v", err)
	}

	var stillThere models.Session
	if err := database.GetDB().First(&stillThere, otherTheaterSession.ID).Error; err != nil {
		t.Fatalf("expected session belonging to a different theater to survive, got error: %v", err)
	}
}

func TestLoadSessionsPreservesIDAcrossRefresh(t *testing.T) {
	s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		resp := sessionsResponse("1446", []api.SessionItem{
			{ID: "s1", Price: 30, Enabled: true, Date: api.SessionDate{LocalDate: "2026-06-08T19:00:00-03:00"}},
		})
		_ = json.NewEncoder(w).Encode(resp)
	})

	database.GetDB().Create(&models.Movie{ExternalID: "123", Title: "Movie A"})

	sessions, err := s.LoadSessions(1446)
	if err != nil {
		t.Fatalf("first LoadSessions failed: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	firstID := sessions[0].ID

	monitoring, err := s.StartMonitoring(firstID)
	if err != nil {
		t.Fatalf("StartMonitoring failed: %v", err)
	}

	sessions, err = s.LoadSessions(1446)
	if err != nil {
		t.Fatalf("second LoadSessions failed: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session after refresh, got %d", len(sessions))
	}

	if sessions[0].ID != firstID {
		t.Fatalf("expected session ID to stay %d across refresh, got %d", firstID, sessions[0].ID)
	}

	var stillActive models.Monitoring
	if err := database.GetDB().First(&stillActive, monitoring.ID).Error; err != nil {
		t.Fatalf("expected monitoring row to still reference a valid session: %v", err)
	}
}

func TestLoadSessionsRemovesSessionsGoneFromFeed(t *testing.T) {
	call := 0
	s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		call++
		var resp []api.TheaterSession
		if call == 1 {
			resp = sessionsResponse("1446", []api.SessionItem{
				{ID: "s1", Price: 30, Enabled: true, Date: api.SessionDate{LocalDate: "2026-06-08T19:00:00-03:00"}},
			})
		} else {
			resp = sessionsResponse("1446", nil)
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	database.GetDB().Create(&models.Movie{ExternalID: "123", Title: "Movie A"})

	if _, err := s.LoadSessions(1446); err != nil {
		t.Fatalf("first LoadSessions failed: %v", err)
	}

	sessions, err := s.LoadSessions(1446)
	if err != nil {
		t.Fatalf("second LoadSessions failed: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("expected session no longer in the feed to be removed, got %d", len(sessions))
	}
}

func TestResolveTheaterIDsFallsBackToConfig(t *testing.T) {
	s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {})

	ids, err := s.ResolveTheaterIDs()
	if err != nil {
		t.Fatalf("ResolveTheaterIDs failed: %v", err)
	}
	if len(ids) != 1 {
		t.Fatalf("expected 1 fallback theater ID, got %d", len(ids))
	}
}

func TestResolveTheaterIDsUsesMonitoredCinemas(t *testing.T) {
	s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {})

	c1 := models.Cinema{ExternalID: "111", Name: "Cinema A", Monitored: true}
	c2 := models.Cinema{ExternalID: "222", Name: "Cinema B", Monitored: false}
	database.GetDB().Create(&c1)
	database.GetDB().Create(&c2)

	ids, err := s.ResolveTheaterIDs()
	if err != nil {
		t.Fatalf("ResolveTheaterIDs failed: %v", err)
	}
	if len(ids) != 1 || ids[0] != 111 {
		t.Fatalf("expected only monitored cinema 111, got %v", ids)
	}
}

func TestSetCinemaMonitoredUnknownID(t *testing.T) {
	s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {})

	if err := s.SetCinemaMonitored(999, true); err == nil {
		t.Fatal("expected error for unknown cinema ID")
	}
}

func TestStartMonitoringRecordsPriceHistory(t *testing.T) {
	s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {})

	movie := models.Movie{ExternalID: "123", Title: "Movie A"}
	database.GetDB().Create(&movie)
	session := models.Session{ExternalID: "s1", MovieID: movie.ID, MovieTitle: movie.Title, Price: 30, Enabled: true}
	database.GetDB().Create(&session)

	if _, err := s.StartMonitoring(session.ID); err != nil {
		t.Fatalf("StartMonitoring failed: %v", err)
	}

	history, err := s.GetPriceHistory(session.ID)
	if err != nil {
		t.Fatalf("GetPriceHistory failed: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 history entry after start, got %d", len(history))
	}
	if history[0].Price != 30 {
		t.Errorf("expected recorded price 30, got %.2f", history[0].Price)
	}
}

func TestRecordPriceSnapshotNoPrevious(t *testing.T) {
	s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {})

	movie := models.Movie{ExternalID: "123", Title: "Movie A"}
	database.GetDB().Create(&movie)
	session := models.Session{ExternalID: "s1", MovieID: movie.ID, MovieTitle: movie.Title, Price: 30, Enabled: true}
	database.GetDB().Create(&session)

	_, hadPrevious, err := s.RecordPriceSnapshot(session.ID, session.Price, session.Enabled)
	if err != nil {
		t.Fatalf("RecordPriceSnapshot failed: %v", err)
	}
	if hadPrevious {
		t.Fatal("expected no previous price on first snapshot")
	}
}

func TestRecordPriceSnapshotDetectsDrop(t *testing.T) {
	s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {})

	movie := models.Movie{ExternalID: "123", Title: "Movie A"}
	database.GetDB().Create(&movie)
	session := models.Session{ExternalID: "s1", MovieID: movie.ID, MovieTitle: movie.Title, Price: 34.20, Enabled: true}
	database.GetDB().Create(&session)

	if _, _, err := s.RecordPriceSnapshot(session.ID, 34.20, true); err != nil {
		t.Fatalf("first RecordPriceSnapshot failed: %v", err)
	}

	previousPrice, hadPrevious, err := s.RecordPriceSnapshot(session.ID, 31.90, true)
	if err != nil {
		t.Fatalf("second RecordPriceSnapshot failed: %v", err)
	}
	if !hadPrevious {
		t.Fatal("expected a previous price on second snapshot")
	}
	if previousPrice != 34.20 {
		t.Errorf("expected previous price 34.20, got %.2f", previousPrice)
	}
}

func TestRecordPriceSnapshotSameOrHigherPrice(t *testing.T) {
	s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {})

	movie := models.Movie{ExternalID: "123", Title: "Movie A"}
	database.GetDB().Create(&movie)
	session := models.Session{ExternalID: "s1", MovieID: movie.ID, MovieTitle: movie.Title, Price: 30, Enabled: true}
	database.GetDB().Create(&session)

	if _, _, err := s.RecordPriceSnapshot(session.ID, 30, true); err != nil {
		t.Fatalf("first RecordPriceSnapshot failed: %v", err)
	}

	previousPrice, hadPrevious, err := s.RecordPriceSnapshot(session.ID, 35, true)
	if err != nil {
		t.Fatalf("second RecordPriceSnapshot failed: %v", err)
	}
	if !hadPrevious || previousPrice != 30 {
		t.Fatalf("expected previous price 30 (had=true), got %.2f (had=%v)", previousPrice, hadPrevious)
	}
}

func TestGetActiveSessionsFiltersTheaterAndTime(t *testing.T) {
	s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {})

	movie := models.Movie{ExternalID: "123", Title: "Movie A"}
	database.GetDB().Create(&movie)

	now := time.Now()
	sessions := []models.Session{
		{ExternalID: "past", MovieID: movie.ID, MovieTitle: movie.Title, TheaterID: 1446, DateTime: now.Add(-1 * time.Hour), Price: 30},
		{ExternalID: "later-today", MovieID: movie.ID, MovieTitle: movie.Title, TheaterID: 1446, DateTime: now.Add(2 * time.Hour), Price: 30},
		{ExternalID: "tomorrow", MovieID: movie.ID, MovieTitle: movie.Title, TheaterID: 1446, DateTime: now.Add(26 * time.Hour), Price: 30},
		{ExternalID: "other-theater", MovieID: movie.ID, MovieTitle: movie.Title, TheaterID: 9999, DateTime: now.Add(2 * time.Hour), Price: 30},
	}
	for i := range sessions {
		database.GetDB().Create(&sessions[i])
	}

	active, err := s.GetActiveSessions([]int{1446})
	if err != nil {
		t.Fatalf("GetActiveSessions failed: %v", err)
	}

	if len(active) != 2 {
		t.Fatalf("expected 2 active sessions (later-today, tomorrow), got %d", len(active))
	}
	for _, sess := range active {
		if sess.ExternalID == "past" || sess.ExternalID == "other-theater" {
			t.Errorf("did not expect session %q in active results", sess.ExternalID)
		}
	}
}
