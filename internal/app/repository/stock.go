package repository

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"r-vBackend/internal/app/ds"
	"strconv"

	"github.com/minio/minio-go"
)

func (r *Repository) GetStocks() ([]ds.Stock, error) {
	var stocks []ds.Stock
	err := r.db.Find(&stocks).Error
	// обязательно проверяем ошибки, и если они появились - передаем выше, то есть хендлеру
	if err != nil {
		return nil, err
	}
	if len(stocks) == 0 {
		return nil, fmt.Errorf("массив пустой")
	}

	return stocks, nil
}

func (r *Repository) AddStock(stock *ds.Stock) error {
	err := r.db.Model(&ds.Stock{}).Create(stock).Error
	if err != nil {
		return fmt.Errorf("ошибка при добавлении акции: %w", err)
	}

	return nil
}

func (r *Repository) GetStock(id int) (ds.Stock, error) {
	stock := ds.Stock{}
	err := r.db.Preload("Creator").Where("id = ?", id).First(&stock).Error
	if err != nil {
		return ds.Stock{}, err
	}
	return stock, nil
}

func (r *Repository) DeleteStock(stockID uint) error {
	// Обратите внимание: жёсткое удаление, не логическое
	err := r.db.Delete(&ds.Stock{}, stockID).Error
	if err != nil {
		return fmt.Errorf("ошибка при удалении команды с id %d: %w", stockID, err)
	}

	return nil
}

func (r *Repository) ModifyStock(id uint, stock *ds.Stock) error {

	var old_stock ds.Stock

	err := r.db.Model(&ds.Stock{}).Where("id = ?", id).First(&old_stock).Error

	if err != nil {
		return fmt.Errorf("не удалось найти команду с id %d: %w", id, err)
	}

	// Попытки изменения ID акции нужно предотвращать
	// Создаем структуру для обновления без ID
	updateData := map[string]interface{}{
		"name":           stock.Name,
		"purchase_price": stock.PurchasePrice,
		"sale_price":     stock.SalePrice,
		"count":          stock.Count,
		"company_name":   stock.CompanyName,
		"inn":            stock.INN,
	}

	err = r.db.Model(&ds.Stock{}).Where("id = ?", id).Updates(&updateData).Error

	if err != nil {
		return fmt.Errorf("ошибка при обновлении акции с id %d: %w", id, err)
	}

	return nil
}

func (r *Repository) AddOrReplaceStockImage(stockID uint, header *multipart.FileHeader) error {
	// Название будущего файла - <номер акции>.png

	filename := strconv.FormatUint(uint64(stockID), 10) + ".png"

	// Расширение .png будет даже у картинок, исходно его не имевших.
	// Это не совсем хорошо, но отображаться всё будет.

	// Открываем файл
	file, err := header.Open()
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %w", err)
	}
	// Как бы мы ни вышли из этой функции, файл обязательно будет закрыт
	defer file.Close()

	// header уже содержит заголовок с типом файла, но для пущей уверенности
	// мы получим Content-Type на базе реального содержимого файла

	// Тип полученного файла хранится в его первых 512 байтах
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)

	if err != nil {
		return fmt.Errorf("ошибка чтения файла: %w", err)
	}

	// Определяем тип файла по его содержимому
	contentType := http.DetectContentType(buffer)

	// Позиция чтения файла возвращается к исходной
	_, err = file.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("ошибка перемещения по файловому потоку: %w", err)
	}

	_, err = r.minio.PutObject(
		r.minio_bucket_name,
		filename,
		file,
		header.Size,
		minio.PutObjectOptions{
			ContentType: contentType,
		})

	if err != nil {
		return fmt.Errorf("не удалось добавить объект в хранилище minio: %w", err)
	}

	// Эндпоинт здесь должен совпадать с эндпоинтом MinIO в конфигурации
	err = r.db.Model(&ds.Stock{}).Where("id = ?", stockID).UpdateColumn("pic", "http://127.0.0.1:9000/"+r.minio_bucket_name+"/"+filename).Error
	if err != nil {
		// Если не удалось сохранить в БД, удаляем из MinIO
		r.minio.RemoveObject(r.minio_bucket_name, filename)
		return fmt.Errorf("ошибка сохранения пути к изображению: %w", err)
	}

	return nil
}
