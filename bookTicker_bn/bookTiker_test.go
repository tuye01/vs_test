package bookticker_bn

import (
	"log"
	"testing"
	"time"
)

func TestBookTicker(t *testing.T) {
	go connectToBinance()
	go sendHeartbeat() // 启动心跳检测
	// 运行 2 小时
	time.Sleep(2 * time.Hour)
	log.Println("🛑 运行 2 小时，自动退出")
}
