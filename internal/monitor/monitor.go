package monitor

import (
	"fmt"
	"os"
	"strings"

	"github.com/opencode/movie-watcher/configs"
	"github.com/opencode/movie-watcher/database"
	"github.com/opencode/movie-watcher/internal/cinema"
	"github.com/opencode/movie-watcher/internal/lock"
	"github.com/opencode/movie-watcher/internal/models"
	"github.com/opencode/movie-watcher/internal/notifier"
)

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

func (r *Runner) cinemaLabel(theaterID uint) string {
	c, err := r.service.GetCinemaByTheaterID(theaterID)
	if err != nil {
		return ""
	}
	return c.Name + " — "
}

func (r *Runner) CheckAll() error {
	lockPath := configs.AppConfig.Database.Path + ".lock"
	release, err := lock.AcquireLock(lockPath)
	if err != nil {
		return err
	}
	defer release()

	monitors, err := r.service.GetActiveMonitors()
	if err != nil {
		return fmt.Errorf("error getting active monitors: %w", err)
	}

	if len(monitors) == 0 {
		fmt.Println("No active monitors found.")
		return nil
	}

	for _, m := range monitors {
		var session models.Session
		if err := database.GetDB().First(&session, m.SessionID).Error; err != nil {
			fmt.Printf("Session %d not found, deactivating monitor\n", m.SessionID)
			if err := database.GetDB().Model(&m).Update("active", false).Error; err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao desativar monitor da sessão %d: %v\n", m.SessionID, err)
			}
			continue
		}

		result, err := r.service.CheckSession(session, m)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error checking session %d: %v\n", session.ID, err)
			continue
		}

		fmt.Printf("Checked: %s%s - %s %s\n",
			r.cinemaLabel(session.TheaterID),
			session.MovieTitle,
			session.Room,
			session.DateTime.Format("15:04"),
		)

		if result.Changed {
			fmt.Println("  Change detected!")
			fmt.Println("  " + result.Message)

			if r.notifier.IsEnabled() {
				if err := r.notifier.SendMessage(result.Message); err != nil {
					fmt.Fprintf(os.Stderr, "Error sending notification: %v\n", err)
				}
			}
		}
	}

	return nil
}

func (r *Runner) Start(sessionID uint) error {
	if _, err := r.service.StartMonitoring(sessionID); err != nil {
		return fmt.Errorf("error starting monitor: %w", err)
	}

	session, _ := r.service.GetSessionByID(sessionID)

	msg := fmt.Sprintf("🎬 Monitoramento iniciado para *%s* — %s%s %s\n💰 Preço atual: R$ %.2f",
		session.MovieTitle,
		r.cinemaLabel(session.TheaterID),
		session.Room,
		session.DateTime.Format("15:04"),
		session.Price,
	)

	fmt.Println(msg)

	if r.notifier.IsEnabled() {
		if err := r.notifier.SendMessage(msg); err != nil {
			fmt.Fprintf(os.Stderr, "Error sending notification: %v\n", err)
		}
	}

	return nil
}

// StartMovie loads sessions for the resolved cinemas and starts monitoring
// every session found for the given movie title, sending a single summary
// notification instead of one per session.
func (r *Runner) StartMovie(movieTitle string) error {
	theaterIDs, err := r.service.ResolveTheaterIDs()
	if err != nil {
		return fmt.Errorf("error resolving cinemas: %w", err)
	}

	if _, err := r.service.LoadSessionsForTheaters(theaterIDs); err != nil {
		return fmt.Errorf("error loading sessions: %w", err)
	}

	sessions, err := r.service.GetSessionsByMovie(movieTitle, theaterIDs...)
	if err != nil {
		return fmt.Errorf("error fetching sessions: %w", err)
	}
	if len(sessions) == 0 {
		return fmt.Errorf("nenhuma sessão encontrada para %q", movieTitle)
	}

	var summary strings.Builder
	fmt.Fprintf(&summary, "🎬 Monitoramento iniciado para *%s* (%d sessões)\n", movieTitle, len(sessions))

	started := 0
	for _, session := range sessions {
		if _, err := r.service.StartMonitoring(session.ID); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao monitorar sessão %d: %v\n", session.ID, err)
			continue
		}
		started++
		line := fmt.Sprintf("  • %s%s — %s — R$ %.2f\n", r.cinemaLabel(session.TheaterID), session.Room, session.DateTime.Format("02/01 15:04"), session.Price)
		fmt.Print(line)
		summary.WriteString(line)
	}

	if started == 0 {
		return fmt.Errorf("não foi possível iniciar nenhum monitoramento para %q", movieTitle)
	}

	if r.notifier.IsEnabled() {
		if err := r.notifier.SendMessage(summary.String()); err != nil {
			fmt.Fprintf(os.Stderr, "Error sending notification: %v\n", err)
		}
	}

	return nil
}

func (r *Runner) Stop(sessionID uint) error {
	err := r.service.StopMonitoring(sessionID)
	if err != nil {
		return fmt.Errorf("error stopping monitor: %w", err)
	}

	fmt.Printf("Monitoramento desativado para sessão %d\n", sessionID)
	return nil
}

func (r *Runner) Status() error {
	monitors, err := r.service.GetActiveMonitors()
	if err != nil {
		return fmt.Errorf("error getting monitors: %w", err)
	}

	if len(monitors) == 0 {
		fmt.Println("Nenhum monitoramento ativo.")
		return nil
	}

	fmt.Println("Monitores ativos:")
	for _, m := range monitors {
		var session models.Session
		if err := database.GetDB().First(&session, m.SessionID).Error; err != nil {
			continue
		}
		fmt.Printf("  [%d] %s%s — %s %s | R$ %.2f\n",
			session.ID,
			r.cinemaLabel(session.TheaterID),
			session.MovieTitle,
			session.Room,
			session.DateTime.Format("15:04"),
			m.LastPrice,
		)
	}

	return nil
}
