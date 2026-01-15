package ws

import (
	"EbitenLyrics/evbus"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/disintegration/imaging"
	"github.com/gorilla/websocket"
	"github.com/hajimehoshi/ebiten/v2"
)

// ===========================
// 1. 数据结构定义
// ===========================

// --- V2 协议相关 ---

type V2PayloadType string

const (
	TypeInitialize V2PayloadType = "initialize"
	TypePing       V2PayloadType = "ping"
	TypePong       V2PayloadType = "pong"
	TypeCommand    V2PayloadType = "command"
	TypeState      V2PayloadType = "state"
)

// GenericV2Payload 用于第一次解析，读取 type 字段
type GenericV2Payload struct {
	Type  V2PayloadType   `json:"type"`
	Value json.RawMessage `json:"value"`
}

// Command 结构体调整
type Command struct {
	Command string                 `json:"command"` // 指令名称，如 "setVolume"
	Data    map[string]interface{} `json:"-"`       // 存储完整的数据字典 (手动填充)
}

// StateUpdate 结构体调整
type StateUpdate struct {
	Update string                 `json:"update"` // 更新类型，如 "setMusic"
	Data   map[string]interface{} `json:"-"`      // 存储完整的数据字典 (手动填充)
}

// MusicInfo 示例具体数据结构
type MusicInfo struct {
	MusicId   string `json:"music_id"`
	MusicName string `json:"music_name"`
	// ... 其他字段

}

type LyricContent struct {
	Ttml string `json:"ttml"`
}

// V2BinaryHeader 对应 Rust 的二进制头部
type V2BinaryHeader struct {
	Magic uint16
	Size  uint32
}

// V2BinaryMessage 用于在 Channel 中传递二进制消息
type V2BinaryMessage struct {
	Type string
	Data []byte
}

// --- V1 协议相关 ---

// V1Body 模拟 V1 二进制结构 (需要根据实际协议补全)
type V1Body struct {
	ID  uint32
	Raw []byte
}

// ===========================
// 2. 服务器框架定义
// ===========================

type ProtocolType int

const (
	Unknown ProtocolType = iota
	BinaryV1
	HybridV2
)

type ProtocolPayload interface{} // 消息通道传递的通用接口
type MessageChannel chan ProtocolPayload

type ConnectionInfo struct {
	Conn     *websocket.Conn
	Protocol ProtocolType
}

type AMLLWebSocketServer struct {
	stopChan    chan struct{}
	connections map[string]ConnectionInfo
	mu          sync.RWMutex
}

func NewAMLLWebSocketServer() *AMLLWebSocketServer {
	return &AMLLWebSocketServer{
		stopChan:    make(chan struct{}),
		connections: make(map[string]ConnectionInfo),
	}
}

func (s *AMLLWebSocketServer) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopChan != nil {
		close(s.stopChan)
		s.stopChan = nil
	}
	for _, info := range s.connections {
		info.Conn.Close()
	}
	s.connections = make(map[string]ConnectionInfo)
}

func (s *AMLLWebSocketServer) Reopen(addr string, msgChan MessageChannel) {
	s.Close()
	s.stopChan = make(chan struct{})
	stopChan := s.stopChan

	go func() {
		log.Printf("INFO: WebSocket 服务器监听中: %s", addr)
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			s.acceptConn(w, r, msgChan)
		})

		server := &http.Server{Addr: addr, Handler: mux}

		go func() {
			<-stopChan
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			server.Shutdown(ctx)
		}()

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("ERROR: 服务器启动失败: %v", err)
		}
	}()
}

// ===========================
// 3. 连接处理逻辑 (核心修改)
// ===========================

