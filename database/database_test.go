package database_test

import (
	"testing"

	"github.com/opencode/movie-watcher/database"
	"github.com/opencode/movie-watcher/internal/models"
)

func TestDatabaseInit(t *testing.T) {
	err := database.Init(":memory:")
	if err != nil {
		t.Fatalf("failed to init database: %v", err)
	}

	if database.GetDB() == nil {
		t.Fatal("expected DB to be non-nil")
	}
}

func TestMovieCRUD(t *testing.T) {
	err := database.Init(":memory:")
	if err != nil {
		t.Fatalf("failed to init database: %v", err)
	}

	movie := models.Movie{
		ExternalID:    "123",
		Title:         "Test Movie",
		OriginalTitle: "Test Movie",
		Duration:      120,
		ContentRating: "12 anos",
		IsPlaying:     true,
	}

	result := database.GetDB().Create(&movie)
	if result.Error != nil {
		t.Fatalf("failed to create movie: %v", result.Error)
	}

	var retrieved models.Movie
	database.GetDB().First(&retrieved, movie.ID)

	if retrieved.Title != "Test Movie" {
		t.Errorf("expected 'Test Movie', got '%s'", retrieved.Title)
	}
}

func TestSessionCRUD(t *testing.T) {
	err := database.Init(":memory:")
	if err != nil {
		t.Fatalf("failed to init database: %v", err)
	}

	movie := models.Movie{
		ExternalID:    "123",
		Title:         "Test Movie",
		OriginalTitle: "Test Movie",
		Duration:      120,
		IsPlaying:     true,
	}
	database.GetDB().Create(&movie)

	session := models.Session{
		ExternalID: "456",
		MovieID:    movie.ID,
		MovieTitle: movie.Title,
		Room:       "Sala 1",
		Price:      34.20,
		Enabled:    true,
	}

	result := database.GetDB().Create(&session)
	if result.Error != nil {
		t.Fatalf("failed to create session: %v", result.Error)
	}

	var retrieved models.Session
	database.GetDB().First(&retrieved, session.ID)

	if retrieved.Price != 34.20 {
		t.Errorf("expected price 34.20, got %.2f", retrieved.Price)
	}
}

func TestMonitoringCRUD(t *testing.T) {
	err := database.Init(":memory:")
	if err != nil {
		t.Fatalf("failed to init database: %v", err)
	}

	movie := models.Movie{Title: "Test Movie", ExternalID: "123"}
	database.GetDB().Create(&movie)

	session := models.Session{
		ExternalID: "456",
		MovieID:    movie.ID,
		MovieTitle: movie.Title,
		Room:       "Sala 1",
		Price:      34.20,
		Enabled:    true,
	}
	database.GetDB().Create(&session)

	monitoring := models.Monitoring{
		SessionID:   session.ID,
		LastPrice:   34.20,
		LastEnabled: true,
		Active:      true,
	}

	result := database.GetDB().Create(&monitoring)
	if result.Error != nil {
		t.Fatalf("failed to create monitoring: %v", result.Error)
	}

	var retrieved models.Monitoring
	database.GetDB().Preload("Session").First(&retrieved, monitoring.ID)

	if !retrieved.Active {
		t.Error("expected monitoring to be active")
	}

	if retrieved.Session.ID != session.ID {
		t.Errorf("expected session ID %d, got %d", session.ID, retrieved.Session.ID)
	}
}
