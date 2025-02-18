package dao

import (
	"fmt"

	"github.com/krau/SaveAny-Bot/types"
)

func GetActiveStorages() ([]types.StorageModel, error) {
	var storageModels []types.StorageModel
	err := db.Where("active = ?", true).Find(&storageModels).Error
	return storageModels, err
}

func GetStorageByHash(hash string) (*types.StorageModel, error) {
	var storageModel types.StorageModel
	err := db.Where("hash = ?", hash).First(&storageModel).Error
	return &storageModel, err
}

func GetStorageByID(id uint) (*types.StorageModel, error) {
	var storageModel types.StorageModel
	err := db.Preload("Users").First(&storageModel, id).Error
	return &storageModel, err
}

func CreateStorage(model *types.StorageModel) (uint, error) {
	if model.Hash == "" {
		model.Hash = model.GenHash()
	}
	getModel, err := GetStorageByHash(model.Hash)
	if err == nil {
		return getModel.ID, nil
	}
	tx := db.Create(model)
	if tx.Error != nil {
		return 0, tx.Error
	}
	if model.Name == "" {
		model.Name = fmt.Sprintf("%s - %d", model.Type, model.ID)
		tx = db.Save(model)
		if tx.Error != nil {
			return 0, tx.Error
		}
	}
	return model.ID, nil
}
