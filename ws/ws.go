package ws

// 文件说明：处理 WebSocket 连接、消息接收和事件分发。
// 主要职责：把外部歌词、封面和播放进度等实时数据送入程序内部。

import (
	"EbitenLyrics/evbus"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/cmplx"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/disintegration/imaging"
	"github.com/gorilla/websocket"
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
type lowFreqAnalyzer struct {
	sampleRate float64
	emaValue   float64
	peakLevel  float64 // 用于动态调整增益的峰值
	formatHint string
}

func newLowFreqAnalyzer(sampleRate float64) *lowFreqAnalyzer {
	return &lowFreqAnalyzer{
		sampleRate: sampleRate,
		emaValue:   0,
		peakLevel:  0.2, // 初始预设一个较小的峰值
	}
}

func (a *lowFreqAnalyzer) AnalyzePCM(data []byte) (float64, bool) {
	if a == nil {
		return 0, false
	}
	samples, format, ok := decodePCMToMono(data)
	if !ok || len(samples) < 128 {
		return 0, false
	}
	if a.formatHint == "" {
		a.formatHint = format
		log.Printf("audio analyzer format=%s samples=%d", format, len(samples))
	}

	// 1. 确定 FFT 长度
	sampleCount := len(samples)
	nfft := 1
	for nfft*2 <= sampleCount && nfft < 2048 {
		nfft *= 2
	}
	if nfft < 256 {
		return 0, false
	}

	// 2. 预处理：去直流分量 + 汉宁窗
	fftBuf := make([]complex128, nfft)
	var mean float64
	for i := 0; i < nfft; i++ {
		mean += samples[i]
	}
	mean /= float64(nfft)

	denom := float64(nfft - 1)
	for i := 0; i < nfft; i++ {
		// Hann Window
		w := 0.5 * (1.0 - math.Cos(2.0*math.Pi*float64(i)/denom))
		fftBuf[i] = complex((samples[i]-mean)*w, 0)
	}

	// 3. 执行 FFT
	fftInPlace(fftBuf)

	// 4. 计算目标频段能量 (20Hz - 300Hz)
	freqRes := a.sampleRate / float64(nfft)
	minBin := int(math.Ceil(20.0 / freqRes))
	maxBin := int(math.Floor(300.0 / freqRes))
	if minBin < 1 {
		minBin = 1
	}
	nyquist := nfft / 2
	if maxBin > nyquist {
		maxBin = nyquist
	}

	var lowEndEnergy float64
	for k := minBin; k <= maxBin; k++ {
		mag := cmplx.Abs(fftBuf[k])
		// 使用幅度的平方作为能量
		lowEndEnergy += mag * mag
	}

	// 归一化能量值 (取均值后开方，得到类似 RMS 的指标)
	currentLevel := math.Sqrt(lowEndEnergy / float64(maxBin-minBin+1))

	// 5. 动态范围压缩与归一化
	// 更新历史峰值（缓慢下降，确保环境适应性）
	a.peakLevel = math.Max(a.peakLevel*0.999, currentLevel)
	if a.peakLevel < 0.01 {
		a.peakLevel = 0.01
	}

	// 将当前能量映射到 0-1 范围
	// 使用 linear 比例，如果想要更夸张的效果可以使用 math.Log10 映射
	rawNorm := math.Log10(1 + 9*currentLevel/a.peakLevel)
	if rawNorm > 1.0 {
		rawNorm = 1.0
	}

	// 6. 平滑滤波 (Attack 快速上升, Decay 缓慢下降)
	var sensitivity float64
	if rawNorm > a.emaValue {
		sensitivity = 0.5 // 反应灵敏：上升系数
	} else {
		sensitivity = 0.2 // 丝滑回落：下降系数
	}
	a.emaValue = a.emaValue*(1-sensitivity) + rawNorm*sensitivity

	// 最终输出强制限制在 0-1
	finalValue := math.Max(0, math.Min(1, a.emaValue))

	return finalValue, true
}

func decodePCMToMono(data []byte) ([]float64, string, bool) {
	if len(data) >= 4*64 && len(data)%4 == 0 {
		signed16 := likelySignedPCM16(data)
		frames := len(data) / 4
		out := make([]float64, frames)
		if signed16 {
			for i := 0; i < frames; i++ {
				base := i * 4
				l := float64(int16(binary.LittleEndian.Uint16(data[base : base+2])))
				r := float64(int16(binary.LittleEndian.Uint16(data[base+2 : base+4])))
				out[i] = (l + r) * (0.5 / 32768.0)
			}
			return out, "i16-stereo-le", true
		}
		for i := 0; i < frames; i++ {
			base := i * 4
			l := float64(binary.LittleEndian.Uint16(data[base:base+2])) - 32768.0
			r := float64(binary.LittleEndian.Uint16(data[base+2:base+4])) - 32768.0
			out[i] = (l + r) * (0.5 / 32768.0)
		}
		return out, "u16-stereo-le", true
	}

	if len(data) >= 2*64 && len(data)%2 == 0 {
		signed16 := likelySignedPCM16(data)
		sampleCount := len(data) / 2
		out := make([]float64, sampleCount)
		if signed16 {
			for i := 0; i < sampleCount; i++ {
				base := i * 2
				out[i] = float64(int16(binary.LittleEndian.Uint16(data[base:base+2]))) / 32768.0
			}
			return out, "i16-mono-le", true
		}
		for i := 0; i < sampleCount; i++ {
			base := i * 2
			out[i] = (float64(binary.LittleEndian.Uint16(data[base:base+2])) - 32768.0) / 32768.0
		}
		return out, "u16-mono-le", true
	}

	// Fallback for non-standard custom sender implementation.
	if len(data) >= 8*64 && len(data)%8 == 0 {
		sampleCount := len(data) / 8
		out := make([]float64, sampleCount)
		const invMaxI64 = 1.0 / 9223372036854775807.0
		for i := 0; i < sampleCount; i++ {
			base := i * 8
			out[i] = float64(int64(binary.LittleEndian.Uint64(data[base:base+8]))) * invMaxI64
		}
		return out, "i64-mono-le", true
	}

	return nil, "", false
}

func likelySignedPCM16(data []byte) bool {
	sampleCount := len(data) / 2
	if sampleCount < 4 {
		return false
	}
	const jumpThreshold = 30000
	jumps := 0
	prev := binary.LittleEndian.Uint16(data[0:2])
	for i := 1; i < sampleCount; i++ {
		base := i * 2
		cur := binary.LittleEndian.Uint16(data[base : base+2])
		diff := int(cur) - int(prev)
		if diff < 0 {
			diff = -diff
		}
		if diff > jumpThreshold {
			jumps++
		}
		prev = cur
	}
	return float64(jumps)/float64(sampleCount-1) > 0.02
}

func fftInPlace(a []complex128) {
	n := len(a)
	if n <= 1 {
		return
	}
	j := 0
	for i := 1; i < n; i++ {
		bit := n >> 1
		for ; (j & bit) != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			a[i], a[j] = a[j], a[i]
		}
	}

	for step := 2; step <= n; step <<= 1 {
		half := step >> 1
		ang := -2.0 * math.Pi / float64(step)
		wStep := cmplx.Exp(complex(0, ang))
		for i := 0; i < n; i += step {
			w := complex(1.0, 0)
			for k := 0; k < half; k++ {
				u := a[i+k]
				v := a[i+k+half] * w
				a[i+k] = u + v
				a[i+k+half] = u - v
				w *= wStep
			}
		}
	}
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
	audioAnalyzer := newLowFreqAnalyzer(48000)
	server := NewAMLLWebSocketServer()

	// 启动服务器
	server.Reopen("0.0.0.0:11445", messageChannel)

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
				log.Printf("MAIN [V2-Command]: %s", p.Command)

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
					lines, ok := p.Data["lines"].([]interface{})
					if !ok {
						log.Printf("MAIN: lines has unexpected type %T", p.Data["lines"])
						break
					}
					evbus.Bus.Publish("ws:setLyric", lines)
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
						if v, ok := p.Data["progress"].(int); ok {
							progress = float64(v)
						} else {
							log.Printf("MAIN: progress has unexpected type %T", p.Data["progress"])
							break
						}
					}
					evbus.Bus.Publish("ws:progress", progress)
				case "volume":
					log.Println(p.Data)
				case "setCover":
					log.Println(p.Data)
				case "setFontConfig", "setFont":
					evbus.Bus.Publish("ws:fontConfig", p.Data)
				}

				// --- 情况 B: 二进制封面 (通常是直接的图片数据) ---
			case V2BinaryMessage:
				//log.Printf("MAIN [Binary]: 类型=%s, 大小=%d bytes", p.Type, len(p.Data))

				switch p.Type {
				case "SetCoverData":
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

					evbus.Bus.Publish("ws:cover", blurred)
				case "OnAudioData":
					if lowFreqVolume, ok := audioAnalyzer.AnalyzePCM(p.Data); ok {
						evbus.Bus.Publish("ws:lowFreqVolume", lowFreqVolume)
						//log.Println("   >>> 频谱:", lowFreqVolume)
					}
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
