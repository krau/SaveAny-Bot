package dao

func SaveReceivedFile(receivedFile *ReceivedFile) (*ReceivedFile, error) {
	record, err := GetReceivedFileByChatAndMessageID(receivedFile.ChatID, receivedFile.MessageID)
	if err == nil {
		receivedFile.ID = record.ID
	}
	db.Save(receivedFile)
	return receivedFile, db.Error
}

func BatchSaveReceivedFiles(receivedFiles []*ReceivedFile) error {
	if len(receivedFiles) == 0 {
		return nil
	}
	for _, file := range receivedFiles {
		record, err := GetReceivedFileByChatAndMessageID(file.ChatID, file.MessageID)
		if err == nil {
			file.ID = record.ID
		}
	}
	return db.Save(receivedFiles).Error
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
