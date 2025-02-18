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

func GetStorageByID(id uint) (*types.StorageModel, error) {
	var storageModel types.StorageModel
	err := db.Preload("Users").First(&storageModel, id).Error
	return &storageModel, err
}

func CreateStorage(model *types.StorageModel) error {
	if model.Name == "" {
		model.Name = fmt.Sprintf("%s_%d", model.Type, model.ID)
	}
	return db.Create(model).Error
}
