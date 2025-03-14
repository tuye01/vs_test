package bookticker_bn

import (
	"fmt"
	"log"
	"testing"
	"time"
)

func TestBookTicker(t *testing.T) {
	fmt.Println("🚀 启动程序")
	go connectToBinance()
	go sendHeartbeat() // 启动心跳检测
	// 运行 2 小时
	time.Sleep(2 * time.Hour)
	log.Println("🛑 运行 2 小时，自动退出")
}
