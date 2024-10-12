package dao

import "github.com/krau/SaveAny-Bot/model"

func AddReceivedFile(receivedFile *model.ReceivedFile) error {
	return db.Create(receivedFile).Error
}

func GetReceivedFileByChatAndMessageID(chatID int64, messageID int32) (*model.ReceivedFile, error) {
	var receivedFile model.ReceivedFile
	err := db.Where("chat_id = ? AND message_id = ?", chatID, messageID).First(&receivedFile).Error
	if err != nil {
		return nil, err
	}
	return &receivedFile, nil
}

func UpdateReceivedFile(receivedFile *model.ReceivedFile) error {
	return db.Save(receivedFile).Error
}

func DeleteReceivedFile(receivedFile *model.ReceivedFile) error {
	return db.Delete(receivedFile).Error
}
