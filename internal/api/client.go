package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var BaseURL = "https://api-content.ingresso.com/v0"

type Client struct {
	httpClient  *http.Client
	Partnership string
	CityID      int
	TheaterID   int
}

func NewClient(partnership string, cityID, theaterID int) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		Partnership: partnership,
		CityID:      cityID,
		TheaterID:   theaterID,
	}
}

func (c *Client) get(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

type TheaterItem struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	URLKey       string `json:"urlKey"`
	CityID       string `json:"cityId"`
	CityName     string `json:"cityName"`
	State        string `json:"state"`
	UF           string `json:"uf"`
	Address      string `json:"address"`
	Number       string `json:"number"`
	Neighborhood string `json:"neighborhood"`
	TotalRooms   int    `json:"totalRooms"`
	Enabled      bool   `json:"enabled"`
}

type TheaterListResponse struct {
	Items []TheaterItem `json:"items"`
	Count int           `json:"count"`
}

func (c *Client) GetTheaters() (*TheaterListResponse, error) {
	url := fmt.Sprintf("%s/theaters/city/%d/partnership/%s", BaseURL, c.CityID, c.Partnership)
	body, err := c.get(url)
	if err != nil {
		return nil, err
	}

	var resp TheaterListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("error decoding theaters: %w", err)
	}

	return &resp, nil
}

type MovieItem struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	OriginalTitle string   `json:"originalTitle"`
	Duration      string   `json:"duration"`
	ContentRating string   `json:"contentRating"`
	Synopsis      string   `json:"synopsis"`
	Cast          string   `json:"cast"`
	Director      string   `json:"director"`
	Distributor   string   `json:"distributor"`
	URLKey        string   `json:"urlKey"`
	IsPlaying     bool     `json:"isPlaying"`
	Genres        []string `json:"genres"`
	ImageFeatured string   `json:"imageFeatured"`
}

type MovieListResponse struct {
	Items []MovieItem `json:"items"`
	Count int         `json:"count"`
}

func (c *Client) GetMovies() (*MovieListResponse, error) {
	url := fmt.Sprintf("%s/events/city/%d/partnership/%s", BaseURL, c.CityID, c.Partnership)
	body, err := c.get(url)
	if err != nil {
		return nil, err
	}

	var resp MovieListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("error decoding movies: %w", err)
	}

	return &resp, nil
}

type SessionDate struct {
	LocalDate string `json:"localDate"`
	IsToday   bool   `json:"isToday"`
	DayOfWeek string `json:"dayOfWeek"`
	Hour      string `json:"hour"`
	Year      string `json:"year"`
}

type SessionType struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Alias string `json:"alias"`
}

type SessionRoom struct {
	Name     string        `json:"name"`
	Sessions []SessionItem `json:"sessions"`
}

type SessionItem struct {
	ID               string        `json:"id"`
	Price            float64       `json:"price"`
	Room             string        `json:"room"`
	Type             []string      `json:"type"`
	Types            []SessionType `json:"types"`
	Date             SessionDate   `json:"date"`
	HasSeatSelection bool          `json:"hasSeatSelection"`
	DefaultSector    string        `json:"defaultSector"`
	Enabled          bool          `json:"enabled"`
	ExternalURL      string        `json:"siteURL"`
}

type MovieSession struct {
	ID            string        `json:"id"`
	Title         string        `json:"title"`
	Duration      string        `json:"duration"`
	URLKey        string        `json:"urlKey"`
	ContentRating string        `json:"contentRating"`
	Genres        []string      `json:"genres"`
	Rooms         []SessionRoom `json:"rooms"`
}

type TheaterSession struct {
	Date          string         `json:"date"`
	DateFormatted string         `json:"dateFormatted"`
	DayOfWeek     string         `json:"dayOfWeek"`
	IsToday       bool           `json:"isToday"`
	Movies        []MovieSession `json:"movies"`
}

func (c *Client) GetSessions(theaterID int) ([]TheaterSession, error) {
	url := fmt.Sprintf("%s/sessions/city/%d/theater/%d/partnership/%s", BaseURL, c.CityID, theaterID, c.Partnership)
	body, err := c.get(url)
	if err != nil {
		return nil, err
	}

	var sessions []TheaterSession
	if err := json.Unmarshal(body, &sessions); err != nil {
		return nil, fmt.Errorf("error decoding sessions: %w", err)
	}

	return sessions, nil
}

type SessionDetail struct {
	ID               string      `json:"id"`
	Price            float64     `json:"price"`
	Room             string      `json:"room"`
	Type             []string    `json:"type"`
	Date             SessionDate `json:"date"`
	HasSeatSelection bool        `json:"hasSeatSelection"`
	DefaultSector    string      `json:"defaultSector"`
	Enabled          bool        `json:"enabled"`
	ExternalURL      string      `json:"siteURL"`
	Theater          struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"theater"`
	Event struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	} `json:"event"`
}

func (c *Client) GetSessionDetail(sessionID string) (*SessionDetail, error) {
	url := fmt.Sprintf("%s/sessions/%s/partnership/%s", BaseURL, sessionID, c.Partnership)
	body, err := c.get(url)
	if err != nil {
		return nil, err
	}

	var detail SessionDetail
	if err := json.Unmarshal(body, &detail); err != nil {
		return nil, fmt.Errorf("error decoding session detail: %w", err)
	}

	return &detail, nil
}