func (s *AMLLWebSocketServer) acceptConn(w http.ResponseWriter, r *http.Request, msgChan MessageChannel) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	addr := conn.RemoteAddr().String()
	log.Printf("INFO: 客户端连接: %s", addr)

	// --- 1. 握手与协议识别 ---
	var protocolType ProtocolType

	// 读取第一条消息
	msgType, reader, err := conn.NextReader()
	if err != nil {
		return
	}
	data, _ := io.ReadAll(reader)

	switch msgType {
	case websocket.TextMessage:
		// 修复点：使用 GenericV2Payload 解析，而不是 V2PayloadType
		var payload GenericV2Payload
		if err := json.Unmarshal(data, &payload); err == nil {
			if payload.Type == TypeInitialize {
				log.Println("INFO: 协议识别 -> HybridV2")
				protocolType = HybridV2
			} else {
				log.Println("WARN: V2 握手失败，非 Initialize 消息")
				return
			}
		} else {
			log.Println("WARN: JSON 解析失败，断开")
			return
		}
	case websocket.BinaryMessage:
		log.Println("INFO: 协议识别 -> BinaryV1")
		protocolType = BinaryV1
		// 处理第一条 V1 消息
		s.processV1Message(msgType, data, msgChan)
	}

	if protocolType != Unknown {
		s.mu.Lock()
		s.connections[addr] = ConnectionInfo{Conn: conn, Protocol: protocolType}
		s.mu.Unlock()
	} else {
		return
	}

	// --- 2. 消息循环 ---
	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("INFO: 客户端 %s 断开", addr)
			break
		}

		s.mu.RLock()
		info, ok := s.connections[addr]
		s.mu.RUnlock()
		if !ok {
			break
		}

		var processErr error
		switch info.Protocol {
		case HybridV2:
			processErr = s.processV2Message(msgType, msg, msgChan)
		case BinaryV1:
			processErr = s.processV1Message(msgType, msg, msgChan)
		}

		if processErr != nil {
			log.Printf("ERROR: 消息处理错误: %v", processErr)
			break
		}
	}

	s.mu.Lock()
	delete(s.connections, addr)
	s.mu.Unlock()
}

func (s *AMLLWebSocketServer) processV1Message(msgType int, data []byte, channel MessageChannel) error {
	if msgType == websocket.BinaryMessage {
		// 模拟解析 V1
		v1Body := V1Body{ID: 1, Raw: data}
		channel <- v1Body
	}
	return nil
}

func (s *AMLLWebSocketServer) processV2Message(msgType int, data []byte, channel MessageChannel) error {
	if msgType == websocket.TextMessage {
		var generic GenericV2Payload
		if err := json.Unmarshal(data, &generic); err != nil {
			return err
		}

		switch generic.Type {
		case TypeInitialize, TypePing, TypePong:
			channel <- generic.Type

		case TypeCommand:
			var rawMap map[string]interface{}
			if err := json.Unmarshal(generic.Value, &rawMap); err != nil {
				return err
			}
			cmdStr, _ := rawMap["command"].(string)
			channel <- Command{Command: cmdStr, Data: rawMap}

		case TypeState:
			var rawMap map[string]interface{}
			if err := json.Unmarshal(generic.Value, &rawMap); err != nil {
				log.Printf("ERROR: 解析 State Value 失败: %v", err)
				return err
			}

			// 关键：确保 update 字段存在
			updateStr, ok := rawMap["update"].(string)
			if !ok {
				log.Printf("WARN: State 消息缺少 update 字段: %+v", rawMap)
				return nil
			}

			channel <- StateUpdate{
				Update: updateStr,
				Data:   rawMap,
			}
		}

	} else if msgType == websocket.BinaryMessage {
		r := bytes.NewReader(data)
		var header V2BinaryHeader
		if err := binary.Read(r, binary.LittleEndian, &header); err != nil {
			return err
		}
		if uint32(r.Len()) < header.Size {
			return fmt.Errorf("长度不足")
		}
		dataBytes := make([]byte, header.Size)
		r.Read(dataBytes)

		// 这里对应 Rust 的 Magic Number
		msgTypeStr := "Unknown"
		if header.Magic == 0 {
			msgTypeStr = "OnAudioData"
		}
		if header.Magic == 1 {
			msgTypeStr = "SetCoverData" // <--- 封面通常走这里
		}

		channel <- V2BinaryMessage{Type: msgTypeStr, Data: dataBytes}
	}
	return nil
}

// ===========================
// 4. 主程序入口 (initws)
// ===========================

