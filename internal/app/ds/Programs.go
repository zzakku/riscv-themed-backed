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
	CreatorID   *uint        `gorm:"not null"`
	ModeratorID *uint

	InitT1 *int64 `gorm:"default:null"`
	InitT2 *int64 `gorm:"default:null"`

	ResT1 *int64 `gorm:"default:null"`
	ResT2 *int64 `gorm:"default:null"`

	Creator   Users `gorm:"foreignKey:CreatorID"`
	Moderator Users `gorm:"foreignKey:ModeratorID"`
}
