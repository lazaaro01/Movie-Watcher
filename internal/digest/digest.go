package digest

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/opencode/movie-watcher/configs"
	"github.com/opencode/movie-watcher/internal/cinema"
	"github.com/opencode/movie-watcher/internal/lock"
	"github.com/opencode/movie-watcher/internal/models"
	"github.com/opencode/movie-watcher/internal/notifier"
)

// maxSessionsPerMovie caps how many showtimes/buttons go into a single
// movie's message — purely a readability choice, well under Telegram's
// actual message/button limits.
const maxSessionsPerMovie = 8

type Runner struct {
	service  *cinema.Service
	notifier *notifier.Notifier
}

func NewRunner() *Runner {
	return &Runner{
		service:  cinema.NewService(),
		notifier: notifier.New(),
	}
}

type priceSnapshot struct {
	previousPrice float64
	hadPrevious   bool
}

// Run sends one Telegram message per movie playing today in the resolved
// cinemas, with a "comprar" button per showtime linking straight to the
// session's purchase page.
//
// Price history is recorded for every still-upcoming session (not just
// today's), so that by the time a session becomes "today", there's already
// a prior observation to compare against and flag a real price drop.
func (r *Runner) Run() error {
	lockPath := configs.AppConfig.Database.Path + ".digest.lock"
	release, err := lock.AcquireLock(lockPath)
	if err != nil {
		return err
	}
	defer release()

	theaterIDs, err := r.service.ResolveTheaterIDs()
	if err != nil {
		return fmt.Errorf("error resolving cinemas: %w", err)
	}

	// Movies must be loaded before sessions: LoadSessions only keeps a
	// session if it can already find the matching Movie row locally, so on
	// a fresh database this order avoids silently dropping every session.
	if _, err := r.service.LoadMovies(); err != nil {
		return fmt.Errorf("error loading movies: %w", err)
	}
	if _, err := r.service.LoadSessionsForTheaters(theaterIDs); err != nil {
		return fmt.Errorf("error loading sessions: %w", err)
	}

	allSessions, err := r.service.GetActiveSessions(theaterIDs)
	if err != nil {
		return fmt.Errorf("error fetching active sessions: %w", err)
	}
	if len(allSessions) == 0 {
		fmt.Println("Nenhuma sessão ativa encontrada.")
		return nil
	}

	today := time.Now().Format("2006-01-02")
	snapshots := make(map[uint]priceSnapshot, len(allSessions))
	var todaysSessions []models.Session

	for _, sess := range allSessions {
		previousPrice, hadPrevious, err := r.service.RecordPriceSnapshot(sess.ID, sess.Price, sess.Enabled)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao registrar preço da sessão %d: %v\n", sess.ID, err)
			continue
		}
		snapshots[sess.ID] = priceSnapshot{previousPrice: previousPrice, hadPrevious: hadPrevious}

		if sess.DateTime.Format("2006-01-02") == today {
			todaysSessions = append(todaysSessions, sess)
		}
	}

	if len(todaysSessions) == 0 {
		fmt.Println("Nenhuma sessão disponível hoje.")
		return nil
	}

	cinemaNames := r.service.GetCinemaNamesByTheaterIDs(theaterIDs)
	showCinemaLabel := len(cinemaNames) > 1

	order := make([]uint, 0)
	groups := make(map[uint][]models.Session)
	for _, sess := range todaysSessions {
		if _, ok := groups[sess.MovieID]; !ok {
			order = append(order, sess.MovieID)
		}
		groups[sess.MovieID] = append(groups[sess.MovieID], sess)
	}

	for _, movieID := range order {
		r.sendMovieDigest(movieID, groups[movieID], snapshots, cinemaNames, showCinemaLabel)
	}

	return nil
}

func (r *Runner) sendMovieDigest(movieID uint, sessions []models.Session, snapshots map[uint]priceSnapshot, cinemaNames map[int]string, showCinemaLabel bool) {
	movie, err := r.service.GetMovieByID(movieID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao buscar filme %d: %v\n", movieID, err)
		return
	}

	rating := movie.ContentRating
	if rating == "" {
		rating = "Livre"
	}

	var body strings.Builder
	fmt.Fprintf(&body, "🎬 *%s*\n%s · %d min · %s\n\n", movie.Title, movie.Genre, movie.Duration, rating)

	var buttons []notifier.Button
	shown, skippedDisabled := 0, 0

	for _, sess := range sessions {
		if !sess.Enabled {
			skippedDisabled++
			continue
		}
		if shown >= maxSessionsPerMovie {
			break
		}

		snap := snapshots[sess.ID]
		isPromo := snap.hadPrevious && sess.Price < snap.previousPrice

		timeLabel := sess.DateTime.Format("15:04")
		lineLabel := timeLabel
		if showCinemaLabel {
			if name := cinemaNames[int(sess.TheaterID)]; name != "" {
				lineLabel = name + " " + timeLabel
			}
		}

		promoTag := ""
		if isPromo {
			promoTag = fmt.Sprintf(" 🔻 promoção (antes R$ %.2f)", snap.previousPrice)
		}
		fmt.Fprintf(&body, "🕒 %s — R$ %.2f%s\n", lineLabel, sess.Price, promoTag)

		if sess.ExternalURL != "" {
			buttonLabel := timeLabel
			if isPromo {
				buttonLabel += " 🔻"
			}
			buttons = append(buttons, notifier.Button{Label: buttonLabel, URL: sess.ExternalURL})
		}
		shown++
	}

	if shown == 0 {
		return
	}

	remaining := len(sessions) - shown - skippedDisabled
	if remaining > 0 {
		fmt.Fprintf(&body, "\n+%d sessões — use 'sessions list \"%s\"'\n", remaining, movie.Title)
	}

	fmt.Print(body.String())

	if err := r.notifier.SendMessageWithButtons(body.String(), buttons); err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao enviar digest de %s: %v\n", movie.Title, err)
	}
}
