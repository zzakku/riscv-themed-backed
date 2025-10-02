package ds

import (
	"database/sql"
	"time"
)

type Program struct {
	ID          uint      `gorm:"primaryKey"`
	Status      string    `gorm:"type:varchar(15);not null"`
	DateCreate  time.Time `gorm:"not null"`
	DateUpdate  time.Time
	DateFinish  sql.NullTime `gorm:"default:null"`
	CreatorID   uint         `gorm:"not null"`
	ModeratorID uint

	InitX1 int `gorm:"default:0"`
	InitX2 int `gorm:"default:0"`

	Creator   Users `gorm:"foreignKey:CreatorID"`
	Moderator Users `gorm:"foreignKey:ModeratorID"`
	//	Commands  []Command // Список команд
	//	NumParams []int     // Список параметров для них
}
