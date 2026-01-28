package integration

import (
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	// 等待服务启动
	time.Sleep(5 * time.Second)

	// 运行测试
	code := m.Run()

	// 清理测试数据
	cleanup()

	os.Exit(code)
}

func cleanup() {
	// 这里可以添加清理测试数据的代码
	// 例如：删除测试过程中创建的配置等
} 