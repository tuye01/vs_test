package bookticker_bn

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Binance WebSocket 地址
const binanceWSURL = "wss://stream.binance.com:9443/ws/btcusdt@depth"

// 订单簿数据结构
type DepthUpdate struct {
	EventType string     `json:"e"`
	EventTime int64      `json:"E"`
	Symbol    string     `json:"s"`
	Bids      [][]string `json:"b"`
	Asks      [][]string `json:"a"`
}

// 内存存储订单簿
var orderBook struct {
	sync.Mutex
	Bids map[string]string
	Asks map[string]string
}

// WebSocket 连接
var conn *websocket.Conn

// 连接 Binance WebSocket
func connectToBinance() {
	for {
		log.Println("🔗 连接 Binance WebSocket:", binanceWSURL)

		var err error
		conn, _, err = websocket.DefaultDialer.Dial(binanceWSURL, nil)
		if err != nil {
			log.Println("❌ 连接失败，5 秒后重试:", err)
			time.Sleep(5 * time.Second)
			continue
		}
		log.Println("✅ 成功连接 Binance WebSocket")

		// 初始化订单簿
		orderBook.Bids = make(map[string]string)
		orderBook.Asks = make(map[string]string)

		// 监听 WebSocket 数据
		readMessages()
		// **⬇️ 只有 `readMessages()` 失败才会执行到这里**
		log.Println("🔄 WebSocket 连接丢失，5 秒后尝试重连...")
		time.Sleep(5 * time.Second) // 避免短时间内频繁重连
	}
}

// 监听 WebSocket 消息
func readMessages() {
	defer conn.Close()

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("⚠️ WebSocket 读取失败:", err)
			break
		}
		if messageType == websocket.PingMessage {
			log.Println("❤️ 收到心跳消息")
			err := conn.WriteMessage(websocket.PongMessage, message)
			if err != nil {
				log.Println("❌ 回复心跳消息失败:", err)
			}
			continue
		}
		// 解析深度数据
		var depth DepthUpdate
		if err := json.Unmarshal(message, &depth); err != nil {
			log.Println("❌ JSON 解析失败:", err)
			continue
		}

		// 更新本地订单簿
		updateOrderBook(depth)

		// 打印格式化订单簿
		printOrderBook()
	}
}

// 更新订单簿
func updateOrderBook(depth DepthUpdate) {
	orderBook.Lock()
	defer orderBook.Unlock()

	for _, bid := range depth.Bids {
		price, quantity := bid[0], bid[1]
		if quantity == "0.00000000" {
			delete(orderBook.Bids, price)
		} else {
			orderBook.Bids[price] = quantity
		}
	}

	for _, ask := range depth.Asks {
		price, quantity := ask[0], ask[1]
		if quantity == "0.00000000" {
			delete(orderBook.Asks, price)
		} else {
			orderBook.Asks[price] = quantity
		}
	}
}

// 获取最优买价 & 卖价
func getTopOrders(m map[string]string, top int, ascending bool) []string {
	orderBook.Lock()
	defer orderBook.Unlock()

	var priceList []float64
	for price := range m {
		priceFloat, err := strconv.ParseFloat(price, 64) // 转换为 float64
		if err != nil {
			log.Println("❌ 价格转换失败:", price, err)
			continue
		}
		priceList = append(priceList, priceFloat)
	}

	// 排序
	sort.Slice(priceList, func(i, j int) bool {
		if ascending {
			return priceList[i] < priceList[j] // 升序 (卖单)
		}
		return priceList[i] > priceList[j] // 降序 (买单)
	})

	var result []string
	for i, price := range priceList {
		if i >= top {
			break
		}
		result = append(result, fmt.Sprintf("%.8f: %s", price, m[fmt.Sprintf("%.8f", price)]))
	}
	return result
}

// 格式化打印订单簿
func printOrderBook() {
	topAsks := getTopOrders(orderBook.Asks, 5, true)  // 最优 5 档卖单
	topBids := getTopOrders(orderBook.Bids, 5, false) // 最优 5 档买单

	fmt.Println("\n📊 最新 Binance BTC/USDT 订单簿 (Top 5)")
	fmt.Println("+------------------------+")
	fmt.Println("|      卖单 (ASKS)       |")
	fmt.Println("+------------------------+")
	for _, ask := range topAsks {
		fmt.Println("|", ask)
	}
	fmt.Println("+------------------------+")
	fmt.Println("|      买单 (BIDS)       |")
	fmt.Println("+------------------------+")
	for _, bid := range topBids {
		fmt.Println("|", bid)
	}
	fmt.Println("+------------------------+")
}

func sendHeartbeat() {
	ticker := time.NewTicker(20 * time.Second) // 每 30 秒发送一次心跳
	defer ticker.Stop()
	//lint:ignore S1000 我需要无限监听多个通道
	for {
		select {
		case <-ticker.C:
			err := conn.WriteMessage(websocket.PingMessage, nil) // 发送 Ping 消息
			if err != nil {
				log.Println("⚠️ 心跳消息发送失败:", err)
				return // 如果发送失败，可能说明连接已断开
			}
			log.Println("❤️ 发送心跳消息")
		}
	}
}