func Initws() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)

	messageChannel := make(MessageChannel, 100) // 带缓冲，防止阻塞
	server := NewAMLLWebSocketServer()

	// 启动服务器
	server.Reopen("127.0.0.1:11445", messageChannel)

	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("MAIN: 等待消息...")

	for {
		select {
		case payload := <-messageChannel:
			// 核心修复：这里必须根据 processV2Message 发送的具体类型来断言
			switch p := payload.(type) {

			// 1. 处理简单的 V2 信号 (Initialize, Ping, Pong)
			case V2PayloadType:
				log.Printf("MAIN [V2-Signal]: %s", p)

			// 2. 处理 V2 指令 (Command)
			case Command:
				log.Printf("MAIN [V2-Command]:", p.Command)

			// 3. 处理 V2 状态更新 (StateUpdate)
			case StateUpdate:
				//log.Printf("MAIN [V2-State]: %s", p.Update)
				// 示例：如果是 SetMusic，尝试二次解析 Value
				switch p.Update {
				case "setMusic":
					log.Println(p.Data)
					evbus.Bus.Publish("ws:setMusic", p.Data)
				case "setLyric":
					//log.Println(p.Data)
					/*d, err := ParseLyricsFromMap(p.Data["lines"].([]interface{}))
					if err != nil {
						log.Println(err)
					} else {
						log.Println(d)
						game.mu.Lock()

						for _, ele := range game.Elements {

							ele.CancelAllAnimate()
							ele.DisposeAll()
							if ele.TranslatedEle != nil {
								ele.TranslatedEle.Image.Dispose()
								ele.TranslatedEle = nil
							}
							for _, bg := range ele.BGs {
								bg.CancelAllAnimate()
								bg.DisposeAll()
								if bg.TranslatedEle != nil {
									bg.TranslatedEle.Image.Dispose()
									bg.TranslatedEle = nil
								}
							}

						}
						game.Elements = nil
						game.nowLyrics = d

						game.Elements = initGamesLyrics(game.nowLyrics)
						Scroll([]int{0}, game, 0)
						game.mu.Unlock()
					}*/
					evbus.Bus.Publish("ws:setLyric", p.Data["lines"].([]interface{}))
				case "progress":
					//log.Println(p.Data)
					// float64
					/*progress, ok := p.Data["progress"].(float64)
					if !ok {
						log.Println("MAIN: progress 不是 float64")
						break
					}
					game.mu.Lock()
					game.progress = time.Duration(progress) * time.Millisecond
					game.mu.Unlock()*/

					progress, ok := p.Data["progress"].(float64)
					if !ok {
						log.Println("MAIN: progress 不是 float64")
						break
					}
					evbus.Bus.Publish("ws:progress", progress)
				case "volume":
					log.Println(p.Data)
				case "setCover":
					log.Println(p.Data)
				}

				// --- 情况 B: 二进制封面 (通常是直接的图片数据) ---
			case V2BinaryMessage:
				//log.Printf("MAIN [Binary]: 类型=%s, 大小=%d bytes", p.Type, len(p.Data))

				if p.Type == "SetCoverData" {
					log.Println("   >>> 收到二进制封面数据 (SetCoverData)!")
					// 这里 p.Data 就是图片的 byte 数组 (png/jpg)
					// 你可以将 p.Data 保存为文件，或者发送给前端
					/*log.Println("   >>> 尝试解码...", len(p.Data))
					img, err := jpeg.Decode(bytes.NewReader(p.Data))
					if err != nil {
						log.Println("   >>> 解码失败:", err)
						continue
					}*/
					img, err := imaging.Decode(bytes.NewReader(p.Data))
					if err != nil {
						log.Println("   >>> 解码失败:", err)
						continue
					}
					blurred := imaging.Blur(img, 40)
					// 亮度0.7
					blurred = imaging.AdjustBrightness(blurred, -30)
					//对比度
					blurred = imaging.AdjustContrast(blurred, -30)
					blurred = imaging.AdjustSaturation(blurred, 10)

					evbus.Bus.Publish("ws:cover", ebiten.NewImageFromImage(blurred))

				}

			// 5. 处理 V1 二进制消息
			case V1Body:
				log.Printf("MAIN [V1-Binary]: ID=%d", p.ID)

			default:
				log.Printf("MAIN [Unknown]: 收到未知类型 %T", payload)
			}

		case <-termChan:
			log.Println("MAIN: 正在关闭...")
			server.Close()
			os.Exit(0)
		}
	}
}
