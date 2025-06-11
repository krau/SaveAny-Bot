package database

func CreateCallbackData(data string) (uint, error) {
	callbackData := CallbackData{
		Data: data,
	}
	err := db.Create(&callbackData).Error
	return callbackData.ID, err
}

func GetCallbackData(id uint) (string, error) {
	var callbackData CallbackData
	err := db.First(&callbackData, id).Error
	return callbackData.Data, err
}

func DeleteCallbackData(id uint) error {
	return db.Unscoped().Where("id = ?", id).Delete(&CallbackData{}).Error
}
