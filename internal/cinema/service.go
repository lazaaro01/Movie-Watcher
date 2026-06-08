package cinema

import (
	"fmt"
	"strconv"
	"time"

	"github.com/opencode/movie-watcher/configs"
	"github.com/opencode/movie-watcher/database"
	"github.com/opencode/movie-watcher/internal/api"
	"github.com/opencode/movie-watcher/internal/models"
)

type Service struct {
	client *api.Client
}

func NewService() *Service {
	cfg := configs.AppConfig
	client := api.NewClient(cfg.Partnership, cfg.CityID, cfg.TheaterID)
	return &Service{client: client}
}

func (s *Service) updateOrCreateMovie(item api.MovieItem) (*models.Movie, error) {
	duration, _ := strconv.Atoi(item.Duration)

	genre := ""
	if len(item.Genres) > 0 {
		genre = item.Genres[0]
	}

	movie := &models.Movie{
		ExternalID:    item.ID,
		Title:         item.Title,
		OriginalTitle: item.OriginalTitle,
		Duration:      duration,
		ContentRating: item.ContentRating,
		Synopsis:      item.Synopsis,
		Cast:          item.Cast,
		Director:      item.Director,
		Distributor:   item.Distributor,
		Genre:         genre,
		URLKey:        item.URLKey,
		ImageURL:      item.ImageFeatured,
		IsPlaying:     item.IsPlaying,
	}

	err := database.GetDB().Where("external_id = ?", item.ID).Assign(movie).FirstOrCreate(movie).Error
	return movie, err
}

func (s *Service) updateOrCreateSession(sessionID string, movie *models.Movie, theaterID uint, room string, roomID string, dateTime time.Time, price float64, enabled bool, hasSeatSelection bool, defaultSector string, externalURL string, sessionTypeInfo string) (*models.Session, error) {
	session := &models.Session{
		ExternalID:       sessionID,
		MovieID:          movie.ID,
		MovieTitle:       movie.Title,
		TheaterID:        theaterID,
		Room:             room,
		RoomID:           roomID,
		DateTime:         dateTime,
		Price:            price,
		Enabled:          enabled,
		HasSeatSelection: hasSeatSelection,
		DefaultSector:    defaultSector,
		ExternalURL:      externalURL,
		SessionType:      sessionTypeInfo,
	}

	err := database.GetDB().Where("external_id = ?", sessionID).Assign(session).FirstOrCreate(session).Error
	return session, err
}

func (s *Service) LoadMovies() ([]models.Movie, error) {
	resp, err := s.client.GetMovies()
	if err != nil {
		return nil, fmt.Errorf("error fetching movies: %w", err)
	}

	for _, item := range resp.Items {
		_, err := s.updateOrCreateMovie(item)
		if err != nil {
			return nil, fmt.Errorf("error saving movie %s: %w", item.Title, err)
		}
	}

	var movies []models.Movie
	database.GetDB().Order("title ASC").Find(&movies)
	return movies, nil
}

func (s *Service) ListMovies() ([]models.Movie, error) {
	var movies []models.Movie
	database.GetDB().Where("is_playing = ?", true).Order("title ASC").Find(&movies)
	if len(movies) > 0 {
		return movies, nil
	}
	return s.LoadMovies()
}

func (s *Service) SearchMovies(query string) ([]models.Movie, error) {
	var movies []models.Movie
	database.GetDB().Where("is_playing = ? AND (title LIKE ? OR original_title LIKE ?)", true, "%"+query+"%", "%"+query+"%").Order("title ASC").Find(&movies)
	return movies, nil
}

func (s *Service) LoadSessions(theaterID int) ([]models.Session, error) {
	cfg := configs.AppConfig
	sessionsResp, err := s.client.GetSessions(theaterID)
	if err != nil {
		return nil, fmt.Errorf("error fetching sessions: %w", err)
	}

	database.GetDB().Where("1 = 1").Delete(&models.Session{})

	for _, day := range sessionsResp {
		for _, movieSession := range day.Movies {
			var movie models.Movie
			err := database.GetDB().Where("external_id = ?", movieSession.ID).First(&movie).Error
			if err != nil {
				continue
			}

			for _, room := range movieSession.Rooms {
				for _, sess := range room.Sessions {
					dateTime, err := time.Parse(time.RFC3339, sess.Date.LocalDate)
					if err != nil {
						continue
					}

					sessionTypeInfo := ""
					if len(sess.Type) > 0 {
						sessionTypeInfo = sess.Type[0]
					}

					_, err = s.updateOrCreateSession(
						sess.ID,
						&movie,
						uint(cfg.TheaterID),
						room.Name,
						"",
						dateTime,
						sess.Price,
						sess.Enabled,
						sess.HasSeatSelection,
						sess.DefaultSector,
						sess.ExternalURL,
						sessionTypeInfo,
					)
					if err != nil {
						return nil, err
					}
				}
			}
		}
	}

	var sessions []models.Session
	database.GetDB().Order("date_time ASC").Find(&sessions)
	return sessions, nil
}

