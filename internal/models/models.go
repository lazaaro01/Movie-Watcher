package models

import (
	"time"

	"gorm.io/gorm"
)

type Cinema struct {
	gorm.Model
	ExternalID   string
	Name         string
	URLKey       string
	CityID       string
	CityName     string
	State        string
	UF           string
	Address      string
	Neighborhood string
	Partnership  string
	TotalRooms   int
	Monitored    bool
}

type Movie struct {
	gorm.Model
	ExternalID    string
	Title         string
	OriginalTitle string
	Duration      int
	ContentRating string
	Synopsis      string
	Cast          string
	Director      string
	Distributor   string
	Genre         string
	URLKey        string
	ImageURL      string
	IsPlaying     bool
}

type Session struct {
	gorm.Model
	ExternalID       string
	MovieID          uint
	MovieTitle       string
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

type Monitoring struct {
	gorm.Model
	SessionID      uint    `gorm:"index"`
	Session        Session `gorm:"foreignKey:SessionID"`
	LastPrice      float64
	LastEnabled    bool
	Active         bool
	NotifyOnChange bool
}

type PriceHistory struct {
	gorm.Model
	SessionID uint `gorm:"index"`
	Price     float64
	Enabled   bool
}
