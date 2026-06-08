package monitor

import (
	"fmt"

	"github.com/opencode/movie-watcher/database"
	"github.com/opencode/movie-watcher/internal/cinema"
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

func (r *Runner) CheckAll() error {
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
			database.GetDB().Model(&m).Update("active", false)
			continue
		}

		result, err := r.service.CheckSession(session, m)
		if err != nil {
			fmt.Printf("Error checking session %d: %v\n", session.ID, err)
			continue
		}

		fmt.Printf("Checked: %s - %s %s\n",
			session.MovieTitle,
			session.Room,
			session.DateTime.Format("15:04"),
		)

		if result.Changed {
			fmt.Println("  Change detected!")
			fmt.Println("  " + result.Message)

			if r.notifier.IsEnabled() {
				err := r.notifier.SendMessage(result.Message)
				if err != nil {
					fmt.Printf("Error sending notification: %v\n", err)
				}
			}
		}
	}

	return nil
}

func (r *Runner) Start(sessionID uint) error {
	monitor, err := r.service.StartMonitoring(sessionID)
	if err != nil {
		return fmt.Errorf("error starting monitor: %w", err)
	}

	session, _ := r.service.GetSessionByID(sessionID)

	msg := fmt.Sprintf("🎬 Monitoramento iniciado para *%s* — %s %s\n💰 Preço atual: R$ %.2f",
		session.MovieTitle,
		session.Room,
		session.DateTime.Format("15:04"),
		session.Price,
	)

	fmt.Println(msg)

	if r.notifier.IsEnabled() {
		r.notifier.SendMessage(msg)
	}

	_ = monitor
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
		fmt.Printf("  [%d] %s — %s %s | R$ %.2f\n",
			session.ID,
			session.MovieTitle,
			session.Room,
			session.DateTime.Format("15:04"),
			m.LastPrice,
		)
	}

	return nil
}


