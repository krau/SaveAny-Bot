package dao

import "github.com/krau/SaveAny-Bot/model"

func AddReceivedFile(receivedFile *model.ReceivedFile) error {
	return db.Create(receivedFile).Error
}

func GetReceivedFileByFileID(fileID string) (*model.ReceivedFile, error) {
	var receivedFile model.ReceivedFile
	err := db.Where("file_id = ?", fileID).First(&receivedFile).Error
	if err != nil {
		return nil, err
	}
	return &receivedFile, nil
}

func GetReceivedFileByFileUniqueID(fileUniqueID string) (*model.ReceivedFile, error) {
	var receivedFile model.ReceivedFile
	err := db.Where("file_unique_id = ?", fileUniqueID).First(&receivedFile).Error
	if err != nil {
		return nil, err
	}
	return &receivedFile, nil
}

func GetReceivedFileByChatAndMessageID(chatID int64, messageID int) (*model.ReceivedFile, error) {
	var receivedFile model.ReceivedFile
	err := db.Where("chat_id = ? AND message_id = ?", chatID, messageID).First(&receivedFile).Error
	if err != nil {
		return nil, err
	}
	return &receivedFile, nil
}

func GetReceivedFilesByMediaGroupID(mediaGroupID string) ([]model.ReceivedFile, error) {
	var receivedFiles []model.ReceivedFile
	err := db.Where("media_group_id = ?", mediaGroupID).Find(&receivedFiles).Error
	if err != nil {
		return nil, err
	}
	return receivedFiles, nil
}

func UpdateReceivedFile(receivedFile *model.ReceivedFile) error {
	return db.Save(receivedFile).Error
}

func DeleteReceivedFileByFileID(fileID string) error {
	return db.Where("file_id = ?", fileID).Delete(&model.ReceivedFile{}).Error
}

func DeleteReceivedFileByFileUniqueID(fileUniqueID string) error {
	return db.Where("file_unique_id = ?", fileUniqueID).Delete(&model.ReceivedFile{}).Error
}
