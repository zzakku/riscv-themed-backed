package ds

type Command struct {
	ID          int    `gorm:"primaryKey"`
	IsDelete    bool   `gorm:"type:boolean not null;default:false"`
	ComName     string `gorm:"type:varchar(70);not null"`
	Fmt         string `gorm:"type:varchar(15);not null"`
	RsNum       int
	RdNum       int
	Description string `gorm:"type:varchar(200)"`
	Img         string `gorm:"type:varchar(100)"`
}

// type Command struct {
// 	ID          int    `gorm:"primaryKey"`
// 	IsDelete    bool   `gorm:"type:boolean not null;default:false"`
// 	Img         string `gorm:"type:varchar(100)"`
// 	Name        string `gorm:"type:varchar(25);not null"`
// 	Info        string `gorm:"type:varchar(100)"`
// 	Nickname    string `gorm:"type:varchar(15);not null"`
// 	Friends     int
// 	Subscribers int
// }
