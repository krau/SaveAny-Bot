package dao

func SaveReceivedFile(receivedFile *ReceivedFile) (*ReceivedFile, error) {
	record, err := GetReceivedFileByChatAndMessageID(receivedFile.ChatID, receivedFile.MessageID)
	if err == nil {
		receivedFile.ID = record.ID
	}
	db.Save(receivedFile)
	return receivedFile, db.Error
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
