package handler

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"

	"r-vBackend/internal/app/ds"

	"github.com/gin-gonic/gin"
)

// Вспомогательная функция, определяющая по заголовку
// Content-File файла, является ли тот изображением
func isImage(contentType string) bool {
	imageTypes := []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/gif",
		"image/webp",
	}

	for _, t := range imageTypes {
		if contentType == t {
			return true
		}
	}
	return false
}

// Вспомогательная функция, выполняющая валидацию загруженного файла
func validateFileUpload(header *multipart.FileHeader) (int, error) {
	// Окрываем чтение файлового потока
	file, err := header.Open()

	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("не удалось получить файл")
	}

	defer file.Close()

	// Знакомая логика определения типа содержимого...

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("не удалось прочитать файл")
	}

	contentType := http.DetectContentType(buffer)

	if !isImage(contentType) {
		return http.StatusBadRequest, fmt.Errorf("файл должен быть изображением")
	}

	// Позиция чтения файла возвращается к исходной
	_, err = file.Seek(0, 0)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("ошибка при обработке файла")
	}

	return 0, nil
}

func (h *Handler) GetStocksAPI(ctx *gin.Context) {
	var stocks []ds.Stock
	var err error

	stocks, err = h.Repository.GetStocks()

	if err != nil {
		if err.Error() != "массив пустой" {
			h.errorHandler(ctx, http.StatusInternalServerError, err)
			return
		} else {
			stocks = []ds.Stock{}
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   stocks,
	})
}

func (h *Handler) GetStockByIdAPI(ctx *gin.Context) {
	strId := ctx.Param("id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	stock, err := h.Repository.GetStock(id)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	// Т.к. мы получаем подробную информацию об акции, используем расширенный сериализатор
	fullStock := ds.FullStockSerializer{Stock: stock}
	fullStock.Creator = stock.Creator

	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   fullStock,
	})
}

func (h *Handler) AddStockAPI(ctx *gin.Context) {
	// Прочитаем в ОЗУ 2 Мб данных формы
	// Этого должно хватить для текстовых полей и несложных изображений (логотипов)

	err := ctx.Request.ParseMultipartForm(2 << 20)

	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	// Пытаемся получить файл из формы
	// header имеет тип *multipart.FileHeader и является указателем
	// на метаданные файла
	header, err := ctx.FormFile("pic")

	// Флаг, устанавливаемый, когда поле pic сопоставлено с неким файлом
	fileFound := false

	if err != nil {
		// Файла в запросе нет - допустимая ситуация
		if err != http.ErrMissingFile {
			h.errorHandler(ctx, http.StatusBadRequest, err)
			return
		}
	} else {
		fileFound = true
	}

	if fileFound {
		code, err := validateFileUpload(header)

		if err != nil {
			h.errorHandler(ctx, code, err)
			return
		}
	}

	stock := ds.Stock{
		Name:        ctx.Request.FormValue("name"),
		CompanyName: ctx.Request.FormValue("company_name"),
		INN:         ctx.Request.FormValue("inn"),
		CreatorID:   1, // временный хардкод
	}

	if ctx.Request.FormValue("purchase_price") != "" {
		stock.PurchasePrice, err = strconv.ParseUint(ctx.Request.FormValue("purchase_price"), 10, 64)
		if err != nil {
			h.errorHandler(ctx, http.StatusBadRequest, err)
			return
		}
	}

	if ctx.Request.FormValue("sale_price") != "" {
		stock.SalePrice, err = strconv.ParseUint(ctx.Request.FormValue("sale_price"), 10, 64)
		if err != nil {
			h.errorHandler(ctx, http.StatusBadRequest, err)
			return
		}
	}

	if ctx.Request.FormValue("count") != "" {
		stock.Count, err = strconv.ParseUint(ctx.Request.FormValue("count"), 10, 64)
		if err != nil {
			h.errorHandler(ctx, http.StatusBadRequest, err)
			return
		}
	}

	err = h.Repository.AddStock(&stock)

	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	if fileFound {
		if err = h.Repository.AddOrReplaceStockImage(stock.ID, header); err != nil {
			h.errorHandler(ctx, http.StatusInternalServerError, err)
			return
		}

		// К сожалению, данные в stock после заполнения pic устарели

		updatedStock, err := h.Repository.GetStock(int(stock.ID))
		if err != nil {
			h.errorHandler(ctx, http.StatusInternalServerError, err)
			return
		}

		ctx.JSON(http.StatusCreated, gin.H{
			"status":  "success",
			"data":    updatedStock,
			"message": "акция успешно добавлена",
		})
	} else {
		ctx.JSON(http.StatusCreated, gin.H{
			"status":  "success",
			"data":    stock,
			"message": "акция успешно добавлена",
		})
	}
}

