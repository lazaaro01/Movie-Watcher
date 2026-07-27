package cinema

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/opencode/movie-watcher/configs"
	"github.com/opencode/movie-watcher/database"
	"github.com/opencode/movie-watcher/internal/api"
	"github.com/opencode/movie-watcher/internal/models"
	"gorm.io/gorm"
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

// sessionInput groups the fields needed to upsert a session, avoiding a
// long positional-argument list that's easy to get wrong at the call site.
type sessionInput struct {
	ExternalID       string
	Movie            *models.Movie
	TheaterID        uint
	Room             string
	RoomID           string
	DateTime         time.Time
	Price            float64
	Enabled          bool
	HasSeatSelection bool
	DefaultSector    string
	ExternalURL      string
	SessionType      string
}

func (s *Service) updateOrCreateSession(in sessionInput) (*models.Session, error) {
	session := &models.Session{
		ExternalID:       in.ExternalID,
		MovieID:          in.Movie.ID,
		MovieTitle:       in.Movie.Title,
		TheaterID:        in.TheaterID,
		Room:             in.Room,
		RoomID:           in.RoomID,
		DateTime:         in.DateTime,
		Price:            in.Price,
		Enabled:          in.Enabled,
		HasSeatSelection: in.HasSeatSelection,
		DefaultSector:    in.DefaultSector,
		ExternalURL:      in.ExternalURL,
		SessionType:      in.SessionType,
	}

	err := database.GetDB().Where("external_id = ?", in.ExternalID).Assign(session).FirstOrCreate(session).Error
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

func (s *Service) GetMovieByID(id uint) (*models.Movie, error) {
	var movie models.Movie
	if err := database.GetDB().First(&movie, id).Error; err != nil {
		return nil, err
	}
	return &movie, nil
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

// LoadSessions refreshes sessions for a single theater. It only touches rows
// belonging to that theater: sessions still reported by the API are updated
// in place (preserving their internal ID, which monitors and history rely
// on), and only sessions that disappeared from this theater's feed are
// removed.
func (s *Service) LoadSessions(theaterID int) ([]models.Session, error) {
	sessionsResp, err := s.client.GetSessions(theaterID)
	if err != nil {
		return nil, fmt.Errorf("error fetching sessions: %w", err)
	}

	seenExternalIDs := make([]string, 0)

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

					_, err = s.updateOrCreateSession(sessionInput{
						ExternalID:       sess.ID,
						Movie:            &movie,
						TheaterID:        uint(theaterID),
						Room:             room.Name,
						DateTime:         dateTime,
						Price:            sess.Price,
						Enabled:          sess.Enabled,
						HasSeatSelection: sess.HasSeatSelection,
						DefaultSector:    sess.DefaultSector,
						ExternalURL:      sess.ExternalURL,
						SessionType:      sessionTypeInfo,
					})
					if err != nil {
						return nil, err
					}

					seenExternalIDs = append(seenExternalIDs, sess.ID)
				}
			}
		}
	}

	staleQuery := database.GetDB().Where("theater_id = ?", theaterID)
	if len(seenExternalIDs) > 0 {
		staleQuery = staleQuery.Where("external_id NOT IN ?", seenExternalIDs)
	}
	if err := staleQuery.Delete(&models.Session{}).Error; err != nil {
		return nil, fmt.Errorf("error removing stale sessions: %w", err)
	}

	var sessions []models.Session
	database.GetDB().Where("theater_id = ?", theaterID).Order("date_time ASC").Find(&sessions)
	return sessions, nil
}

// LoadSessionsForTheaters refreshes sessions for every theater ID given and
// returns the combined, up-to-date list.
func (s *Service) LoadSessionsForTheaters(theaterIDs []int) ([]models.Session, error) {
	for _, id := range theaterIDs {
		if _, err := s.LoadSessions(id); err != nil {
			return nil, err
		}
	}

	var sessions []models.Session
	database.GetDB().Where("theater_id IN ?", theaterIDs).Order("date_time ASC").Find(&sessions)
	return sessions, nil
}

// GetSessionsByMovie returns sessions matching the given title, optionally
// restricted to a set of theater IDs.
func (s *Service) GetSessionsByMovie(movieTitle string, theaterIDs ...int) ([]models.Session, error) {
	query := database.GetDB().Where("movie_title LIKE ?", "%"+movieTitle+"%")
	if len(theaterIDs) > 0 {
		query = query.Where("theater_id IN ?", theaterIDs)
	}

	var sessions []models.Session
	query.Order("date_time ASC").Find(&sessions)
	return sessions, nil
}

