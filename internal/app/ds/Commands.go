package ds

type Command struct {
	ID          int    `gorm:"primaryKey" json:"id"`
	IsDelete    bool   `gorm:"type:boolean not null;default:false" json:"is_delete"`
	ComName     string `gorm:"type:varchar(70);not null" json:"com_name"`
	Fmt         string `gorm:"type:varchar(15);not null" json:"fmt"`
	RsNum       int    `json:"rs_num"`
	RdNum       int    `json:"rd_num"`
	Description string `gorm:"type:varchar(200)" json:"description"`
	Img         string `gorm:"type:varchar(100)" json:"img"`
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
