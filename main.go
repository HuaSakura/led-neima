package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"syscall"
	"unsafe"

	"github.com/gorilla/websocket"
)

// ---------------- DLL 动态加载 ----------------

var (
	dll                     *syscall.DLL
	procLV_InitLed          *syscall.Proc
	procLV_GetError         *syscall.Proc
	procLV_TestOnline       *syscall.Proc
	procLV_RefreshNeiMaArea *syscall.Proc
	onceDLLLoad             sync.Once
	dllPath                 = "./DLL/32/lv_led_32.dll" // 根据实际路径修改
)

type LedMessage struct {
	IP  string `json:"ip"`
	Msg string `json:"msg"`
}

type LedResponse struct {
	Online    bool   `json:"online"`
	Sent      bool   `json:"sent"`
	Error     string `json:"error"`
	ErrorCode int    `json:"error_code,omitempty"`
}

type COMMUNICATIONINFO struct {
	LEDType   int32
	SendType  int32
	IpStr     [64]uint16 // wchar_t[64]
	Commport  int32
	Baud      int32
	LedNumber int32
	OutputDir [260]uint16
}

var sendSem = make(chan struct{}, 1)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

//加载DLL
func loadDLL() {
	var err error
	dll, err = syscall.LoadDLL(dllPath)
	if err != nil {
		panic(fmt.Sprintf("Load DLL error: %v", err))
	}
	procLV_InitLed, _ = dll.FindProc("LV_InitLed")
	procLV_GetError, _ = dll.FindProc("LV_GetError")
	procLV_TestOnline, _ = dll.FindProc("LV_TestOnline")
	procLV_RefreshNeiMaArea, _ = dll.FindProc("LV_RefreshNeiMaArea")
}

// ---------------- SDK 调用封装 ----------------
func getError(code int32) string {
	buf := make([]uint16, 512)
	r1, _, _ := procLV_GetError.Call(
		uintptr(code),
		uintptr(len(buf)),
		uintptr(unsafe.Pointer(&buf[0])),
	)
	if r1 != 0 {
		return "unknown error"
	}
	for i, b := range buf {
		if b == 0 {
			return syscall.UTF16ToString(buf[:i])
		}
	}
	return syscall.UTF16ToString(buf)
}

// 注意：这里的 sendToLed 不再自己做互斥（不再使用 ledMutex）。
// 调用方需保证同步（本代码通过 sendSem 保证同一时间只有一个调用）。
func sendToLed(ip, msg string) LedResponse {

	var comm COMMUNICATIONINFO
	comm.LEDType = 0
	comm.SendType = 0 // TCP 固定 IP

	copy(comm.IpStr[:], syscall.StringToUTF16(ip))

	// 1) 测试在线，不通则直接返回，不调用 LV_RefreshNeiMaArea
	retTest, _, _ := procLV_TestOnline.Call(uintptr(unsafe.Pointer(&comm)))

	if int32(retTest) != 0 {
		return LedResponse{
			Online:    false,
			Sent:      false,
			Error:     getError(int32(retTest)),
			ErrorCode: int(retTest),
		}
	}

	log.Printf("Go recv msg: %q", msg)

	// 2) 发送内容
	msgUTF16 := syscall.StringToUTF16(msg)

	retSend, _, _ := procLV_RefreshNeiMaArea.Call(
		uintptr(unsafe.Pointer(&comm)),
		uintptr(unsafe.Pointer(&msgUTF16[0])),
	)

	if int32(retSend) != 0 {
		return LedResponse{
			Online:    true,
			Sent:      false,
			Error:     getError(int32(retSend)),
			ErrorCode: int(retSend),
		}
	}

	// 成功
	return LedResponse{
		Online: true,
		Sent:   true,
		Error:  "",
	}
}

// ---------------- WebSocket 逻辑 ----------------
func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		var msg LedMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			// 无效消息直接丢弃
			conn.WriteJSON(LedResponse{Online: false, Sent: false, Error: "invalid json"})
			continue
		}

		// 非阻塞尝试获取发送许可（信号量）
		select {
		case sendSem <- struct{}{}:
			// 成功获得许可：在 goroutine 中发送，结束时释放许可
			go func(m LedMessage) {
				// 确保释放信号量（即使 panic 也释放）
				defer func() {
					<-sendSem
				}()

				result := sendToLed(m.IP, m.Msg)
				if err := conn.WriteJSON(result); err != nil {
					// 写回客户端失败，记录日志即可
					log.Println("WriteJSON error:", err)
				}
			}(msg)

		default:
			// sendSem 已满，表示正在发送；直接丢弃（不排队）
			log.Printf("Message dropped (sendToLed busy): %q", msg.Msg)
		}
	}
}

// ---------------- main ----------------

func main() {
	fmt.Println("Starting WebSocket LED server on :18090")
	onceDLLLoad.Do(loadDLL)

	procLV_InitLed.Call(0, 0)

	http.HandleFunc("/neima", wsHandler)
	log.Fatal(http.ListenAndServe("0.0.0.0:18090", nil))
}
