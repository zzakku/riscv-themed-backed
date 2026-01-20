package ds

type Stock struct {
	ID            uint   `gorm:"primaryKey" json:"id"`
	Name          string `json:"name"`
	PurchasePrice uint64 `json:"purchase_price"`
	SalePrice     uint64 `json:"sale_price"`
	Count         uint64 `json:"count"`
	CompanyName   string `json:"company_name"`
	INN           string `json:"inn"`
	Pic           string `json:"pic"`
	CreatorID     uint   `json:"-"`
	Creator       User   `gorm:"foreignKey:CreatorID" json:"-"`
}

type FullStockSerializer struct {
	Stock
	Creator User `json:"creator"`
}