func (h *Handler) ModifyStockAPI(ctx *gin.Context) {
	strId := ctx.Param("id")
	id, err := strconv.Atoi(strId)

	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	if err := ctx.Request.ParseMultipartForm(2 << 20); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	// Загружаем старые данные, а поля на новые значения будем менять по ходу
	// Потенциально неоптимально, но так мы точно не опустошим лишние поля
	stock, err := h.Repository.GetStock(int(id))
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	if ctx.Request.PostForm.Has("name") {
		stock.Name = ctx.Request.FormValue("name")
	}
	if ctx.Request.PostForm.Has("company_name") {
		stock.CompanyName = ctx.Request.FormValue("company_name")
	}
	if ctx.Request.PostForm.Has("inn") {
		stock.INN = ctx.Request.FormValue("inn")
	}

	if ctx.Request.FormValue("purchase_price") != "" {
		stock.PurchasePrice, err = strconv.ParseUint(ctx.Request.FormValue("purchase_price"), 10, 64)
		if err != nil {
			h.errorHandler(ctx, http.StatusBadRequest, err)
			return
		}
	}

	if ctx.Request.FormValue("sale_price") != "" {
		stock.SalePrice, err = strconv.ParseUint(ctx.Request.FormValue("sale_price"), 10, 64)
		if err != nil {
			h.errorHandler(ctx, http.StatusBadRequest, err)
			return
		}
	}

	if ctx.Request.FormValue("count") != "" {
		stock.Count, err = strconv.ParseUint(ctx.Request.FormValue("count"), 10, 64)
		if err != nil {
			h.errorHandler(ctx, http.StatusBadRequest, err)
			return
		}
	}

	header, err := ctx.FormFile("pic")

	var fileFound bool

	if err != nil {
		// Файла в запросе нет - допустимая ситуация
		if err != http.ErrMissingFile {
			h.errorHandler(ctx, http.StatusBadRequest, err)
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "File error: " + err.Error()})
			return
		}
	} else {
		fileFound = true
	}

	if fileFound {
		code, err := validateFileUpload(header)

		if err != nil {
			h.errorHandler(ctx, code, err)
			return
		}
	}

	if err = h.Repository.ModifyStock(uint(id), &stock); err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	if fileFound {
		if err = h.Repository.AddOrReplaceStockImage(stock.ID, header); err != nil {
			h.errorHandler(ctx, http.StatusInternalServerError, err)
			return
		}
	}

	updatedStock, err := h.Repository.GetStock(int(id))
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"data":    updatedStock,
		"message": "запись успешно обновлена",
	})
}

func (h *Handler) DeleteStockAPI(ctx *gin.Context) {
	strId := ctx.Param("id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	err = h.Repository.DeleteStock(uint(id))
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "команда успешно удалена",
	})
}

func (h *Handler) GetUserStocksAPI(ctx *gin.Context) {
	strId := ctx.Param("id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	userStocks, err := h.Repository.GetUserStocks(id)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   userStocks,
	})
}
