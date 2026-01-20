package repository

import (
	"r-vBackend/internal/app/ds"
)

func (r *Repository) GetUserStocks(id int) (*ds.UserStocks, error) {

	var userStocks ds.UserStocks

	err := r.db.Where("id = ?", id).First(&userStocks.User).Error

	if err != nil {
		return nil, err
	}

	err = r.db.Where("creator_id = ?", id).Find(&userStocks.Stocks).Error

	if err != nil {
		return nil, err
	}

	return &userStocks, nil
}
