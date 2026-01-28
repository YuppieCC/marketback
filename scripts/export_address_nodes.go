package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"marketcontrol/internal/models"
	"marketcontrol/pkg/config"
)

func main() {
	// 解析命令行参数
	mapID := flag.Uint("map-id", 0, "地址图谱ID")
	nodeType := flag.String("node-type", "", "节点类型")
	fileName := flag.String("file-name", "", "输出文件名")
	flag.Parse()

	// 验证参数
	if *mapID == 0 {
		log.Fatal("请提供有效的地址图谱ID")
	}
	if *nodeType == "" {
		log.Fatal("请提供节点类型")
	}
	if *fileName == "" {
		log.Fatal("请提供输出文件名")
	}

	// 初始化数据库连接
	config.InitDB()

	// 查询符合条件的节点
	var nodes []models.AddressNode
	if err := config.DB.Where("map_id = ? AND node_type = ?", *mapID, *nodeType).Find(&nodes).Error; err != nil {
		log.Fatalf("查询节点失败: %v", err)
	}

	if len(nodes) == 0 {
		log.Fatal("未找到符合条件的节点")
	}

	// 创建结果映射
	result := make(map[string]string)

	// 遍历节点，查询对应的地址管理信息
	for _, node := range nodes {
		var addressManage models.AddressManage
		if err := config.DB.Where("address = ?", node.NodeValue).First(&addressManage).Error; err != nil {
			log.Printf("警告: 未找到地址 %s 的管理信息", node.NodeValue)
			continue
		}
		result[addressManage.Address] = addressManage.PrivateKey
	}

	if len(result) == 0 {
		log.Fatal("未找到任何匹配的地址管理信息")
	}

	// 确保输出目录存在
	outputDir := "output/address_keystore"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("创建输出目录失败: %v", err)
	}

	// 构建输出文件路径
	outputPath := filepath.Join(outputDir, fmt.Sprintf("%s.json", *fileName))

	// 将结果写入文件
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("JSON编码失败: %v", err)
	}

	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		log.Fatalf("写入文件失败: %v", err)
	}

	log.Printf("成功导出 %d 个地址到文件: %s", len(result), outputPath)
}