package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetMovies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := MovieListResponse{
			Items: []MovieItem{
				{
					ID:          "123",
					Title:       "Test Movie",
					OriginalTitle: "Test Movie",
					Duration:    "120",
					ContentRating: "12 anos",
					Synopsis:    "A test movie",
					IsPlaying:   true,
				},
			},
			Count: 1,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	origBaseURL := BaseURL
	BaseURL = server.URL
	defer func() { BaseURL = origBaseURL }()

	client := NewClient("cinepolis", 36, 1446)
	movies, err := client.GetMovies()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if movies.Count != 1 {
		t.Errorf("expected 1 movie, got %d", movies.Count)
	}

	if movies.Items[0].Title != "Test Movie" {
		t.Errorf("expected 'Test Movie', got '%s'", movies.Items[0].Title)
	}
}

func TestGetTheaters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := TheaterListResponse{
			Items: []TheaterItem{
				{
					ID:     "1446",
					Name:   "Cinépolis RioMar Kennedy",
					CityID: "36",
					TotalRooms: 6,
					Enabled:    true,
				},
			},
			Count: 1,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	origBaseURL := BaseURL
	BaseURL = server.URL
	defer func() { BaseURL = origBaseURL }()

	client := NewClient("cinepolis", 36, 1446)
	theaters, err := client.GetTheaters()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if theaters.Count != 1 {
		t.Errorf("expected 1 theater, got %d", theaters.Count)
	}

	if theaters.Items[0].Name != "Cinépolis RioMar Kennedy" {
		t.Errorf("expected 'Cinépolis RioMar Kennedy', got '%s'", theaters.Items[0].Name)
	}
}

func TestGetSessions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessions := []TheaterSession{
			{
				Date:  "2026-06-08",
				IsToday: true,
				Movies: []MovieSession{
					{
						ID:    "123",
						Title: "Test Movie",
						Rooms: []SessionRoom{
							{
								Name: "Sala 1",
								Sessions: []SessionItem{
									{
										ID:    "456",
										Price: 34.20,
										Room:  "Sala 1",
										Date: SessionDate{
											LocalDate: "2026-06-08T19:00:00-03:00",
											Hour:      "19:00",
										},
										Enabled: true,
									},
								},
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sessions)
	}))
	defer server.Close()

	origBaseURL := BaseURL
	BaseURL = server.URL
	defer func() { BaseURL = origBaseURL }()

	client := NewClient("cinepolis", 36, 1446)
	result, err := client.GetSessions(1446)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 day, got %d", len(result))
	}

	if len(result[0].Movies) != 1 {
		t.Fatalf("expected 1 movie, got %d", len(result[0].Movies))
	}

	if result[0].Movies[0].Rooms[0].Sessions[0].Price != 34.20 {
		t.Errorf("expected price 34.20, got %.2f", result[0].Movies[0].Rooms[0].Sessions[0].Price)
	}
}

func TestGetSessionDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		detail := SessionDetail{
			ID:    "456",
			Price: 34.20,
			Room:  "Sala 1",
			Enabled: true,
			Date: SessionDate{
				LocalDate: "2026-06-08T19:00:00-03:00",
				Hour:      "19:00",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(detail)
	}))
	defer server.Close()

	origBaseURL := BaseURL
	BaseURL = server.URL
	defer func() { BaseURL = origBaseURL }()

	client := NewClient("cinepolis", 36, 1446)
	session, err := client.GetSessionDetail("456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if session.Price != 34.20 {
		t.Errorf("expected price 34.20, got %.2f", session.Price)
	}

	if !session.Enabled {
		t.Error("expected session to be enabled")
	}
}

func TestHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`not found`))
	}))
	defer server.Close()

	origBaseURL := BaseURL
	BaseURL = server.URL
	defer func() { BaseURL = origBaseURL }()

	client := NewClient("cinepolis", 36, 1446)
	_, err := client.GetMovies()
	if err == nil {
		t.Fatal("expected error for 404 status")
	}
}
