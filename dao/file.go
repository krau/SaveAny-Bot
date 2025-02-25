package dao

func SaveReceivedFile(receivedFile *ReceivedFile) error {
	record, err := GetReceivedFileByChatAndMessageID(receivedFile.ChatID, receivedFile.MessageID)
	if err == nil {
		receivedFile.ID = record.ID
	}
	return db.Save(receivedFile).Error
}

func GetReceivedFileByChatAndMessageID(chatID int64, messageID int) (*ReceivedFile, error) {
	var receivedFile ReceivedFile
	err := db.Where("chat_id = ? AND message_id = ?", chatID, messageID).First(&receivedFile).Error
	if err != nil {
		return nil, err
	}
	return &receivedFile, nil
}

func DeleteReceivedFile(receivedFile *ReceivedFile) error {
	return db.Unscoped().Delete(receivedFile).Error
}