func (s *Service) GetSessionsByMovie(movieTitle string) ([]models.Session, error) {
	var sessions []models.Session
	database.GetDB().Where("movie_title LIKE ?", "%"+movieTitle+"%").Order("date_time ASC").Find(&sessions)
	return sessions, nil
}

func (s *Service) UpdateCinemas() error {
	resp, err := s.client.GetTheaters()
	if err != nil {
		return fmt.Errorf("error fetching theaters: %w", err)
	}

	for _, item := range resp.Items {
		cinema := &models.Cinema{
			ExternalID:   item.ID,
			Name:         item.Name,
			URLKey:       item.URLKey,
			CityID:       item.CityID,
			CityName:     item.CityName,
			State:        item.State,
			UF:           item.UF,
			Address:      item.Address + ", " + item.Number,
			Neighborhood: item.Neighborhood,
			Partnership:  configs.AppConfig.Partnership,
			TotalRooms:   item.TotalRooms,
		}

		err := database.GetDB().Where("external_id = ?", item.ID).Assign(cinema).FirstOrCreate(cinema).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) GetCinemas() ([]models.Cinema, error) {
	var cinemas []models.Cinema
	database.GetDB().Order("name ASC").Find(&cinemas)
	return cinemas, nil
}

type SessionCheckResult struct {
	Session    models.Session
	OldPrice   float64
	OldEnabled bool
	Changed    bool
	Message    string
}

func (s *Service) CheckSession(session models.Session, monitoring models.Monitoring) (*SessionCheckResult, error) {
	detail, err := s.client.GetSessionDetail(session.ExternalID)
	if err != nil {
		return nil, fmt.Errorf("error fetching session detail: %w", err)
	}

	result := &SessionCheckResult{
		Session:    session,
		OldPrice:   monitoring.LastPrice,
		OldEnabled: monitoring.LastEnabled,
	}

	if detail.Price != monitoring.LastPrice {
		result.Changed = true
		result.Message += fmt.Sprintf("💰 Preço alterado: R$ %.2f → R$ %.2f", monitoring.LastPrice, detail.Price)
	}

	if detail.Enabled != monitoring.LastEnabled {
		if result.Changed {
			result.Message += "\n"
		}
		if detail.Enabled {
			result.Message += "✅ Sessão reativada/disponível"
		} else {
			result.Message += "❌ Sessão indisponível/bloqueada"
		}
		result.Changed = true
	}

	database.GetDB().Model(&monitoring).Updates(map[string]interface{}{
		"last_price":   detail.Price,
		"last_enabled": detail.Enabled,
	})

	if result.Changed {
		result.Message = fmt.Sprintf("🎬 *%s* — %s %s\n%s",
			session.MovieTitle,
			session.Room,
			session.DateTime.Format("15:04"),
			result.Message,
		)
	}

	return result, nil
}

func (s *Service) RefreshDB() error {
	if err := s.UpdateCinemas(); err != nil {
		return err
	}
	_, err := s.LoadMovies()
	return err
}

func (s *Service) StartMonitoring(sessionID uint) (*models.Monitoring, error) {
	var session models.Session
	if err := database.GetDB().First(&session, sessionID).Error; err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	monitoring := &models.Monitoring{
		SessionID:      session.ID,
		LastPrice:      session.Price,
		LastEnabled:    session.Enabled,
		Active:         true,
		NotifyOnChange: true,
	}

	err := database.GetDB().Where("session_id = ?", session.ID).Assign(monitoring).FirstOrCreate(monitoring).Error
	return monitoring, err
}

func (s *Service) StopMonitoring(sessionID uint) error {
	return database.GetDB().Model(&models.Monitoring{}).Where("session_id = ?", sessionID).Update("active", false).Error
}

func (s *Service) GetActiveMonitors() ([]models.Monitoring, error) {
	var monitors []models.Monitoring
	database.GetDB().Where("active = ?", true).Preload("Session").Find(&monitors)
	return monitors, nil
}

func (s *Service) GetSessionByID(id uint) (*models.Session, error) {
	var session models.Session
	if err := database.GetDB().First(&session, id).Error; err != nil {
		return nil, err
	}
	return &session, nil
}
