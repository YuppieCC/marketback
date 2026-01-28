package business

import (
	"marketcontrol/internal/models"
	dbconfig "marketcontrol/pkg/config"
)

// GetProjectAddresses 获取所有项目相关地址，包括 AddressManage 和启用的 ProjectExtraAddress
func GetProjectAddresses() ([]string, error) {
	var addresses []string
	
	// 获取 AddressManage 中的地址
	if err := dbconfig.DB.Model(&models.AddressManage{}).
		Pluck("address", &addresses).Error; err != nil {
		return nil, err
	}
	
	// 获取 ProjectExtraAddress 中 Enabled 为 true 的地址
	var extraAddresses []string
	if err := dbconfig.DB.Model(&models.ProjectExtraAddress{}).
		Where("enabled = ?", true).
		Pluck("address", &extraAddresses).Error; err != nil {
		return nil, err
	}
	
	// 合并地址列表
	addresses = append(addresses, extraAddresses...)
	
	return addresses, nil
} 