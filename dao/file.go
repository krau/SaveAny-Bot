package dao

import "github.com/krau/SaveAny-Bot/types"

func AddReceivedFile(receivedFile *types.ReceivedFile) error {
	return db.Create(receivedFile).Error
}

func GetReceivedFileByChatAndMessageID(chatID int64, messageID int) (*types.ReceivedFile, error) {
	var receivedFile types.ReceivedFile
	err := db.Where("chat_id = ? AND message_id = ?", chatID, messageID).First(&receivedFile).Error
	if err != nil {
		return nil, err
	}
	return &receivedFile, nil
}

func UpdateReceivedFile(receivedFile *types.ReceivedFile) error {
	return db.Save(receivedFile).Error
}

func DeleteReceivedFile(receivedFile *types.ReceivedFile) error {
	return db.Delete(receivedFile).Error
}