// GetActiveSessions returns sessions for the given theaters that haven't
// happened yet (today or later), ordered by date/time. Used both to build
// the daily digest and to keep price history flowing for sessions before
// they're actually shown to the user.
func (s *Service) GetActiveSessions(theaterIDs []int) ([]models.Session, error) {
	var sessions []models.Session
	if err := database.GetDB().Where("theater_id IN ?", theaterIDs).
		Order("date_time ASC, movie_title ASC").Find(&sessions).Error; err != nil {
		return nil, err
	}

	now := time.Now()
	result := make([]models.Session, 0, len(sessions))
	for _, sess := range sessions {
		if sess.DateTime.After(now) {
			result = append(result, sess)
		}
	}
	return result, nil
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

func (s *Service) SearchCinemas(query string) ([]models.Cinema, error) {
	var cinemas []models.Cinema
	database.GetDB().Where("name LIKE ?", "%"+query+"%").Order("name ASC").Find(&cinemas)
	return cinemas, nil
}

func (s *Service) GetMonitoredCinemas() ([]models.Cinema, error) {
	var cinemas []models.Cinema
	database.GetDB().Where("monitored = ?", true).Order("name ASC").Find(&cinemas)
	return cinemas, nil
}

// SetCinemaMonitored marks (or unmarks) a cinema, identified by its internal
// ID, as one whose movies/sessions should be pulled in by default.
func (s *Service) SetCinemaMonitored(id uint, monitored bool) error {
	result := database.GetDB().Model(&models.Cinema{}).Where("id = ?", id).Update("monitored", monitored)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("cinema %d não encontrado", id)
	}
	return nil
}

// GetCinemaTheaterID resolves a cinema's internal ID to the external
// theater ID used by the Ingresso.com API.
func (s *Service) GetCinemaTheaterID(cinemaID uint) (int, error) {
	var c models.Cinema
	if err := database.GetDB().First(&c, cinemaID).Error; err != nil {
		return 0, fmt.Errorf("cinema %d não encontrado: %w", cinemaID, err)
	}
	theaterID, err := strconv.Atoi(c.ExternalID)
	if err != nil {
		return 0, fmt.Errorf("cinema %d tem external_id inválido: %w", cinemaID, err)
	}
	return theaterID, nil
}

func (s *Service) GetCinemaByTheaterID(theaterID uint) (*models.Cinema, error) {
	var c models.Cinema
	err := database.GetDB().Where("external_id = ?", strconv.Itoa(int(theaterID))).First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// GetCinemaNamesByTheaterIDs returns a best-effort map of theater ID to
// cinema name, skipping any theater that hasn't been synced yet.
func (s *Service) GetCinemaNamesByTheaterIDs(theaterIDs []int) map[int]string {
	names := make(map[int]string, len(theaterIDs))
	for _, id := range theaterIDs {
		var c models.Cinema
		if err := database.GetDB().Where("external_id = ?", strconv.Itoa(id)).First(&c).Error; err != nil {
			continue
		}
		names[id] = c.Name
	}
	return names
}

// ResolveTheaterIDs returns the theater IDs to operate on: every cinema
// explicitly marked as monitored, or the config's default theater as a
// fallback when none has been marked yet.
func (s *Service) ResolveTheaterIDs() ([]int, error) {
	cinemas, err := s.GetMonitoredCinemas()
	if err != nil {
		return nil, err
	}

	ids := make([]int, 0, len(cinemas))
	for _, c := range cinemas {
		id, err := strconv.Atoi(c.ExternalID)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return []int{configs.AppConfig.TheaterID}, nil
	}
	return ids, nil
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

	if err := database.GetDB().Model(&monitoring).Updates(map[string]interface{}{
		"last_price":   detail.Price,
		"last_enabled": detail.Enabled,
	}).Error; err != nil {
		return nil, fmt.Errorf("error updating monitoring state: %w", err)
	}

	if err := database.GetDB().Create(&models.PriceHistory{
		SessionID: session.ID,
		Price:     detail.Price,
		Enabled:   detail.Enabled,
	}).Error; err != nil {
		return nil, fmt.Errorf("error recording price history: %w", err)
	}

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

	if err := database.GetDB().Where("session_id = ?", session.ID).Assign(monitoring).FirstOrCreate(monitoring).Error; err != nil {
		return nil, err
	}

	if err := database.GetDB().Create(&models.PriceHistory{
		SessionID: session.ID,
		Price:     session.Price,
		Enabled:   session.Enabled,
	}).Error; err != nil {
		return nil, fmt.Errorf("error recording price history: %w", err)
	}

	return monitoring, nil
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

func (s *Service) GetPriceHistory(sessionID uint) ([]models.PriceHistory, error) {
	var history []models.PriceHistory
	database.GetDB().Where("session_id = ?", sessionID).Order("created_at ASC").Find(&history)
	return history, nil
}

// RecordPriceSnapshot inserts a PriceHistory row for sessionID and reports
// the previously recorded price, if any. Unlike StartMonitoring/
// CheckSession, this doesn't require an active Monitoring row — it's used
// to track price movement for any session (e.g. from the daily digest)
// independent of whether the user asked to be notified about it.
func (s *Service) RecordPriceSnapshot(sessionID uint, price float64, enabled bool) (previousPrice float64, hadPrevious bool, err error) {
	var last models.PriceHistory
	lookupErr := database.GetDB().Where("session_id = ?", sessionID).
		Order("created_at DESC").First(&last).Error

	switch {
	case lookupErr == nil:
		previousPrice, hadPrevious = last.Price, true
	case errors.Is(lookupErr, gorm.ErrRecordNotFound):
		// first time seeing this session — no previous price, not an error
	default:
		return 0, false, fmt.Errorf("error reading price history: %w", lookupErr)
	}

	if err := database.GetDB().Create(&models.PriceHistory{
		SessionID: sessionID,
		Price:     price,
		Enabled:   enabled,
	}).Error; err != nil {
		return previousPrice, hadPrevious, fmt.Errorf("error recording price history: %w", err)
	}

	return previousPrice, hadPrevious, nil
}
